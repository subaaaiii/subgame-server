package controllers

import (
	"bnsp2/server/config"
	"bnsp2/server/database"
	"bnsp2/server/helpers"
	"bnsp2/server/models"
	"bnsp2/server/redis"
	"bnsp2/server/services"
	"bnsp2/server/structs"
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/snap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	StatusPending = "PENDING"
	StatusPaid    = "PAID"
	StatusFailed  = "FAILED"
	StatusExpired = "EXPIRED"
)

func CreateOrder(c *gin.Context) {
	var req structs.OrderRequest
	var order models.Order
	var product models.Product
	var user models.User

	userId := c.MustGet("user_id").(uint)

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, structs.ErrorResponse{
			Success: false,
			Message: "Validation Errors",
		})
		return
	}

	if req.Qty <= 0 {
		c.JSON(http.StatusBadRequest, structs.ErrorResponse{
			Success: false,
			Message: "Quantity must be greater than 0",
		})
		return
	}

	// Ambil customer
	if err := database.DB.First(&user, "id = ?", userId).Error; err != nil {
		c.JSON(http.StatusNotFound, structs.ErrorResponse{
			Success: false,
			Message: "Customer not found",
		})
		return
	}

	if err := database.DB.First(&product, "id = ?", req.ProductId).Error; err != nil {
		c.JSON(http.StatusNotFound, structs.ErrorResponse{
			Success: false,
			Message: "Product not found",
		})
		return
	}

	if product.UserId == userId {
		c.JSON(http.StatusForbidden, structs.ErrorResponse{
			Success: false,
			Message: "You are not allowed to buy your own product",
		})
		return
	}

	if product.Stock < req.Qty {
		c.JSON(http.StatusBadRequest, structs.ErrorResponse{
			Success: false,
			Message: "Product stock not enough",
		})
		return
	}

	order = models.Order{
		Id:        uuid.NewString(),
		Invoice:   fmt.Sprintf("INV-%d%d", time.Now().Unix(), rand.Intn(1000)),
		ProductId: req.ProductId,
		UserId:    userId,
		SellerId:  product.UserId,
		Qty:       req.Qty,
		Total:     product.Price * req.Qty,
		Status:    StatusPending,
	}

	if err := database.DB.Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, structs.ErrorResponse{
			Success: false,
			Message: "Failed to create order",
		})
		return
	}

	itemName := product.Title
	if len(itemName) > 50 {
		itemName = itemName[:47] + "..."
	}

	snapReq := &snap.Request{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:  order.Id,
			GrossAmt: int64(order.Total),
		},
		CustomerDetail: &midtrans.CustomerDetails{
			FName: user.Name,
			Email: user.Email,
		},
		Items: &[]midtrans.ItemDetails{
			{
				ID:    fmt.Sprintf("%d", product.Id),
				Price: int64(product.Price),
				Qty:   int32(req.Qty),
				Name:  itemName,
			},
		},
	}

	snapResp, midtransErr := config.SnapClient.CreateTransaction(snapReq)
	if midtransErr != nil {
		c.JSON(http.StatusInternalServerError, structs.ErrorResponse{
			Success: false,
			Message: "Failed to get payment token",
		})
		return
	}

	c.JSON(http.StatusOK, structs.SuccessResponse{
		Success: true,
		Message: "Order successfully created",
		Data: map[string]interface{}{
			"order":      order,
			"snap_token": snapResp.Token,
			"snap_url":   snapResp.RedirectURL,
		},
	})
}

var SendInvoiceEmail = helpers.SendInvoiceEmail

