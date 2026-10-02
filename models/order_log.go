package models

import "time"

type OrderLog struct {
	Id      uint   `gorm:"primaryKey" json:"id"`
	OrderId string `gorm:"type:varchar(36);not null" json:"order_id"`
	Status  string `gorm:"not null" json:"status"`
	Title   string `gorm:"not null" json:"title"`

	Order Order `gorm:"foreignKey:OrderId" json:"order"`

	CreatedAt time.Time `json:"created_at"`
}
