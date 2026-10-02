package controllers

import (
	"bnsp2/server/database"
	"bnsp2/server/helpers"
	"bnsp2/server/models"
	"bnsp2/server/structs"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterSeller menangani proses registrasi seller baru
func RegisterSeller(c *gin.Context) {

	userId := c.MustGet("user_id").(uint)

	var user models.User
	if err := database.DB.First(&user, userId).Error; err != nil {
		c.JSON(http.StatusNotFound, structs.ErrorResponse{
			Success: false,
			Message: "User not found",
		})
		return
	}

	errors := map[string]string{}

	identityNumber := c.PostForm("identity_number")
	if identityNumber == "" {
		errors["IdentityNumber"] = "Identity number is required"
	}

	identityImage, err := c.FormFile("identity_image")
	if identityImage == nil {
		errors["IdentityImage"] = "Identity image is required"
	} else if err != nil {
		errors["IdentityImage"] = "Failed to retrieve identity image"
	}

	name := c.PostForm("name")
	if name == "" {
		errors["Name"] = "Name is required"
	}
	user.Name = name

	address := c.PostForm("address")
	if address == "" {
		errors["Address"] = "Address is required"
	}
	user.Address = address

	gender := c.PostForm("gender")
	if gender == "" {
		errors["Gender"] = "Gender is required"
	}
	user.Gender = gender

	birthday := c.PostForm("birthday")
	if birthday == "" {
		errors["Birthday"] = "Birthday is required"
	}
	if birthday != "" {
		parsedDate, err := time.Parse("2006-01-02", birthday)
		if err != nil {
			errors["Birthday"] = "Birthday must be in the format YYYY-MM-DD"
		}
		user.Birthday = &parsedDate
	}

	if len(errors) > 0 {
		c.JSON(http.StatusBadRequest, structs.ErrorResponse{
			Success: false,
			Message: "Validation failed",
			Errors:  errors,
		})
		return
	}

	uploadPath := "images/sellers/identities"
	if err := os.MkdirAll(uploadPath, os.ModePerm); err != nil {
		c.JSON(http.StatusInternalServerError, structs.ErrorResponse{
			Success: false,
			Message: "Failed to create upload directory",
		})
		return
	}
	filename := fmt.Sprintf("seller-%d%s", time.Now().Unix(), filepath.Ext(identityImage.Filename))
	filepath := filepath.Join(uploadPath, filename)

	if err := c.SaveUploadedFile(identityImage, filepath); err != nil {
		c.JSON(http.StatusInternalServerError, structs.ErrorResponse{
			Success: false,
			Message: "Failed to save identity image",
		})
		return
	}

	seller := models.Seller{
		IdentityNumber: identityNumber,
		IdentityImage:  filename,
		UserId:         userId,
	}

	// save database
	tx := database.DB.Begin()
	if err := tx.Model(&user).Updates(user).Error; err != nil {
		tx.Rollback()
		os.Remove(filepath)
		c.JSON(http.StatusInternalServerError, structs.ErrorResponse{
			Success: false,
			Message: "Failed to create user",
			Errors:  helpers.TranslateErrorMessage(err),
		})
		return
	}
	if err := tx.Create(&seller).Error; err != nil {
		tx.Rollback()
		os.Remove(filepath)
		c.JSON(http.StatusInternalServerError, structs.ErrorResponse{
			Success: false,
			Message: "Failed to create seller",
			Errors:  helpers.TranslateErrorMessage(err),
		})
		return
	}
	tx.Commit()

	// Jika berhasil, kirimkan response sukses
	c.JSON(http.StatusCreated, structs.SuccessResponse{
		Success: true,
		Message: "Seller created successfully",
		Data: structs.SellerResponse{
			Id:             seller.Id,
			IdentityNumber: seller.IdentityNumber,
			IdentityImage:  seller.IdentityImage,
			UserId:         seller.UserId,
			Status:         "pending",
			User: structs.UserProfileResponse{
				Name:     user.Name,
				Address:  user.Address,
				Gender:   user.Gender,
				Birthday: helpers.FormatBirthday(user.Birthday),
			},
		},
	})
}

func GetSellerByUserId(c *gin.Context) {
	userId := c.MustGet("user_id").(uint)

	var seller models.Seller
	if err := database.DB.Preload("User").Where("user_id = ?", userId).First(&seller).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, structs.ErrorResponse{
				Success: false,
				Message: "Seller not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, structs.ErrorResponse{
			Success: false,
			Message: "Failed to fetch seller",
		})
		return
	}

	c.JSON(http.StatusOK, structs.SuccessResponse{
		Success: true,
		Message: "Seller found",
		Data: structs.SellerResponse{
			Id:             seller.Id,
			IdentityNumber: seller.IdentityNumber,
			IdentityImage:  seller.IdentityImage,
			UserId:         seller.UserId,
			Status:         seller.Status,
			User: structs.UserProfileResponse{
				Name:     seller.User.Name,
				Address:  seller.User.Address,
				Gender:   seller.User.Gender,
				Birthday: helpers.FormatBirthday(seller.User.Birthday),
			},
		},
	})
}

func GetSellers(c *gin.Context) {

	var sellers []models.Seller
	if err := database.DB.Preload("User").Find(&sellers).Error; err != nil {
		c.JSON(http.StatusNotFound, structs.ErrorResponse{
			Success: false,
			Message: "Failed to create user",
			Errors:  helpers.TranslateErrorMessage(err),
		})
		return
	}

	var result []structs.SellerResponse

	for _, seller := range sellers {
		result = append(result, structs.SellerResponse{
			Id:             seller.Id,
			IdentityNumber: seller.IdentityNumber,
			IdentityImage:  seller.IdentityImage,
			UserId:         seller.UserId,
			Status:         seller.Status,
			User: structs.UserProfileResponse{
				Name:     seller.User.Name,
				Address:  seller.User.Address,
				Gender:   seller.User.Gender,
				Birthday: helpers.FormatBirthday(seller.User.Birthday),
			},
		})
	}

	c.JSON(http.StatusOK, structs.SuccessResponse{
		Success: true,
		Message: "Seller found",
		Data:    result,
	})
}

func UpdateSellerStatus(c *gin.Context) {
	var req structs.UpdateSellerStatusRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, structs.ErrorResponse{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	tx := database.DB.Begin()

	// 1. Update seller status
	if err := tx.Model(&models.Seller{}).
		Where("id IN ?", req.Ids).
		Update("status", req.Status).Error; err != nil {

		tx.Rollback()
		c.JSON(http.StatusInternalServerError, structs.ErrorResponse{
			Success: false,
			Message: "Failed to update seller status",
		})
		return
	}

	// 2. Kalau approved → update role user
	if req.Status == "approved" {

		// ambil user_id dari sellers
		var userIds []uint
		if err := tx.Model(&models.Seller{}).
			Where("id IN ?", req.Ids).
			Pluck("user_id", &userIds).Error; err != nil {

			tx.Rollback()
			c.JSON(http.StatusInternalServerError, structs.ErrorResponse{
				Success: false,
				Message: "Failed to get user ids",
			})
			return
		}

		// update role user jadi seller
		if err := tx.Model(&models.User{}).
			Where("id IN ?", userIds).
			Update("role", "seller").Error; err != nil {

			tx.Rollback()
			c.JSON(http.StatusInternalServerError, structs.ErrorResponse{
				Success: false,
				Message: "Failed to update user role",
			})
			return
		}
	}

	// commit kalau semua sukses
	tx.Commit()

	c.JSON(http.StatusOK, structs.SuccessResponse{
		Success: true,
		Message: "Status updated successfully",
	})
}
