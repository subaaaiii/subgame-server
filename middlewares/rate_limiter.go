package middlewares

import (
	"bnsp2/server/redis"
	"bnsp2/server/structs"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func RateLimit(maxRequest int64, duration time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {

		userID, exists := c.Get("user_id")
		var key string

		if exists {
			uid := userID.(uint)
			key = fmt.Sprintf(
				"rate_limit:%s:user:%d",
				c.FullPath(),
				uid,
			)
		} else {
			key = fmt.Sprintf(
				"rate_limit:%s:ip:%s",
				c.FullPath(),
				c.ClientIP(),
			)
		}
		ctx := c.Request.Context()

		count, err := redis.RedisClient.Incr(ctx, key).Result()
		if err != nil {

			c.Next()
			return
		}

		if count == 1 {
			redis.RedisClient.Expire(ctx, key, duration)
		}

		if count > maxRequest {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, structs.ErrorResponse{
				Success: false,
				Message: "Too many requests, wait a moments",
			})
			return
		}
		c.Next()
	}
}
