package controllers

import (
	"bnsp2/server/database"
	"bnsp2/server/helpers"
	"bnsp2/server/models"
	"bnsp2/server/redis"
	"bnsp2/server/structs"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/markbates/goth/gothic"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type RegisterTemp struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	OTP      string `json:"otp"`
}

var SendOTPEmail = helpers.SendOTPEmail

func Register(c *gin.Context) {
	var req = structs.UserCreateRequest{}

	if err := c.ShouldBindJSON(&req); err != nil {

		c.JSON(http.StatusUnprocessableEntity, structs.ErrorResponse{
			Success: false,
			Message: "Validasi Errors",
			Errors:  helpers.TranslateErrorMessage(err),
		})
		return
	}

	var emailExists bool
	var usernameExists bool

	database.DB.
		Model(&models.User{}).
		Where("email = ?", req.Email).
		Select("count(*) > 0").
		Find(&emailExists)

	database.DB.
		Model(&models.User{}).
		Where("username = ?", req.Username).
		Select("count(*) > 0").
		Find(&usernameExists)

	errors := map[string]string{}

	if emailExists {
		errors["Email"] = "Email already exists"
	}

	if usernameExists {
		errors["Username"] = "Username already exists"
	}

	if len(errors) > 0 {
		c.JSON(http.StatusConflict, structs.ErrorResponse{
			Success: false,
			Message: "Duplicate entry",
			Errors:  errors,
		})
		return
	}

	otp := helpers.GenerateOTP()

	data := RegisterTemp{
		Name:     req.Name,
		Username: req.Username,
		Email:    req.Email,
		Password: helpers.HashPassword(req.Password),
		OTP:      otp,
	}

	jsonData, _ := json.Marshal(data)

	ctx := c.Request.Context()

	key := fmt.Sprintf("register:%s", req.Email)

	exists, _ := redis.RedisClient.Exists(ctx, key).Result()

	if exists > 0 {
		c.JSON(http.StatusConflict, structs.ErrorResponse{
			Success: false,
			Message: "Verification already pending",
		})
		return
	}

	redis.RedisClient.Set(
		ctx,
		fmt.Sprintf("register:%s", req.Email),
		jsonData,
		10*time.Minute,
	)

	SendOTPEmail(req.Email, otp)

	c.JSON(http.StatusOK, structs.SuccessResponse{
		Success: true,
		Message: "otp sent",
	})
}

type CreateRequest struct {
	Email string `json:"email"`
	OTP   string `json:"otp"`
}

func CreateUser(c *gin.Context) {
	var req = CreateRequest{}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, structs.ErrorResponse{
			Success: false,
			Message: "Validation Errors",
			Errors:  helpers.TranslateErrorMessage(err),
		})
		return
	}

	key := fmt.Sprintf("register:%s", req.Email)
	ctx := c.Request.Context()

	result, err := redis.RedisClient.Get(ctx, key).Result()

	if err != nil {
		c.JSON(http.StatusBadRequest, structs.ErrorResponse{
			Success: false,
			Message: "Registration data expired",
		})
		return
	}

	var temp RegisterTemp

	if err := json.Unmarshal([]byte(result), &temp); err != nil {
		c.JSON(http.StatusInternalServerError, structs.ErrorResponse{
			Success: false,
			Message: "Failed to read registration data",
		})
		return
	}

	if temp.OTP != req.OTP {
		c.JSON(http.StatusBadRequest, structs.ErrorResponse{
			Success: false,
			Message: "Invalid OTP",
		})
		return
	}

	user := models.User{
		Name:          temp.Name,
		Username:      temp.Username,
		Email:         temp.Email,
		Password:      temp.Password,
		Picture:       "default.png",
		EmailVerified: true,
	}

	if err := database.DB.Create(&user).Error; err != nil {

		if helpers.IsDuplicateEntryError(err) {

			c.JSON(http.StatusConflict, structs.ErrorResponse{
				Success: false,
				Message: "Duplicate entry error",
				Errors:  helpers.TranslateErrorMessage(err),
			})
		} else {

			c.JSON(http.StatusInternalServerError, structs.ErrorResponse{
				Success: false,
				Message: "Failed to create user",
				Errors:  helpers.TranslateErrorMessage(err),
			})
		}
		return
	}

	redis.RedisClient.Del(ctx, key)

	c.JSON(http.StatusCreated, structs.SuccessResponse{
		Success: true,
		Message: "User created successfully",
		Data: structs.UserResponse{
			Id:        user.Id,
			Name:      user.Name,
			Username:  user.Username,
			Email:     user.Email,
			Role:      user.Role,
			Picture:   user.Picture,
			CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt: user.UpdatedAt.Format("2006-01-02 15:04:05"),
		},
	})
}

