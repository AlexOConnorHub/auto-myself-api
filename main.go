package main

import (
	"auto-myself-server/database"
	"auto-myself-server/routes"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	database.Init()

	r := gin.Default()
	r.TrustedPlatform = gin.PlatformCloudflare
	r.SetTrustedProxies(nil)

	routes.SetupRoutes(r)

	r.Run(":8080")
}
