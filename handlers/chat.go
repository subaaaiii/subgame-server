package handlers

import (
	"net/http"
	"strconv"

	"bnsp2/server/database"
	"bnsp2/server/models"
	"bnsp2/server/services"

	"github.com/gin-gonic/gin"
)

type MessageRequest struct {
	To        string `json:"to" binding:"required"`
	Message   string `json:"message" binding:"required"`
	OrderId   string `json:"order_id"`
	ProductId string `json:"product_id"`
}

type MessageResponse struct {
	From      string  `json:"from"`
	To        string  `json:"to"`
	Message   string  `json:"message"`
	Timestamp int64   `json:"timestamp"`
	ProductId *uint   `json:"product_id"`
	OrderId   *string `json:"order_id"`
}

func SendMessage(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	userIDStr := strconv.FormatUint(uint64(userID), 10)

	var req MessageRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	toUint, err := strconv.ParseUint(req.To, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid target user"})
		return
	}
	toID := uint(toUint)

	if userID == toID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot send message to yourself"})
		return
	}

	msgDB := models.Message{
		FromUserID: userID,
		ToUserID:   toID,
		Message:    req.Message,
	}

	if req.ProductId != "" {
		pid, _ := strconv.Atoi(req.ProductId)
		id := uint(pid)
		msgDB.ProductId = &id
	}

	if req.OrderId != "" {
		msgDB.OrderId = &req.OrderId
	}

	if err := database.DB.Create(&msgDB).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save message"})
		return
	}

	msg := MessageResponse{
		From:      userIDStr,
		To:        req.To,
		Message:   msgDB.Message,
		Timestamp: msgDB.CreatedAt.Unix(),
	}

	channel := "private-chat-" + req.To

	if err := services.Client.Trigger(channel, "new-message", msg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send message"})
		return
	}

	sender := "private-chat-" + userIDStr
	if err := services.Client.Trigger(sender, "new-message", msg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send message"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "sent",
		"data":   msg,
	})
}
