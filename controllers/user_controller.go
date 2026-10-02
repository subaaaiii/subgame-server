package controllers

import (
	"bnsp2/server/database"
	"bnsp2/server/helpers"
	"bnsp2/server/models"
	"bnsp2/server/redis"
	"bnsp2/server/structs"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func FindUsers(c *gin.Context) {

	var users []models.User

	if err := database.DB.Find(&users).Error; err != nil {
		c.JSON(http.StatusNotFound, structs.ErrorResponse{
			Success: false,
			Message: "No User found",
		})
		return
	}

	c.JSON(http.StatusOK, structs.SuccessResponse{
		Success: true,
		Message: "Lists Data Users",
		Data:    users,
	})
}

func FindUserById(c *gin.Context) {

	id := c.Param("id")

	var user models.User

	key := fmt.Sprintf("user_profile:%v", id)

	ctx := c.Request.Context()
	var cachedResp structs.UserProfileResponse
	if cached, err := redis.RedisClient.Get(ctx, key).Result(); err == nil {
		if err := json.Unmarshal([]byte(cached), &cachedResp); err == nil {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"message": "user retrieved from cache",
				"data":    cachedResp,
			})
			fmt.Println("cache user id hit", cachedResp)
			return
		}
		redis.RedisClient.Del(ctx, key)
	}

	if err := database.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, structs.ErrorResponse{
			Success: false,
			Message: "User not found",
		})
		return
	}

	resp := structs.UserProfileResponse{
		Name:     user.Name,
		Birthday: helpers.FormatBirthday(user.Birthday),
		Gender:   user.Gender,
		Address:  user.Address,
		Picture:  user.Picture,
	}

	if data, err := json.Marshal(resp); err == nil {
		redis.RedisClient.Set(ctx, key, data, time.Hour)
	}

	c.JSON(http.StatusOK, structs.SuccessResponse{
		Success: true,
		Message: "User Found",
		Data:    resp,
	})

}

func UpdateUser(c *gin.Context) {

	id := c.Param("id")

	var user models.User

	if err := database.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, structs.ErrorResponse{
			Success: false,
			Message: "User not found",
		})
		return
	}

	name := c.PostForm("name")
	address := c.PostForm("address")
	gender := c.PostForm("gender")
	birthday := c.PostForm("birthday")

	updates := map[string]interface{}{}

	if name != "" {
		updates["name"] = name
	}
	if address != "" {
		updates["address"] = address
	}
	if gender != "" {
		updates["gender"] = gender
	}
	if birthday != "" {
		parsedDate, err := time.Parse("2006-01-02", birthday)
		if err != nil {
			c.JSON(http.StatusBadRequest, structs.ErrorResponse{
				Success: false,
				Message: "Invalid date format"})
			return
		}
		updates["birthday"] = parsedDate
	}

	file, err := c.FormFile("picture")
	if err == nil {
		if user.Picture != "" && user.Picture != "default.png" {

			oldPath := "images/users/" + user.Picture

			if err := os.Remove(oldPath); err != nil {
				fmt.Println("failed delete old image:", err)
			}
		}

		filename := fmt.Sprintf(
			"user-%s-%d%s",
			id,
			time.Now().Unix(),
			filepath.Ext(file.Filename),
		)

		path := "images/users/" + filename

		if err := c.SaveUploadedFile(file, path); err != nil {
			c.JSON(http.StatusInternalServerError, structs.ErrorResponse{
				Success: false,
				Message: "Failed to upload picture",
			})
			return
		}
		updates["picture"] = filename
	}

	if err := database.DB.Model(&user).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, structs.ErrorResponse{
			Success: false,
			Message: "Failed to update user",
		})
		return
	}

	ctx := c.Request.Context()

	redis.RedisClient.Del(ctx, fmt.Sprintf("user:%v", user.Id))
	redis.RedisClient.Del(ctx, fmt.Sprintf("user_profile:%v", user.Id))

	c.JSON(http.StatusOK, structs.SuccessResponse{
		Success: false,
		Message: "User updated successfully",
		Data:    user,
	})
}