func PaymentCallback(c *gin.Context) {

	var notificationPayload map[string]interface{}
	if err := c.ShouldBindJSON(&notificationPayload); err != nil {
		c.JSON(http.StatusUnprocessableEntity, structs.ErrorResponse{Success: false, Message: "Validation Errors"})
		return
	}

	orderId, exists := notificationPayload["order_id"].(string)
	if !exists {
		c.JSON(http.StatusBadRequest, structs.ErrorResponse{Success: false, Message: "Order ID not found in payload"})
		return
	}

	transactionStatusResp, err := config.CoreAPIClient.CheckTransaction(orderId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, structs.ErrorResponse{Success: false, Message: "Failed to verify transaction"})
		return
	}

	var mappedStatus string
	if transactionStatusResp != nil {
		switch transactionStatusResp.TransactionStatus {
		case "capture", "settlement":
			mappedStatus = StatusPaid
		case "cancel", "deny":
			mappedStatus = StatusFailed
		case "expire":
			mappedStatus = StatusExpired
		case "pending":
			mappedStatus = StatusPending
		default:
			mappedStatus = StatusPending
		}
	}

	if mappedStatus == StatusPending {
		c.JSON(http.StatusOK, gin.H{"status": "ignored"})
		return
	}

	var order models.Order
	var orderLog models.OrderLog

	var shouldSendEmail bool
	var affectedProduct models.Product
	var shouldInvalidateProductCache bool

	dbErr := database.DB.Transaction(func(tx *gorm.DB) error {
		var product models.Product

		if err := tx.First(&order, "id = ?", orderId).Error; err != nil {
			return err
		}

		if order.Status != StatusPending {
			return nil
		}

		if mappedStatus == StatusPaid {
			shouldSendEmail = true

			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				First(&product, order.ProductId).Error; err != nil {
				return err
			}

			if product.Stock < order.Qty {
				return errors.New("stock not enough")
			}

			product.Stock -= order.Qty

			if product.Stock == 0 {
				product.Status = "sold"
			}

			if err := tx.Save(&product).Error; err != nil {
				return err
			}

			affectedProduct = product

			shouldInvalidateProductCache = true

		}

		if err := tx.Model(&order).Update("status", mappedStatus).Error; err != nil {
			return err
		}
		order.Status = mappedStatus

		orderLog.OrderId = orderId
		orderLog.Status = mappedStatus
		orderLog.Title = helpers.GenerateLogTitle(mappedStatus)

		if err := tx.Create(&orderLog).Error; err != nil {
			return err
		}

		return nil
	})

	if dbErr != nil {
		log.Printf("[Midtrans Webhook Error] OrderID: %s, Error: %v\n", orderId, dbErr)
		var errorMessage string
		switch {
		case errors.Is(dbErr, gorm.ErrRecordNotFound):
			errorMessage = "order not found"
		case dbErr.Error() == "stock not enough":
			errorMessage = "stock not enough"
		default:
			errorMessage = "Internal server error"
		}
		c.JSON(http.StatusOK, structs.ErrorResponse{Success: false, Message: errorMessage})
		return
	}

	if shouldInvalidateProductCache {
		ctx := c.Request.Context()

		redis.RedisClient.Del(
			ctx,
			fmt.Sprintf("product:%d", affectedProduct.Id),
		)
		redis.RedisClient.Del(ctx, "products_all_recent")

		helpers.DeleteByPattern(
			ctx,
			fmt.Sprintf("products_public:game:%d:*", affectedProduct.GameId),
		)

		helpers.DeleteByPattern(
			ctx,
			fmt.Sprintf("products:user:%d:*", affectedProduct.UserId),
		)
	}

	var user models.User
	database.DB.Select("name", "email").First(&user, order.UserId)

	var product models.Product
	database.DB.Select("title").First(&product, order.ProductId)

	if shouldSendEmail {
		go func(order models.Order) {
			buf, err := services.GenerateInvoiceBuffer(order, user.Name, product.Title)
			if err == nil {
				SendInvoiceEmail(user.Email, buf, order.Invoice)
			}
		}(order)
	}

	c.JSON(http.StatusOK, structs.SuccessResponse{
		Success: true,
		Message: "Callback successfully processed",
	})
}

