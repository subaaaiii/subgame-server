package handlers

import (
	"bnsp2/server/database"
	"bnsp2/server/helpers"
	"bnsp2/server/models"
	"strconv"

	"github.com/gin-gonic/gin"
)

func GetMessages(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	targetIDStr := c.Query("target_id")

	targetID, _ := strconv.Atoi(targetIDStr)

	var messages []models.Message

	database.DB.Where(
		"(from_user_id = ? AND to_user_id = ?) OR (from_user_id = ? AND to_user_id = ?)",
		userID, targetID, targetID, userID,
	).Order("created_at asc").Find(&messages)

	var result []MessageResponse

	for _, m := range messages {
		result = append(result, MessageResponse{
			From:      strconv.FormatUint(uint64(m.FromUserID), 10),
			To:        strconv.FormatUint(uint64(m.ToUserID), 10),
			Message:   m.Message,
			ProductId: m.ProductId,
			OrderId:   m.OrderId,
			Timestamp: m.CreatedAt.Unix(),
		})
	}

	c.JSON(200, result)

}

func GetChatListHandler(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	q := c.Query("q")

	result, err := helpers.GetChatList(database.DB, userID, q)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, result)
}
