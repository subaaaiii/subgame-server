package tests_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"bnsp2/server/controllers"
	"bnsp2/server/redis"
	"bnsp2/server/routes"

	"github.com/gin-gonic/gin"
)

var authToken string

func TestAuthenticationFlow(t *testing.T) {

	gin.SetMode(gin.TestMode)
	router := routes.SetupRouter()

	testUser := map[string]string{
		"name":     "Test User",
		"username": "test",
		"email":    "testuser_integration@example.com",
		"password": "password123",
	}

	t.Run("1. Register User Baru", func(t *testing.T) {

		original := controllers.SendOTPEmail

		controllers.SendOTPEmail = func(email, otp string) error {
			return nil
		}

		defer func() {
			controllers.SendOTPEmail = original
		}()

		body, _ := json.Marshal(testUser)
		req, _ := http.NewRequest("POST", "/api/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Register gagal. Expected 200, got %d. Body: %s", w.Code, w.Body.String())
		}
	})
	t.Run("2. Create user", func(t *testing.T) {
		email := testUser["email"]
		ctx := context.Background()
		redisKey := fmt.Sprintf("register:%s", email)

		result, err := redis.RedisClient.Get(ctx, redisKey).Result()
		if err != nil {
			t.Fatalf("Data tidak ditemukan di Redis! Apakah langkah 1 (/api/register) berhasil? Error: %v", err)
		}

		// EKSTRAK OTP ASLI
		var temp map[string]interface{}
		if err := json.Unmarshal([]byte(result), &temp); err != nil {
			t.Fatalf("Gagal membaca JSON dari Redis: %v", err)
		}

		otpAsli, ok := temp["otp"].(string)
		if !ok {
			t.Fatalf("Gagal membaca JSON otp")
		}

		// KIRIM REQUEST KE /api/users MENGGUNAKAN OTP ASLI
		requestBody := map[string]string{
			"email": email,
			"otp":   otpAsli,
		}
		body, _ := json.Marshal(requestBody)

		req, _ := http.NewRequest("POST", "/api/users", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// ASSERTION
		if w.Code != http.StatusCreated {
			t.Errorf("Harusnya sukses (201), tapi dapat status %d. Body: %s", w.Code, w.Body.String())
		}

		// PASTIKAN REDIS DIHAPUS OLEH CONTROLLER
		_, err = redis.RedisClient.Get(ctx, redisKey).Result()
		if err == nil {
			t.Errorf("Harusnya key '%s' dihapus dari Redis setelah sukses, tapi masih ada!", redisKey)
		}
	})

	var authCookie *http.Cookie
	t.Run("3. Login User", func(t *testing.T) {

		loginData := map[string]string{
			"username": testUser["username"],
			"password": testUser["password"],
		}
		body, _ := json.Marshal(loginData)
		req, _ := http.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Login gagal. Expected 200, got %d", w.Code)
		}

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Gagal parse response login: %v", err)
		}

		cookies := w.Result().Cookies()

		var cookieDitemukan bool
		for _, c := range cookies {
			if c.Name == "access_token" {
				authCookie = c
				cookieDitemukan = true
				break
			}
		}

		if !cookieDitemukan {
			t.Fatalf("HttpOnly Cookie 'token' tidak ditemukan pada header response!")
		}
	})

	t.Run("4. Get My Profile", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/me", nil)

		req.AddCookie(authCookie)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Akses /api/me gagal. Expected 200, got %d. Body: %s", w.Code, w.Body.String())
		}
	})
}
