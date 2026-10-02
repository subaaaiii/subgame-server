package handlers

import (
	"bnsp2/server/config"
	"bnsp2/server/helpers"
	"bnsp2/server/structs"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var refreshKey = config.GetJWTRefreshKey()

func Refresh(c *gin.Context) {

	refreshToken, err := c.Cookie(
		"refresh_token",
	)

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "unauthorized",
		})
		return
	}

	claims := &helpers.JWTClaims{}

	token, err := jwt.ParseWithClaims(
		refreshToken,
		claims,
		func(token *jwt.Token) (interface{}, error) {
			return []byte(refreshKey), nil
		},
	)

	if err != nil || !token.Valid {
		c.AbortWithStatusJSON(http.StatusUnauthorized, structs.ErrorResponse{
			Success: false,
			Message: "Invalid token",
		})
		return
	}

	newAccessToken := helpers.GenerateAccessToken(claims.UserId, claims.Role)

	c.SetCookie(
		"access_token",
		newAccessToken,
		900,
		"/",
		"",
		false,
		true,
	)

	c.JSON(http.StatusOK, structs.SuccessResponse{
		Success: true,
		Message: "Token refreshed",
	})
}
