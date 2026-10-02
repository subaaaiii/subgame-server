package database

import (
	"bnsp2/server/config"
	"bnsp2/server/models"
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {

	env := config.GetEnv("APP_ENV", "")

	dbName := config.GetEnv("DB_NAME", "")

	if env == "test" {
		dbName = config.GetEnv("DB_TEST", "db_bnsp_test")
	}
	dbUser := config.GetEnv("DB_USER", "root")
	dbPass := config.GetEnv("DB_PASS", "root")
	dbHost := config.GetEnv("DB_HOST", "localhost")
	dbPort := config.GetEnv("DB_PORT", "3306")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dbUser, dbPass, dbHost, dbPort, dbName)

	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	fmt.Println("Database connected successfully!")

	err = DB.AutoMigrate(&models.User{}, &models.Product{}, &models.Order{}, &models.Cart{}, &models.Game{}, &models.Seller{}, &models.Message{}, models.OrderLog{})
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	fmt.Println("Database migrated successfully!")
}
