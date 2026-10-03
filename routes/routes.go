package routes

import (
	"bnsp2/server/config"
	"bnsp2/server/controllers"
	"bnsp2/server/handlers"
	"bnsp2/server/middlewares"
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {

	//initialize gin
	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{config.GetEnv("ALLOWED_ORIGIN", "http://localhost:5173")},
		AllowMethods:     []string{"GET", "PATCH", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	router.Use(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/images/") {
			c.Header(
				"Cache-Control",
				"public, max-age=31536000, immutable",
			)
		}

		c.Next()
	})

	router.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "SubGAME API is running",
			"version": "CI/CD test v1",
		})
	})

	// route register
	router.Static("/images", "./images")
	// router.POST("/api/register", middlewares.RateLimit(10, time.Hour), controllers.Register)
	router.POST("/api/register", controllers.Register)
	router.POST("/api/register/resend-otp", middlewares.RateLimit(1, time.Minute), controllers.ResendOTPRegister)
	router.POST("/api/users", controllers.CreateUser)
	router.GET("/api/me", middlewares.AuthMiddleware(), controllers.Me)
	router.POST("/api/login", middlewares.RateLimit(5, time.Minute), controllers.Login)
	router.POST("/api/logout", controllers.Logout)
	router.POST("/api/auth/refresh", middlewares.RateLimit(30, time.Minute), handlers.Refresh)
	// router.GET("/api/users", controllers.FindUsers)
	router.GET("/api/users", middlewares.AuthMiddleware(), middlewares.AdminOnly(), controllers.FindUsers)
	router.GET("/api/users/:id", middlewares.RateLimit(100, time.Minute), middlewares.AuthMiddleware(), middlewares.OwnerOrAdmin(), controllers.FindUserById)
	router.PATCH("/api/users/:id", middlewares.AuthMiddleware(), middlewares.OwnerOrAdmin(), controllers.UpdateUser)
	router.DELETE("/api/users/:id", middlewares.AuthMiddleware(), middlewares.OwnerOrAdmin(), controllers.DeleteUser)
	// router.POST("/send-otp-change-email", middlewares.AuthMiddleware(), middlewares.RateLimit(10, time.Hour), controllers.SendChangeEmailOTP)
	router.POST("/send-otp", middlewares.AuthMiddleware(), controllers.SendOTP)
	router.POST("/verify-otp", middlewares.AuthMiddleware(), controllers.VerifyOTP)
	router.POST("/change-email", middlewares.AuthMiddleware(), controllers.ChangeEmail)
	// router.POST("/send-otp-verify-email", middlewares.RateLimit(10, time.Hour), controllers.SendEmailOTP)
	// router.POST("/verify-email", middlewares.RateLimit(5, time.Minute), controllers.VerifyEmailOTP)
	// router.POST("/verify-otp-change-email", middlewares.AuthMiddleware(), middlewares.RateLimit(5, time.Minute), controllers.VerifyChangeEmailOTP)
	router.POST("/verify-password", middlewares.AuthMiddleware(), middlewares.RateLimit(10, time.Minute), controllers.VerifyPassword)
	router.POST("/change-password", middlewares.AuthMiddleware(), middlewares.RateLimit(5, time.Hour), controllers.ChangePassword)
	// router.POST("/change-password", middlewares.AuthMiddleware(), controllers.ChangePassword)
	router.GET("/auth/google", controllers.GoogleLogin)
	router.GET("/auth/google/callback", controllers.GoogleCallback)
	router.POST("/api/sellers/register", middlewares.AuthMiddleware(), controllers.RegisterSeller)
	router.GET("/api/seller", middlewares.AuthMiddleware(), controllers.GetSellerByUserId)
	router.GET("/api/sellers", middlewares.AuthMiddleware(), controllers.GetSellers)
	router.PATCH("/api/sellers/status", middlewares.AuthMiddleware(), controllers.UpdateSellerStatus)
	router.POST("/api/forgot-password", middlewares.RateLimit(10, time.Hour), controllers.SendResetPasswordEmail)
	router.POST("/api/reset-password", middlewares.RateLimit(10, time.Hour), controllers.ResetPassword)
	admin := router.Group("/api/admin")
	{
		admin.POST("/games", controllers.CreateGame)
		admin.GET("/games", middlewares.RateLimit(100, time.Minute), controllers.GetGames)
		admin.GET("/games/:id", controllers.GetGameByID)
		admin.PUT("/games/:id", controllers.UpdateGame)
		admin.DELETE("/games/:id", controllers.DeleteGame)
	}
	router.POST("/api/products", middlewares.AuthMiddleware(), middlewares.RateLimit(30, time.Hour), controllers.CreateProduct)
	router.GET("/api/products", middlewares.AuthMiddleware(), controllers.GetProducts)
	router.GET("/api/products/:id", middlewares.RateLimit(100, time.Minute), controllers.GetProductByID)
	router.PUT("/api/products/:id", middlewares.AuthMiddleware(), middlewares.RateLimit(80, time.Hour), controllers.UpdateProduct)
	router.DELETE("/api/products/:id", middlewares.AuthMiddleware(), middlewares.RateLimit(40, time.Hour), controllers.DeleteProduct)
	router.PATCH("/api/products/status", middlewares.AuthMiddleware(), middlewares.RateLimit(60, time.Hour), controllers.ChangeProductStatus)
	router.GET("/api/products/public", middlewares.RateLimit(100, time.Minute), controllers.GetProductsPublic)

	router.POST("/pusher/auth", middlewares.AuthMiddleware(), handlers.AuthHandler)
	router.POST("/api/chat/send", middlewares.AuthMiddleware(), middlewares.RateLimit(40, time.Minute), handlers.SendMessage)
	router.GET("/api/chat/messages", middlewares.AuthMiddleware(), middlewares.RateLimit(120, time.Minute), handlers.GetMessages)
	router.GET("/api/chat/list", middlewares.AuthMiddleware(), handlers.GetChatListHandler)
	// router.POST("/api/payment/callback", middlewares.AuthMiddleware(), controllers.PaymentCallback)
	router.POST("/api/payment/callback", controllers.PaymentCallback)

	router.POST("/api/orders", middlewares.AuthMiddleware(), middlewares.RateLimit(10, time.Minute), controllers.CreateOrder)
	router.GET("/api/orders/:id", middlewares.AuthMiddleware(), controllers.GetOrder)
	router.GET("/api/orders", middlewares.AuthMiddleware(), middlewares.RateLimit(100, time.Minute), controllers.GetOrders)
	router.POST("/api/orders/status", middlewares.AuthMiddleware(), middlewares.RateLimit(50, time.Minute), controllers.UpdateStatusOrder)
	router.GET("/api/orders/batch", middlewares.AuthMiddleware(), controllers.GetOrderBatch)
	router.GET("/api/products/batch", middlewares.AuthMiddleware(), controllers.GetProductBatch)
	return router
}
