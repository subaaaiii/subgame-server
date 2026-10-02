package controllers

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"

	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"bnsp2/server/database"
	"bnsp2/server/helpers"
	"bnsp2/server/models"
	"bnsp2/server/redis"
	"bnsp2/server/structs"

	"github.com/gin-gonic/gin"
)

func CreateGame(c *gin.Context) {

	errors := map[string]string{}

	name := c.PostForm("name")
	if name == "" {
		errors["Name"] = "Name is required"
	}

	fieldsStr := c.PostForm("fields")

	var fields []map[string]interface{}

	if fieldsStr != "" {
		if err := json.Unmarshal([]byte(fieldsStr), &fields); err != nil {
			c.JSON(http.StatusBadRequest, structs.ErrorResponse{
				Success: false,
				Message: "invalid fields format",
			})
			return
		}
	}

	file, err := c.FormFile("image")
	if file == nil {
		errors["Image"] = "Image is required"
	} else if err != nil {
		errors["Image"] = "Failed to retrieve image"
	}

	if len(errors) > 0 {
		c.JSON(http.StatusBadRequest, structs.ErrorResponse{
			Success: false,
			Message: "Validation failed",
			Errors:  errors,
		})
		return
	}

	uploadPath := "./images/games/covers/"
	os.MkdirAll(uploadPath, os.ModePerm)

	filename := fmt.Sprintf("%d%s", time.Now().Unix(), filepath.Ext(file.Filename))
	filepath := path.Join(uploadPath, filename)

	if err := c.SaveUploadedFile(file, filepath); err != nil {
		c.JSON(http.StatusInternalServerError, structs.ErrorResponse{
			Success: false,
			Message: "failed to save image",
		})
		return
	}
	fieldJSON, err := json.Marshal(fields)
	if err != nil {
		c.JSON(http.StatusInternalServerError, structs.ErrorResponse{
			Success: false,
			Message: "failed to parse fields",
		})
		os.Remove(filepath)
		return
	}

	game := models.Game{
		Name:        name,
		Image:       filename,
		FieldSchema: fieldJSON,
	}

	if err := database.DB.Create(&game).Error; err != nil {
		c.JSON(http.StatusInternalServerError, structs.ErrorResponse{
			Success: false,
			Message: "failed to create game",
		})
		os.Remove(filepath)
		return
	}

	ctx := c.Request.Context()

	helpers.DeleteByPattern(
		ctx,
		"games:*",
	)

	c.JSON(http.StatusOK, structs.SuccessResponse{
		Success: true,
		Message: "Game successfully created",
		Data:    game,
	})
}

type GameResponse struct {
	Data []models.Game `json:"data"`
	Meta gin.H         `json:"meta"`
}

func GetGames(c *gin.Context) {
	var games []models.Game
	query := database.DB.Model(&models.Game{})
	q := c.Query("q")

	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	page, _ := strconv.Atoi(pageStr)
	limit, _ := strconv.Atoi(limitStr)

	if page < 1 {
		page = 1
	}
	if limit < 0 {
		limit = 10
	}

	ctx := c.Request.Context()

	cacheable := page == 1 && q == ""

	var key string
	if cacheable {
		key = fmt.Sprintf("games:limit:%v", limit)
		var cachedResp GameResponse
		if cached, err := redis.RedisClient.Get(ctx, key).Result(); err == nil {
			if err := json.Unmarshal([]byte(cached), &cachedResp); err == nil {
				c.JSON(http.StatusOK, gin.H{
					"success": true,
					"message": "Games retrieved from cache",
					"data":    cachedResp.Data,
					"meta":    cachedResp.Meta,
				})
				fmt.Printf("cache games hit | key=%s\n", key)
				return
			}
		}
	}

	offset := (page - 1) * limit
	var total int64

	if q != "" {
		query = query.Where("LOWER(name) LIKE ?", "%"+strings.ToLower(q)+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, structs.ErrorResponse{
			Success: false,
			Message: "failed to count products",
		})
		return
	}
	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	if err := query.Limit(limit).
		Offset(offset).Find(&games).Error; err != nil {
		c.JSON(http.StatusInternalServerError, structs.ErrorResponse{
			Success: false,
			Message: "failed to fetch games",
		})
		return
	}

	resp := GameResponse{
		Data: games,
		Meta: gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
		},
	}

	if cacheable {
		if data, err := json.Marshal(resp); err == nil {
			redis.RedisClient.Set(ctx, key, data, 24*time.Hour)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Games retrieved successfully",
		"data":    games,
		"meta": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

type GameByIdResponse struct {
	ID     uint                     `json:"id"`
	Name   string                   `json:"name"`
	Image  string                   `json:"image"`
	Fields []map[string]interface{} `json:"fields"`
}

func GetGameByID(c *gin.Context) {
	id := c.Param("id")

	var game models.Game

	ctx := c.Request.Context()

	var key string
	key = fmt.Sprintf("game:%v", id)
	var cachedResp GameByIdResponse
	if cached, err := redis.RedisClient.Get(ctx, key).Result(); err == nil {
		if err := json.Unmarshal([]byte(cached), &cachedResp); err == nil {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"message": "Game retrieved from cache",
				"data":    cachedResp,
			})
			fmt.Printf("cache game hit | key=%s\n", key)
			return
		}
		redis.RedisClient.Del(ctx, key)
	}

	if err := database.DB.First(&game, id).Error; err != nil {
		c.JSON(http.StatusNotFound, structs.ErrorResponse{
			Success: false,
			Message: "game not found",
		})
		return
	}

	var fields []map[string]interface{}
	json.Unmarshal(game.FieldSchema, &fields)

	resp := GameByIdResponse{
		ID:     game.ID,
		Name:   game.Name,
		Image:  game.Image,
		Fields: fields,
	}

	if data, err := json.Marshal(resp); err == nil {
		redis.RedisClient.Set(ctx, key, data, 24*time.Hour)
	}

	c.JSON(http.StatusOK, structs.SuccessResponse{
		Success: true,
		Data:    resp,
	})
}

