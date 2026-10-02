package controllers

import (
	"bnsp2/server/database"
	"bnsp2/server/helpers"
	"bnsp2/server/models"
	"bnsp2/server/redis"
	"bnsp2/server/structs"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// func SendEmailOTP(c *gin.Context) {

// 	var req = structs.UserVerifyEmailRequest{}
// 	var user = models.User{}

// 	// Validasi input dari request body menggunakan ShouldBindJSON
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusUnprocessableEntity, structs.ErrorResponse{
// 			Success: false,
// 			Message: "Validation Errors",
// 			Errors:  helpers.TranslateErrorMessage(err),
// 		})
// 		return
// 	}

// 	// ambil user dari database
// 	if err := database.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
// 		c.JSON(http.StatusNotFound, structs.ErrorResponse{
// 			Success: false,
// 			Message: "User not found",
// 		})
// 		return
// 	}

// 	// generate OTP
// 	otp := helpers.GenerateOTP()

// 	// set expired OTP
// 	expired := time.Now().Add(5 * time.Minute)

// 	// simpan OTP ke database
// 	user.EmailOTP = otp
// 	user.OTPExpiresAt = &expired

// 	database.DB.Save(&user)

// 	// kirim OTP ke email
// 	err := helpers.SendOTPEmail(req.Email, otp)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, structs.ErrorResponse{
// 			Success: false,
// 			Message: "OTP Failed to send",
// 		})
// 		return
// 	}

// 	c.JSON(http.StatusOK, structs.SuccessResponse{
// 		Success: true,
// 		Message: "OTP Sent to email",
// 	})

// }

type VerifyEmailRequest struct {
	OTP   string `json:"otp"`
	Email string `json:"email"`
}

// func VerifyEmailOTP(c *gin.Context) {
// 	var req VerifyEmailRequest

// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, structs.ErrorResponse{
// 			Success: false,
// 			Message: "Validation Errors",
// 			Errors:  helpers.TranslateErrorMessage(err),
// 		})
// 		return
// 	}

// 	var user models.User

// 	// cari user berdasarkan email
// 	if err := database.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
// 		c.JSON(http.StatusNotFound, structs.ErrorResponse{
// 			Success: false,
// 			Message: "User not found",
// 		})
// 		return
// 	}

// 	// cek OTP
// 	if user.EmailOTP != req.OTP {
// 		c.JSON(http.StatusBadRequest, structs.ErrorResponse{
// 			Success: false,
// 			Message: "Wrong OTP",
// 		})
// 		return
// 	}

// 	// cek expired
// 	if user.OTPExpiresAt == nil || time.Now().After(*user.OTPExpiresAt) {
// 		c.JSON(http.StatusBadRequest, structs.ErrorResponse{
// 			Success: false,
// 			Message: "OTP expired",
// 		})
// 		return
// 	}

// 	// update email (optional tergantung flow)
// 	user.EmailOTP = ""
// 	user.OTPExpiresAt = nil // reset
// 	user.EmailVerified = true

// 	if err := database.DB.Save(&user).Error; err != nil {
// 		c.JSON(http.StatusInternalServerError, structs.ErrorResponse{
// 			Message: "Failed update user",
// 		})
// 		return
// 	}

// 	c.JSON(http.StatusOK, structs.SuccessResponse{
// 		Success: true,
// 		Message: "Email Verified",
// 	})
// }

type SendOTPRequest struct {
	Email   string `json:"email" binding:"required,email"`
	Purpose string `json:"purpose" binding:"required"`
}

type OTPData struct {
	Email   string `json:"email"`
	OTP     string `json:"otp"`
	Purpose string `json:"purpose"`
}

func SendOTP(c *gin.Context) {
	var req = SendOTPRequest{}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, structs.ErrorResponse{
			Success: false,
			Message: "Validation Errors",
			Errors:  helpers.TranslateErrorMessage(err),
		})
		return
	}

	if req.Purpose == "change_email" {
		var existing models.User

		err := database.DB.
			Where("email = ?", req.Email).
			First(&existing).Error

		if err == nil {
			c.JSON(http.StatusBadRequest, structs.ErrorResponse{
				Success: false,
				Message: "Email already registered",
			})
			return
		}
	}

	key := fmt.Sprintf("otp:%s:%s", req.Purpose, req.Email)

	ctx := c.Request.Context()
	otp := helpers.GenerateOTP()

	otpData := OTPData{
		Email:   req.Email,
		OTP:     otp,
		Purpose: req.Purpose,
	}

	jsonData, _ := json.Marshal(otpData)

	redis.RedisClient.Set(
		ctx,
		key,
		jsonData,
		10*time.Minute,
	)

	helpers.SendOTPEmail(req.Email, otp)

	log.Println(key)

	c.JSON(http.StatusOK, structs.SuccessResponse{
		Success: true,
		Message: "OTP Resent",
	})

}

