package helpers

import (
	"bnsp2/server/database"
	"bnsp2/server/models"
	"fmt"
	"strings"
)

func existsUsername(username string) bool {
	var count int64

	result := database.DB.
		Model(&models.User{}).
		Where("username = ?", username).
		Count(&count)

	if result.Error != nil {
		return true // anggap sudah ada kalau query gagal
	}

	return count > 0
}

func GenerateUsername(email string) string {

	base := strings.Split(email, "@")[0]

	username := base

	counter := 1

	for existsUsername(username) {
		username = fmt.Sprintf(
			"%s%d",
			base,
			counter,
		)

		counter++
	}

	return username
}
