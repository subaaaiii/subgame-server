package models

type Seller struct {
	Id             uint   `gorm:"primaryKey" json:"id"`
	IdentityNumber string `gorm:"size:255;not null" json:"identity_number"`
	IdentityImage  string `gorm:"size:255;not null" json:"identity_image"`
	Status         string `gorm:"size:50;not null;default:'pending'" json:"status"`
	UserId         uint   `gorm:"not null" json:"user_id"`
	User           User   `gorm:"foreignKey:UserId" json:"user"`
}
