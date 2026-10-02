package middlewares

import (
	"bnsp2/server/config"
	"bnsp2/server/helpers"
	"bnsp2/server/structs"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"strconv"
)

var accessKey = config.GetJWTAccessKey()

func AuthMiddleware() gin.HandlerFunc {

	return func(c *gin.Context) {

		tokenString, err := c.Cookie("access_token")

		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, structs.ErrorResponse{
				Success: false,
				Message: "unauthorized",
			},
			)
			return
		}

		claims := &helpers.JWTClaims{}

		token, err := jwt.ParseWithClaims(
			tokenString,
			claims,
			func(token *jwt.Token) (interface{}, error) {
				return accessKey, nil
			})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, structs.ErrorResponse{
				Success: false,
				Message: "Invalid token",
			})
			return
		}

		c.Set("user_id", claims.UserId)
		c.Set("role", claims.Role)

		c.Next()
	}
}

func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {

		roleValue, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusForbidden, structs.ErrorResponse{
				Success: false,
				Message: "Role not found",
			})
			c.Abort()
			return
		}

		role, ok := roleValue.(string)
		if !ok || role != "admin" {
			c.JSON(http.StatusForbidden, structs.ErrorResponse{
				Success: false,
				Message: "Action forbidden",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
func OwnerOrAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {

		paramID := c.Param("id")
		id, _ := strconv.Atoi(paramID)

		user_id, _ := c.Get("user_id")
		role, _ := c.Get("role")

		if role != "admin" && uint(id) != user_id.(uint) {
			c.JSON(http.StatusForbidden, structs.ErrorResponse{
				Success: false,
				Message: "Action forbidden",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