type VerifyRequest struct {
	Email   string `json:"email" binding:"required,email"`
	OTP     string `json:"otp" binding:"required"`
	Purpose string `json:"purpose" binding:"required"`
}

func VerifyOTP(c *gin.Context) {
	var req = VerifyRequest{}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, structs.ErrorResponse{
			Success: false,
			Message: "Validation Errors",
			Errors:  helpers.TranslateErrorMessage(err),
		})
		return
	}

	key := fmt.Sprintf("otp:%s:%s", req.Purpose, req.Email)

	log.Println(key)

	ctx := c.Request.Context()

	result, err := redis.RedisClient.Get(ctx, key).Result()
	if err != nil {
		c.JSON(http.StatusBadRequest, structs.ErrorResponse{
			Success: false,
			Message: "OTP expired",
		})
		return
	}

	var otpData OTPData

	if err := json.Unmarshal([]byte(result), &otpData); err != nil {
		c.JSON(http.StatusInternalServerError, structs.ErrorResponse{
			Success: false,
			Message: "Failed to read otp data",
		})
		return
	}

	if otpData.OTP != req.OTP {
		c.JSON(http.StatusBadRequest, structs.ErrorResponse{
			Success: false,
			Message: "Invalid OTP",
		})
		return
	}
	redis.RedisClient.Del(ctx, key)

	if req.Purpose == "change_email" {
		verificationToken := uuid.NewString()

		key := fmt.Sprintf(
			"verified:change_email:%s",
			req.Email,
		)

		err := redis.RedisClient.Set(
			ctx,
			key,
			verificationToken,
			10*time.Minute,
		).Err()

		log.Println(key)

		if err != nil {
			c.JSON(http.StatusInternalServerError, structs.ErrorResponse{
				Success: false,
				Message: "Failed to set cache",
			})
		}

		c.JSON(http.StatusOK, structs.SuccessResponse{
			Success: true,
			Message: "Verify OTP success",
			Data: gin.H{
				"verification_token": verificationToken,
			},
		})

		return
	}

	c.JSON(http.StatusOK, structs.SuccessResponse{
		Success: true,
		Message: "Valid OTP",
	})
}

type VerifyOTPRequest struct {
	OTP      string `json:"otp"`
	NewEmail string `json:"new_email"`
}

// func VerifyChangeEmailOTP(c *gin.Context) {

// 	userID := c.MustGet("user_id").(uint)

// 	var req VerifyOTPRequest
// 	c.ShouldBindJSON(&req)

// 	var user models.User

// 	database.DB.First(&user, userID)

// 	// cek OTP
// 	if user.EmailOTP != req.OTP {
// 		c.JSON(http.StatusBadRequest, structs.ErrorResponse{
// 			Success: false,
// 			Message: "wrong OTP",
// 		})
// 		return
// 	}

// 	// cek expired
// 	if user.OTPExpiresAt == nil || time.Now().After(*user.OTPExpiresAt) {
// 		c.JSON(http.StatusBadRequest, structs.ErrorResponse{
// 			Success: false,
// 			Message: "OTP expired",
// 		})
// 		return
// 	}

// 	// update email
// 	user.Email = req.NewEmail
// 	user.EmailOTP = ""

// 	database.DB.Save(&user)

// 	c.JSON(http.StatusOK, structs.SuccessResponse{
// 		Success: true,
// 		Message: "Email successfully changed",
// 	})
// }
