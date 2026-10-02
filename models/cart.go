package models

import "time"

type Cart struct {
	Id        uint `gorm:"primaryKey" json:"id"`
	UserId    uint `gorm:"not null" json:"user_id"`
	ProductId uint `gorm:"not null" json:"product_id"`
	Quantity  int  `gorm:"not null" json:"quantity"`
	Total     int  `gorm:"not null" json:"total"`

	User    User    `gorm:"foreignKey:UserId" json:"user"`
	Product Product `gorm:"foreignKey:ProductId" json:"product"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
