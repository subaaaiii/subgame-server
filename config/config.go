package config

import (
	"fmt"
	"log"
	"os"

	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/google"
	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/coreapi"
	"github.com/midtrans/midtrans-go/snap"
)

func LoadEnv(filenames ...string) {
	err := godotenv.Load(filenames...)
	if err != nil {
		log.Println("Warning: No .env file found, using system environment variables")
	}
}

func GetEnv(key string, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	return value
}

func GetJWTAccessKey() []byte {
	return []byte(GetEnv("JWT_ACCESS_SECRET", "secret"))
}

func GetJWTRefreshKey() []byte {
	return []byte(GetEnv("JWT_REFRESH_SECRET", "secret2"))
}

func InitOAuth() {
	goth.UseProviders(
		google.New(
			os.Getenv("GOOGLE_CLIENT_ID"),
			os.Getenv("GOOGLE_CLIENT_SECRET"),
			os.Getenv("GOOGLE_CALLBACK_URL"),
			"email",
			"profile",
		),
	)
}

func InitSession() {

	secret := os.Getenv("SESSION_SECRET")

	fmt.Println(
		"SESSION SECRET:",
		secret,
	)
	store := sessions.NewCookieStore(
		[]byte(os.Getenv("SESSION_SECRET")),
	)

	gothic.Store = store
}

var SnapClient snap.Client
var CoreAPIClient coreapi.Client

func InitMidtrans() {
	serverKey := os.Getenv("MIDTRANS_SERVER_KEY")
	SnapClient.New(serverKey, midtrans.Sandbox)
	CoreAPIClient.New(serverKey, midtrans.Sandbox)
}
