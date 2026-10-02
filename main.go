package main

import (
	"bnsp2/server/config"
	"bnsp2/server/database"
	"bnsp2/server/redis"
	"bnsp2/server/routes"
	"bnsp2/server/services"
)

func main() {

	//load config .env
	config.LoadEnv()
	config.InitSession()
	config.InitOAuth()
	config.InitMidtrans()
	redis.ConnectRedis()
	database.InitDB()

	r := routes.SetupRouter()
	services.StartExpireJob()

	//mulai server
	r.Run(":" + config.GetEnv("APP_PORT", "3000"))
}