func UpdateGame(c *gin.Context) {

	id := c.Param("id")

	var game models.Game

	if err := database.DB.First(&game, id).Error; err != nil {
		c.JSON(http.StatusNotFound, structs.ErrorResponse{
			Success: false,
			Message: "Game not found",
		})
		return
	}

	name := c.PostForm("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, structs.ErrorResponse{
			Success: false,
			Message: "name is required",
		})
		return
	}
	game.Name = name

	fieldsStr := c.PostForm("fields")

	var fields []map[string]interface{}

	if fieldsStr != "" {
		if err := json.Unmarshal([]byte(fieldsStr), &fields); err != nil {
			c.JSON(http.StatusBadRequest, structs.ErrorResponse{
				Success: false,
				Message: "invalid fields format",
			})
			return
		}
		fieldsJSON, _ := json.Marshal(fields)
		game.FieldSchema = fieldsJSON
	}

	file, err := c.FormFile("image")
	if err != nil {
		if !errors.Is(err, http.ErrMissingFile) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file upload"})
			return
		}
	} else {
		ext := strings.ToLower(filepath.Ext(file.Filename))

		if ext != ".jpg" && ext != ".png" && ext != ".webp" && ext != ".jpeg" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file type"})
			return
		}

		if game.Image != "" {
			oldPath := "images/games/covers/" + game.Image
			_ = os.Remove(oldPath)
		}

		filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
		path := "images/games/covers/" + filename

		if err := c.SaveUploadedFile(file, path); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload picture"})
			return
		}

		game.Image = filename
	}

	if err := database.DB.Save(&game).Error; err != nil {
		c.JSON(http.StatusInternalServerError, structs.ErrorResponse{
			Success: false,
			Message: "Failed to update game",
		})
		return
	}

	ctx := c.Request.Context()

	helpers.DeleteByPattern(
		ctx,
		"games:*",
	)

	c.JSON(http.StatusOK, structs.SuccessResponse{
		Success: true,
		Message: "Game updated successfully",
		Data:    game,
	})
}

func DeleteGame(c *gin.Context) {
	id := c.Param("id")
	var game models.Game
	if err := database.DB.First(&game, id).Error; err != nil {
		c.JSON(http.StatusNotFound, structs.ErrorResponse{
			Success: false,
			Message: "Game not found",
		})
		return
	}
	if game.Image != "" {
		oldPath := "images/games/covers/" + game.Image
		_ = os.Remove(oldPath)
	}
	if err := database.DB.Delete(&game).Error; err != nil {
		c.JSON(http.StatusInternalServerError, structs.ErrorResponse{
			Success: false,
			Message: "Failed to delete game, server error",
		})
	}
	ctx := c.Request.Context()

	helpers.DeleteByPattern(
		ctx,
		"games:*",
	)
	c.JSON(http.StatusOK, structs.SuccessResponse{
		Success: true,
		Message: "Game deleted successfully",
	})
}