func DeleteUser(c *gin.Context) {

	id := c.Param("id")

	var user models.User

	if err := database.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, structs.ErrorResponse{
			Success: false,
			Message: "User not found",
			Errors:  helpers.TranslateErrorMessage(err),
		})
		return
	}

	if err := database.DB.Delete(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, structs.ErrorResponse{
			Success: false,
			Message: "Failed to delete user",
			Errors:  helpers.TranslateErrorMessage(err),
		})
		return
	}

	ctx := c.Request.Context()

	redis.RedisClient.Del(ctx, fmt.Sprintf("user:%v", user.Id))
	redis.RedisClient.Del(ctx, fmt.Sprintf("user_profile:%v", user.Id))

	c.JSON(http.StatusOK, structs.SuccessResponse{
		Success: true,
		Message: "User deleted successfully",
	})
}

func Me(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, structs.ErrorResponse{
			Success: false,
			Message: "Unauthorized",
		})
		return
	}

	cacheKey := fmt.Sprintf("user:%v", userID)

	ctx := context.Background()

	cached, err := redis.RedisClient.Get(ctx, cacheKey).Result()
	if err == nil && cached != "" {
		var res structs.UserResponse

		if err := json.Unmarshal([]byte(cached), &res); err == nil {
			c.JSON(http.StatusOK, structs.SuccessResponse{
				Success: true,
				Message: "Data user",
				Data:    res,
			})
			fmt.Println("cache user hit", res)
			return
		}
	}

	var user models.User

	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, structs.ErrorResponse{
			Success: false,
			Message: "User not found",
		})
		return
	}

	res := structs.UserResponse{
		Id:            user.Id,
		Name:          user.Name,
		Username:      user.Username,
		Email:         user.Email,
		Picture:       user.Picture,
		Role:          user.Role,
		EmailVerified: user.EmailVerified,
		CreatedAt:     user.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:     user.UpdatedAt.Format("2006-01-02 15:04:05"),
	}

	jsonData, err := json.Marshal(res)
	if err == nil {
		_ = redis.RedisClient.Set(
			ctx,
			cacheKey,
			jsonData,
			10*time.Minute,
		).Err()
	}

	c.JSON(http.StatusOK, structs.SuccessResponse{
		Success: true,
		Message: "Data user",
		Data:    res,
	})
}

func VerifyPassword(c *gin.Context) {
	var req structs.VerifyPasswordRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, structs.ErrorResponse{
			Success: false,
			Message: "Validation Errors",
			Errors:  helpers.TranslateErrorMessage(err),
		})
		return
	}

	userID := c.MustGet("user_id").(uint)

	var user models.User

	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, structs.ErrorResponse{
			Success: false,
			Message: "User tidak ditemukan",
		})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, structs.ErrorResponse{
			Success: false,
			Message: "Invalid Password",
			Errors: map[string]string{
				"Password": "Wrong password",
			},
		})
		return
	}

	c.JSON(http.StatusOK, structs.SuccessResponse{
		Success: true,
		Message: "Password Verified",
	})
}

func ChangePassword(c *gin.Context) {
	var req structs.ChangePasswordRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, structs.ErrorResponse{
			Success: false,
			Message: "Validation Errors",
			Errors:  helpers.TranslateErrorMessage(err),
		})
		return
	}

	userID := c.MustGet("user_id").(uint)

	var user models.User

	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, structs.ErrorResponse{
			Success: false,
			Message: "User tidak ditemukan",
		})
		return
	}

	user.Password = helpers.HashPassword(req.NewPassword)
	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, structs.ErrorResponse{
			Success: false, Message: "Failed to update password"})
		return
	}

	c.JSON(http.StatusOK, structs.SuccessResponse{
		Success: true,
		Message: "Password Updated Successfully",
	})
}

