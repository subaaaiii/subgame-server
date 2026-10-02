package models

import "gorm.io/datatypes"

type Game struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name"`
	Image       string         `json:"image"`
	FieldSchema datatypes.JSON `json:"field_schema"` // dynamic fields

	Products []Product `json:"products,omitempty"`
}
