package tests_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bnsp2/server/middlewares" // Sesuaikan dengan module proyek Anda

	"github.com/gin-gonic/gin"
)

func TestAuthMiddleware_TanpaToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()

	router.GET("/rahasia", middlewares.AuthMiddleware(), func(c *gin.Context) {
		c.String(http.StatusOK, "sukses")
	})

	req, _ := http.NewRequest("GET", "/rahasia", nil)

	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Gagal: Harusnya ditolak dengan 401 Unauthorized, tapi malah dapat status %d", w.Code)
	}
}

func TestRateLimitMiddleware_DiblokirJikaSpam(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.GET("/spam", middlewares.RateLimit(2, time.Minute), func(c *gin.Context) {
		c.String(http.StatusOK, "sukses")
	})

	req1, _ := http.NewRequest("GET", "/spam", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	req2, _ := http.NewRequest("GET", "/spam", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	req3, _ := http.NewRequest("GET", "/spam", nil)
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)

	if w3.Code != http.StatusTooManyRequests {
		t.Errorf("Gagal: Harusnya kena blokir (429 Too Many Requests), tapi dapat status %d", w3.Code)
	}
}