type ResendOTPRequest struct {
	Email string `json:"email"`
}

func ResendOTPRegister(c *gin.Context) {
	var req = ResendOTPRequest{}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, structs.ErrorResponse{
			Success: false,
			Message: "Validation Errors",
			Errors:  helpers.TranslateErrorMessage(err),
		})
		return
	}
	key := fmt.Sprintf("register:%s", req.Email)
	ctx := c.Request.Context()

	result, err := redis.RedisClient.Get(ctx, key).Result()

	if err != nil {
		c.JSON(http.StatusBadRequest, structs.ErrorResponse{
			Success: false,
			Message: "Registration data expired",
		})
		return
	}

	var temp RegisterTemp

	if err := json.Unmarshal([]byte(result), &temp); err != nil {
		c.JSON(http.StatusInternalServerError, structs.ErrorResponse{
			Success: false,
			Message: "Failed to read registration data",
		})
		return
	}

	otp := helpers.GenerateOTP()

	temp.OTP = otp

	jsonData, err := json.Marshal(temp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, structs.ErrorResponse{
			Success: false,
			Message: "Failed to encode registration data",
		})
		return
	}

	redis.RedisClient.Set(
		ctx,
		key,
		jsonData,
		10*time.Minute,
	)

	helpers.SendOTPEmail(req.Email, otp)

	c.JSON(http.StatusOK, structs.SuccessResponse{
		Success: true,
		Message: "OTP Resent",
	})

}

func handleFailedLogin(ctx context.Context, ip string, username string) {
	key := fmt.Sprintf(
		"login_fail:%s:%s",
		ip,
		username,
	)

	count, _ := redis.RedisClient.Incr(ctx, key).Result()

	if count == 1 {
		redis.RedisClient.Expire(ctx, key, 15*time.Minute)
	}

	if count >= 3 {
		redis.RedisClient.Set(
			ctx,
			fmt.Sprintf(
				"captcha_required:%s:%s",
				ip,
				username,
			),
			"1",
			15*time.Minute,
		)
	}
}

