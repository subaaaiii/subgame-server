package models

import "time"

type Message struct {
	ID         uint
	FromUserID uint
	ToUserID   uint
	Message    string
	CreatedAt  time.Time

	ProductId *uint
	OrderId   *string `gorm:"type:varchar(36)" json:"order_id"`

	Product *Product `gorm:"foreignKey:ProductId"`
	Order   *Order   `gorm:"foreignKey:OrderId;references:Id"`

	FromUser User `gorm:"foreignKey:FromUserID"`
	ToUser   User `gorm:"foreignKey:ToUserID"`
}
