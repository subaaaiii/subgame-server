package helpers

import "github.com/golang-jwt/jwt/v5"

type JWTClaims struct {
	UserId uint   `json:"id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}
