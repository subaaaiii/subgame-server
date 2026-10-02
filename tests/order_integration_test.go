package tests_test

import (
	"bnsp2/server/controllers"
	"bnsp2/server/database"
	"bnsp2/server/models"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gopkg.in/h2non/gock.v1"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	router.Use(func(c *gin.Context) {
		c.Set("user_id", uint(1))
		c.Next()
	})

	router.POST("/order", controllers.CreateOrder)
	router.POST("/callback", controllers.PaymentCallback)

	return router
}

var orderId any

func TestCreateOrder_Success(t *testing.T) {

	database.DB.Create(&models.Game{ID: 1, Name: "Game X"})
	database.DB.Create(&models.User{Id: 10, Name: "Seller", Username: "seller", Email: "Seller1@test.com"})
	database.DB.Create(&models.Product{Id: 1, UserId: 10, Stock: 10, Price: 50000, GameId: 1, Status: "available", Guarantee: 14})

	defer gock.Off()

	gock.New("https://app.sandbox.midtrans.com").
		Post("/snap/v1/transactions").
		Reply(201).
		JSON(map[string]interface{}{
			"token":        "dummy-snap-token-12345",
			"redirect_url": "https://app.sandbox.midtrans.com/snap/v2/vtweb/dummy-snap-token-12345",
		})

	router := setupTestRouter()
	t.Log(router.Routes())

	payload := map[string]interface{}{
		"product_id": 1,
		"qty":        2,
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "/order", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Rekam Response
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["snap_token"])
	assert.Equal(t, "dummy-snap-token-12345", data["snap_token"])
	// fmt.Println("DATA RESPON:", data)
	orderData := data["order"].(map[string]interface{})
	orderId = orderData["id"]

	// Validasi Database
	var order models.Order
	database.DB.First(&order, "product_id = ?", 1)
	assert.Equal(t, "PENDING", order.Status)

}

func TestPaymentCallback_SettlementSuccess(t *testing.T) {
	original := controllers.SendInvoiceEmail

	controllers.SendInvoiceEmail = func(to string, pdfBuffer *bytes.Buffer, invoiceID string) error {
		return nil
	}

	defer func() {
		controllers.SendInvoiceEmail = original
	}()

	defer gock.Off()

	gock.New("https://api.sandbox.midtrans.com").
		Get(fmt.Sprintf("/v2/%s/status", orderId)).
		Reply(200).
		JSON(map[string]interface{}{
			"transaction_status": "settlement",
		})

	router := setupTestRouter()

	//  Payload
	payload := map[string]interface{}{
		"order_id": orderId,
	}
	body, _ := json.Marshal(payload)
	// fmt.Printf("BODY YANG DIKIRIM: %s\n", body)

	req, _ := http.NewRequest("POST", "/callback", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	// resp := w.Body.String()
	// fmt.Println("Body Response:", resp)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)

	// Validasi Database
	var order models.Order
	database.DB.Where("id = ?", orderId).First(&order)
	assert.Equal(t, "PAID", order.Status)

	var product models.Product
	database.DB.Where("id = ?", 1).First(&product)
	assert.Equal(t, 8, product.Stock)

}