func SendResetPasswordEmail(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, structs.ErrorResponse{
			Success: false,
			Message: "Validation Errors",
			Errors:  helpers.TranslateErrorMessage(err),
		})
		return
	}

	var user models.User

	if err := database.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, structs.ErrorResponse{
			Success: false,
			Message: "Email not registered",
		})
		return
	}

	token := helpers.GenerateResetToken(user.Id)

	ctx := c.Request.Context()
	key := fmt.Sprintf("reset_token:%v", token)

	err := redis.RedisClient.Set(ctx, key, user.Id, 10*time.Minute).Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, structs.ErrorResponse{
			Success: false,
			Message: "Failed to set token",
		})
		return
	}

	if err := helpers.SendResetPasswordEmail(user.Email, token); err != nil {
		c.JSON(http.StatusInternalServerError, structs.ErrorResponse{
			Success: false,
			Message: "Failed to send reset email",
		})
		return
	}
	c.JSON(http.StatusOK, structs.SuccessResponse{
		Success: true,
		Message: "Email for password sent successfully",
	})

}

func ResetPassword(c *gin.Context) {
	var req struct {
		Token    string `json:"token" binding:"required"`
		Password string `json:"password" binding:"required,min=8"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, structs.ErrorResponse{
			Success: false,
			Message: "Validation Errors",
			Errors:  helpers.TranslateErrorMessage(err),
		})
		return
	}

	ctx := c.Request.Context()
	key := fmt.Sprintf("reset_token:%v", req.Token)

	result, err := redis.RedisClient.Get(ctx, key).Result()

	if err != nil {
		c.JSON(http.StatusBadRequest, structs.ErrorResponse{
			Success: false,
			Message: "Token expired",
		})
		return
	}

	userId, err := strconv.ParseUint(result, 10, 64)
	if err != nil {
		c.JSON(http.StatusInternalServerError, structs.ErrorResponse{
			Success: false,
			Message: "Failed to parse user id",
		})
		return
	}

	var user models.User

	if err := database.DB.Where("id = ?", userId).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, structs.ErrorResponse{
			Success: false,
			Message: "User not found",
		})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err == nil {
		c.JSON(http.StatusBadRequest, structs.ErrorResponse{
			Success: false,
			Errors: map[string]string{
				"Password": "New password cannot be the same as the old password.",
			},
		})
		return
	}
	user.Password = helpers.HashPassword(req.Password)

	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, structs.ErrorResponse{
			Success: false,
			Message: "Failed to reset password",
		})
		return
	}

	c.JSON(http.StatusOK, structs.SuccessResponse{
		Success: true,
		Message: "Password reset successfully, please login again",
	})

}

type ChangeEmailRequest struct {
	NewEmail          string `json:"new_email" binding:"required,email"`
	VerificationToken string `json:"verification_token" binding:"required"`
}

func ChangeEmail(c *gin.Context) {
	var req ChangeEmailRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, structs.ErrorResponse{
			Success: false,
			Message: "Validation Errors",
			Errors:  helpers.TranslateErrorMessage(err),
		})
		return
	}

	userID := c.GetUint("user_id")

	ctx := c.Request.Context()

	key := fmt.Sprintf("verified:change_email:%s", req.NewEmail)

	token, err := redis.RedisClient.Get(ctx, key).Result()

	if err != nil {
		c.JSON(http.StatusBadRequest, structs.ErrorResponse{
			Success: false,
			Message: "Email verification required",
		})
		return
	}

	if token != req.VerificationToken {
		c.JSON(http.StatusBadRequest, structs.ErrorResponse{
			Success: false,
			Message: "Invalid verification token",
		})
		return
	}

	var existing models.User

	err = database.DB.
		Where("email = ?", req.NewEmail).
		First(&existing).Error

	if err == nil {
		c.JSON(http.StatusBadRequest, structs.ErrorResponse{
			Success: false,
			Message: "Email already registered",
		})
		return
	}

	if err := database.DB.
		Model(&models.User{}).
		Where("id = ?", userID).
		Update("email", req.NewEmail).Error; err != nil {

		c.JSON(http.StatusInternalServerError, structs.ErrorResponse{
			Success: false,
			Message: "Failed to update email",
		})
		return
	}

	redis.RedisClient.Del(ctx, key)
	redis.RedisClient.Del(ctx, fmt.Sprintf("user:%v", userID))
	redis.RedisClient.Del(ctx, fmt.Sprintf("user_profile:%v", userID))

	c.JSON(http.StatusOK, structs.SuccessResponse{
		Success: true,
		Message: "Email updated successfully",
	})
}
