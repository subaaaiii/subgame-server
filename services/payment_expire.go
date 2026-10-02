package services

import (
	"bnsp2/server/database"
	"bnsp2/server/models"
	"time"
)

func StartExpireJob() {
	go func() {
		for {
			time.Sleep(1 * time.Minute)

			var orders []models.Order

			database.DB.Where("status = ? AND created_at < ?",
				"PENDING",
				time.Now().Add(-15*time.Minute),
			).Find(&orders)

			for _, order := range orders {
				order.Status = "EXPIRED"
				database.DB.Save(&order)
			}
		}
	}()
}
