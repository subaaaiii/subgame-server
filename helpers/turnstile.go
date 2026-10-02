package helpers

import (
	"bnsp2/server/config"
	"encoding/json"
	"net/http"
	"net/url"
)

func VerifyTurnstile(token string) (bool, error) {
	form := url.Values{}

	form.Set("secret", config.GetEnv("TURNSTILE_SECRET_KEY", "wrong"))
	form.Set("response", token)

	resp, err := http.PostForm(
		"https://challenges.cloudflare.com/turnstile/v0/siteverify",
		form,
	)

	if err != nil {
		return false, err
	}

	defer resp.Body.Close()

	var result struct {
		Success bool `json:"success"`
	}

	err = json.NewDecoder(resp.Body).Decode(&result)

	if err != nil {
		return false, err
	}

	return result.Success, nil
}
