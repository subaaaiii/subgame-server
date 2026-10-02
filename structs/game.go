package structs

import "gorm.io/datatypes"

type Game struct {
	ID          uint
	Name        string
	Image       string
	FieldSchema datatypes.JSON
}

type Product struct {
	ID     uint
	GameID uint
	Data   datatypes.JSON
}

type Field struct {
	Name     string `json:"name"`
	Type     string `json:"type"` // text, number, etc
	Required bool   `json:"required"`
}

type CreateGameRequest struct {
	Name   string  `json:"name" binding:"required"`
	Fields []Field `json:"fields" binding:"required"`
}
