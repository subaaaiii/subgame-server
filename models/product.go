package models

import (
	"time"

	"gorm.io/datatypes"
)

type Product struct {
	Id          uint           `gorm:"primaryKey" json:"id"`
	GameId      uint           `gorm:"not null" json:"game_id"`
	UserId      uint           `gorm:"not null" json:"user_id"`
	Price       int            `gorm:"not null" json:"price"`
	Title       string         `gorm:"size:255" json:"title"`
	Description string         `gorm:"type:text" json:"description"`
	Status      string         `gorm:"size:50;not null" json:"status"`
	Image       string         `gorm:"size:255" json:"image"`
	Stock       int            `gorm:"not null" json:"stock"`
	Guarantee   int            `gorm:"not null" json:"guarantee"`
	FieldValues datatypes.JSON `gorm:"type:json" json:"field_values"`
	RemoveImage bool           `gorm:"-" json:"remove_image,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`

	User User `gorm:"foreignKey:UserId" json:"user"`
	Game Game `gorm:"foreignKey:GameId" json:"game"`
}
