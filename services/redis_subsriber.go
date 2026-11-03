package services

import (
	"bnsp2/server/database"
	"bnsp2/server/redis"
	"context"
	"fmt"
	"log"
	"strings"
)

func StartRedisSubscriber() {
	ctx := context.Background()

	pubsub := redis.RedisClient.Subscribe(ctx, "__keyevent@0__:expired")

	go func() {
		defer pubsub.Close()
		log.Println("[Redis] Subscriber berjalan, mendengarkan key expired...")

		for {
			msg, err := pubsub.ReceiveMessage(ctx)
			if err != nil {
				log.Printf("[Redis] Error menerima pesan: %v\n", err)
				continue
			}

			key := msg.Payload

			if strings.HasPrefix(key, "order_pending:") {

				parts := strings.Split(key, ":")
				if len(parts) == 2 {
					orderID := parts[1]

					fmt.Println("mengeksekusi perubahan order dengan id:", orderID)
					err := database.DB.Exec(
						"UPDATE orders SET status = 'EXPIRED' WHERE id = ? AND status = 'PENDING'",
						orderID,
					).Error

					if err != nil {
						log.Printf("[ExpireJob] Gagal membatalkan order ID %s: %v\n", orderID, err)
					} else {
						log.Printf("[ExpireJob] Order ID %s otomatis EXPIRED\n", orderID)
					}
				}
			}
		}
	}()
}