func Login(c *gin.Context) {

	var req = structs.UserLoginRequest{}
	var user = models.User{}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, structs.ErrorResponse{
			Success: false,
			Message: "Validation Errors",
			Errors:  helpers.TranslateErrorMessage(err),
		})
		return
	}

	ctx := c.Request.Context()
	ip := c.ClientIP()

	key := fmt.Sprintf(
		"captcha_required:%s:%s",
		ip,
		req.Username,
	)
	captchaRequired, err := redis.RedisClient.Exists(
		ctx,
		key,
	).Result()

	if err != nil {
		log.Println(err)
	}

	if captchaRequired > 0 {
		if req.CaptchaToken == "" {
			c.JSON(http.StatusForbidden, structs.ErrorResponse{
				Success: false,
				Message: "Captcha required",
				Errors: map[string]string{
					"Error":   "Captcha required",
					"captcha": "Captcha required",
				},
			})
			return
		}

		valid, err := helpers.VerifyTurnstile(req.CaptchaToken)

		if err != nil || !valid {
			c.JSON(http.StatusForbidden, structs.ErrorResponse{
				Success: false,
				Message: "Invalid captcha",
				Errors: map[string]string{
					"Error":   "Invalid captcha",
					"captcha": "Invalid captcha",
				},
			})
			return
		}
	}

	if err := database.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, structs.ErrorResponse{
			Success: false,
			Message: "Wrong username or password",
			Errors: map[string]string{
				"Error": "Wrong username or password",
			},
		})
		handleFailedLogin(ctx, ip, req.Username)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, structs.ErrorResponse{
			Success: false,
			Message: "Wrong username or password",
			Errors: map[string]string{
				"Error": "Wrong username or password",
			},
		})
		handleFailedLogin(ctx, ip, req.Username)
		return
	}

	redis.RedisClient.Del(
		ctx,
		fmt.Sprintf(
			"login_fail:%s:%s",
			ip,
			req.Username,
		),
	)

	redis.RedisClient.Del(
		ctx,
		key,
	)

	accessToken := helpers.GenerateAccessToken(user.Id, user.Role)
	refreshToken := helpers.GenerateRefreshToken(user.Id, user.Role)

	c.SetCookie(
		"access_token",
		accessToken,
		900,
		"/",
		"",
		false,
		true,
	)

	c.SetCookie(
		"refresh_token",
		refreshToken,
		604800,
		"/api/auth/refresh",
		"",
		false,
		true,
	)

	c.JSON(http.StatusOK, structs.SuccessResponse{
		Success: true,
		Message: "Login Success",
		Data: structs.UserResponse{
			Id:        user.Id,
			Name:      user.Name,
			Username:  user.Username,
			Email:     user.Email,
			CreatedAt: user.CreatedAt.String(),
			UpdatedAt: user.UpdatedAt.String(),
			Role:      user.Role,
			Picture:   user.Picture,
		},
	})
}

func Logout(c *gin.Context) {

	c.SetCookie(
		"access_token",
		"",
		-1,
		"/",
		"",
		false,
		true,
	)

	c.SetCookie(
		"refresh_token",
		"",
		-1,
		"/api/auth/refresh",
		"",
		false,
		true,
	)

	c.JSON(http.StatusOK, structs.SuccessResponse{
		Success: true,
		Message: "Logout success",
	})
}

// func GoogleLogin(c *gin.Context) {
// 	var req structs.GoogleLoginRequest

// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"message": "invalid request",
// 		})
// 		return
// 	}

// 	payload, err := idtoken.Validate(
// 		context.Background(),
// 		req.Credential,
// 		os.Getenv("GOOGLE_CLIENT_ID"),
// 	)

// 	if err != nil {
// 		c.JSON(http.StatusUnauthorized, gin.H{
// 			"message": "invalid google token",
// 		})
// 		return
// 	}

// 	var user = models.User{}

// 	email, ok := payload.Claims["email"].(string)
// 	if !ok {
// 		c.JSON(400, gin.H{
// 			"message": "invalid google payload",
// 		})
// 		return
// 	}
// 	name, ok := payload.Claims["name"].(string)
// 	if !ok {
// 		c.JSON(400, gin.H{
// 			"message": "invalid google payload",
// 		})
// 		return
// 	}
// 	picture, ok := payload.Claims["picture"].(string)
// 	if !ok {
// 		c.JSON(400, gin.H{
// 			"message": "invalid google payload",
// 		})
// 		return
// 	}
// 	googleID := payload.Subject

// 	result := database.DB.Where(
// 		"email = ?",
// 		email,
// 	).First(&user)

// 	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
// 		user = models.User{
// 			Email:      email,
// 			Name:       name,
// 			Picture:    picture,
// 			Provider:   "google",
// 			ProviderID: googleID,
// 		}

