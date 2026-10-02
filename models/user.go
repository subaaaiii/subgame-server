package models

import "time"

type User struct {
	Id            uint       `gorm:"primaryKey" json:"id"`
	Name          string     `gorm:"size:100;not null" json:"name"`
	Username      string     `gorm:"size:100;uniqueIndex;not null" json:"username"`
	Email         string     `gorm:"size:100;uniqueIndex;not null" json:"email"`
	Password      string     `gorm:"size:255;not null" json:"password"`
	Birthday      *time.Time `gorm:"type:date" json:"birthday"`
	Gender        string     `gorm:"size:20" json:"gender"`
	AccountNumber string     `gorm:"size:50" json:"account_number"`
	Bank          string     `gorm:"size:50" json:"bank"`
	Address       string     `gorm:"type:text" json:"address"`
	Role          string     `gorm:"size:20;default:'customer'" json:"role"`
	Picture       string     `gorm:"size:255" json:"picture"`
	// EmailOTP            string     `gorm:"size:10" json:"email_otp"`
	EmailVerified bool `gorm:"default:false" json:"email_verified"`
	// OTPExpiresAt        *time.Time `json:"otp_expires_at"`
	Products []Product `json:"products,omitempty"`
	// ResetToken          string     `gorm:"size:255" json:"reset_token"`
	// ResetTokenExpiresAt *time.Time `json:"reset_token_expires_at"`
	Provider   string `gorm:"type:varchar(255)" json:"provider"`
	ProviderID string `gorm:"type:varchar(255)" json:"provider_id"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
