package helpers

import (
	"encoding/json"
)

type FieldSchema struct {
	Label    string   `json:"label"`
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Options  []string `json:"options"`
	Required bool     `json:"required"`
}

func ValidateFieldValues(schemaJSON []byte, values map[string]interface{}) map[string]string {
	errors := make(map[string]string)

	var schemas []FieldSchema
	json.Unmarshal(schemaJSON, &schemas)

	for _, s := range schemas {
		v, exists := values[s.Name]

		if s.Required && (!exists || v == nil || v == "") {
			errors[s.Name] = s.Name + " is required"
		}
	}

	return errors
}
