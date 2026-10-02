package handlers

import (
	"log"
	"net/http"
	"strconv"

	"bnsp2/server/services"

	"github.com/gin-gonic/gin"
)

func AuthHandler(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	userIDStr := strconv.FormatUint(uint64(userID), 10)

	if err := c.Request.ParseForm(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad request"})
		return
	}

	channelName := c.Request.FormValue("channel_name")
	socketID := c.Request.FormValue("socket_id")

	log.Println("channel:", channelName)
	log.Println("socket:", socketID)

	expected := "private-chat-" + userIDStr

	if channelName != expected {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
		return
	}

	body := c.Request.PostForm.Encode()

	authResponse, err := services.Client.AuthorizePrivateChannel([]byte(body))
	if err != nil {
		log.Println("auth error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Auth failed"})
		return
	}

	c.Data(http.StatusOK, "application/json", authResponse)
}
