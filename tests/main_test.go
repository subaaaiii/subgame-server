package tests_test

import (
	"fmt"
	"log"
	"os"
	"testing"

	"bnsp2/server/config"
	"bnsp2/server/database"
	"bnsp2/server/redis"

	"github.com/alicebob/miniredis/v2"
)

func TestMain(m *testing.M) {
	os.Setenv("APP_ENV", "test")

	config.LoadEnv("../../.env")
	mr, err := miniredis.Run()
	if err != nil {
		log.Fatalf("Gagal menjalankan miniredis: %v", err)
	}

	os.Setenv("REDIS_ADDR", mr.Addr())

	fmt.Println("1. START TEST")

	redis.ConnectRedis()
	fmt.Println("2. REDIS OK")

	database.InitDB()
	fmt.Println("3. DATABASE OK")

	os.Setenv("MIDTRANS_SERVER_KEY", "dummy-server-key")
	config.InitMidtrans()
	fmt.Println("4. MIDTRANS OK")

	BersihkanDatabase()

	exitCode := m.Run()

	// BersihkanDatabase()

	mr.Close()
	os.Exit(exitCode)
}

func BersihkanDatabase() {
	database.DB.Exec("SET FOREIGN_KEY_CHECKS = 0;")

	var tables []string
	if err := database.DB.Raw("SHOW TABLES").Scan(&tables).Error; err != nil {
		log.Fatal("Gagal mengambil daftar tabel: ", err)
	}

	for _, table := range tables {
		database.DB.Exec("TRUNCATE TABLE " + table + ";")
	}
	database.DB.Exec("SET FOREIGN_KEY_CHECKS = 1;")
}
