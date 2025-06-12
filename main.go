package main

import (
	"auto-myself-server/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()
	router.TrustedPlatform = gin.PlatformCloudflare

	router.GET("/favicon.ico", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	routes.SetupRoutes(router)

	router.Run(":8080") // listen and serve on 0.0.0.0:8080
}