// 		if err := database.DB.Create(&user).Error; err != nil {
// 			c.JSON(http.StatusInternalServerError, structs.ErrorResponse{
// 				Success: false,
// 				Message: "Failed to create user",
// 			})
// 			return
// 		}
// 	} else if result.Error != nil {

// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"message": "database error",
// 		})
// 		return

// 	} else {
// 		updated := false
// 		if user.Provider == "" {
// 			user.Provider = "google"
// 			user.ProviderID = googleID
// 			updated = true
// 		}

// 		if user.Picture == "default.png" {
// 			user.Picture = picture
// 			updated = true
// 		}
// 		if updated {
// 			database.DB.Save(&user)
// 		}
// 	}

// 	accessToken := helpers.GenerateAccessToken(user.Id, user.Role)
// 	refreshToken := helpers.GenerateRefreshToken(user.Id, user.Role)

// 	c.SetCookie(
// 		"access_token",
// 		accessToken,
// 		900,
// 		"/",
// 		"",
// 		false,
// 		true,
// 	)

// 	c.SetCookie(
// 		"refresh_token",
// 		refreshToken,
// 		604800,
// 		"/api/auth/refresh",
// 		"",
// 		false,
// 		true,
// 	)

// 	c.JSON(http.StatusOK, structs.SuccessResponse{
// 		Success: true,
// 		Message: "Login Success",
// 		Data: structs.UserResponse{
// 			Id:        user.Id,
// 			Name:      user.Name,
// 			Username:  user.Username,
// 			Email:     user.Email,
// 			CreatedAt: user.CreatedAt.String(),
// 			UpdatedAt: user.UpdatedAt.String(),
// 			Role:      user.Role,
// 			Picture:   user.Picture,
// 		},
// 	})
// }

func GoogleLogin(c *gin.Context) {
	q := c.Request.URL.Query()
	q.Add("provider", "google")
	c.Request.URL.RawQuery = q.Encode()

	gothic.BeginAuthHandler(c.Writer, c.Request)
}

func GoogleCallback(c *gin.Context) {
	q := c.Request.URL.Query()
	q.Add("provider", "google")
	c.Request.URL.RawQuery = q.Encode()

	user, err := gothic.CompleteUserAuth(
		c.Writer,
		c.Request,
	)

	if err != nil {
		c.JSON(400, gin.H{
			"error": err.Error(),
		})
		return
	}

	var dbuser = models.User{}

	result := database.DB.Where(
		"email = ?",
		user.Email,
	).First(&dbuser)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		dbuser = models.User{
			Email:         user.Email,
			Name:          user.Name,
			Picture:       user.AvatarURL,
			Provider:      "google",
			ProviderID:    user.UserID,
			Username:      helpers.GenerateUsername(user.Email),
			EmailVerified: true,
		}

		if err := database.DB.Create(&dbuser).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "Failed to create user error",
				"error":   err.Error(),
			})
			return
		}
	} else if result.Error != nil {

		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "database error",
		})
		return

	} else {
		updated := false
		if dbuser.Provider == "" {
			dbuser.Provider = "google"
			dbuser.ProviderID = user.UserID
			updated = true
		}

		if dbuser.Picture == "default.png" {
			dbuser.Picture = user.AvatarURL
			updated = true
		}
		if updated {
			database.DB.Save(&dbuser)
		}
	}

	accessToken := helpers.GenerateAccessToken(dbuser.Id, dbuser.Role)
	refreshToken := helpers.GenerateRefreshToken(dbuser.Id, dbuser.Role)

	c.SetCookie(
		"access_token",
		accessToken,
		900,
		"/",
		"",
		false,
		true,
	)

	c.SetCookie(
		"refresh_token",
		refreshToken,
		604800,
		"/api/auth/refresh",
		"",
		false,
		true,
	)

	//
	frontendUrl := os.Getenv("DOMAIN")
	c.Redirect(
		http.StatusTemporaryRedirect,
		frontendUrl,
	)
}
