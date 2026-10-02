package helpers

import (
	"bnsp2/server/config"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var accessKey = config.GetJWTAccessKey()
var refreshKey = config.GetJWTRefreshKey()

func GenerateAccessToken(user_id uint, role string) string {

	expirationTime := time.Now().Add(15 * time.Minute)

	claims := &JWTClaims{
		UserId: user_id,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(accessKey)

	return token
}

func GenerateRefreshToken(user_id uint, role string) string {

	expirationTime := time.Now().Add(7 * 24 * time.Hour)

	claims := &JWTClaims{
		UserId: user_id,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(refreshKey)

	return token
}
func GenerateResetToken(user_id uint) string {

	expirationTime := time.Now().Add(10 * time.Minute)

	claims := &JWTClaims{
		UserId: user_id,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(refreshKey)

	return token
}
