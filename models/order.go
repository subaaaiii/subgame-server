package models

import "time"

type Order struct {
	Id        string `gorm:"type:varchar(36);primaryKey" json:"id"`
	Invoice   string `gorm:"not null" json:"invoice"`
	UserId    uint   `gorm:"not null" json:"user_id"`
	SellerId  uint   `gorm:"not null" json:"seller_id"`
	ProductId uint   `gorm:"not null" json:"product_id"`
	Qty       int    `gorm:"not null" json:"qty"`
	Total     int    `gorm:"not null" json:"total"`
	Status    string `gorm:"not null" json:"status"`

	Seller   User       `gorm:"foreignKey:SellerId" json:"seller"`
	User     User       `gorm:"foreignKey:UserId" json:"user"`
	Product  Product    `gorm:"foreignKey:ProductId" json:"product"`
	OrderLog []OrderLog `gorm:"foreignKey:OrderId" json:"order_log"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