func GetOrder(c *gin.Context) {
	id := c.Param("id")

	var order models.Order

	if err := database.DB.Preload("Product").Preload("Product.Game").Preload("User", func(db *gorm.DB) *gorm.DB {
		return db.Select("id", "name")
	}).Preload("Seller", func(db *gorm.DB) *gorm.DB {
		return db.Select("id", "name")
	}).Preload("OrderLog", func(db *gorm.DB) *gorm.DB {
		return db.Order("created_at desc")
	}).First(&order, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, structs.ErrorResponse{
			Success: false,
			Message: "order not found",
		})
		return

	}

	c.JSON(http.StatusOK, structs.SuccessResponse{
		Success: true,
		Message: "Get order success",
		Data:    order,
	})
}

func GetOrders(c *gin.Context) {
	var orders []models.Order
	userId := c.MustGet("user_id").(uint)

	query := database.DB.Model(&models.Order{})

	orderType := c.DefaultQuery("type", "sold")

	if orderType == "sold" {
		query = query.Where("seller_id = ?", userId)
	} else if orderType == "purchase" {
		query = query.Where("user_id = ?", userId)
	} else {
		c.JSON(http.StatusBadRequest, structs.ErrorResponse{
			Success: false,
			Message: "invalid type (use 'sold' or 'purchase')",
		})
		return
	}

	status := c.Query("status")
	q := c.Query("q")

	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	page, _ := strconv.Atoi(pageStr)
	limit, _ := strconv.Atoi(limitStr)

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	offset := (page - 1) * limit
	var total int64

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if q != "" {
		query = query.Where("LOWER(invoice) LIKE ?", "%"+strings.ToLower(q)+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, structs.ErrorResponse{
			Success: false,
			Message: "failed to count orders",
		})
		return
	}

	if err := query.Preload("Product").
		Order("created_at DESC").Limit(limit).
		Offset(offset).Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, structs.ErrorResponse{
			Success: false,
			Message: "failed to fetch orders",
		})
		return
	}
	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Orders retrieved successfully",
		"data":    orders,
		"meta": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

func UpdateStatusOrder(c *gin.Context) {

	var req structs.UpdateStatusRequest

	var order models.Order

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, structs.ErrorResponse{
			Success: false,
			Message: "Validation Errors",
		})
		return
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&order, "id = ?", req.OrderID).Error; err != nil {
			return err
		}

		order.Status = req.Status

		orderLog := models.OrderLog{
			OrderId: req.OrderID,
			Status:  req.Status,
			Title:   helpers.GenerateLogTitle(req.Status),
		}

		if err := tx.Save(&order).Error; err != nil {
			return err
		}

		if err := tx.Create(&orderLog).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {

		fmt.Printf("UpdateStatusOrder error: %v\n", err)
		c.JSON(http.StatusInternalServerError, structs.ErrorResponse{
			Success: false,
			Message: "Failed to update status",
		})

		return
	}

	c.JSON(http.StatusOK, structs.SuccessResponse{
		Success: true,
		Message: "Order status updated",
	})

}

func GetOrderBatch(c *gin.Context) {
	idsParam := c.Query("ids")
	if idsParam == "" {
		c.JSON(http.StatusBadRequest, structs.ErrorResponse{
			Success: false,
			Message: "ids is required",
		})
		return
	}

	ids := strings.Split(idsParam, ",")

	var orders []models.Order

	if err := database.DB.
		Preload("Product").
		Preload("Product.Game").
		Where("id IN ?", ids).
		Find(&orders).Error; err != nil {

		c.JSON(http.StatusInternalServerError, structs.ErrorResponse{
			Success: false,
			Message: "failed to fetch orders",
		})
		return
	}

	c.JSON(http.StatusOK, structs.SuccessResponse{
		Success: true,
		Message: "Get order batch success",
		Data:    orders,
	})
}
