package main

import (
	"auto-myself-api/controllers"
	"auto-myself-api/database"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/joho/godotenv/autoload"
)

// @title			Auto Myself API
// @version		1.0
// @description	This API is used to store and share vehicle maintenance history.
// @termsOfService	https://github.com/AlexOConnorHub/auto-myself-api/blob/main/TERMS_OF_SERVICE.md
// @contact.name	AlexOConnorHub
// @contact.url	https://automyself.com
// @contact.email	api@automyself.com
// @license.name	FSL-1.1-ALv2
// @license.url	https://github.com/AlexOConnorHub/auto-myself-api/blob/main/LICENSE
// @servers.url			http://localhost.automyself.com:8080/v1/
// @servers.description	Local development server
// @servers.url			https://api.automyself.com/v1/
// @servers.description	Production server
// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
// @description				Type "Bearer" followed by a space and JWT token.
func main() {
	database.Init()

	r := gin.Default()
	r.TrustedPlatform = gin.PlatformCloudflare

	controllers.SetupRoutes(r)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: r.Handler(),
	}
	go func() {
		// service connections
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal, 1)
	// kill (no params) by default sends syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be caught, so don't need add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutdown Server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Println("Server Shutdown:", err)
	}
	log.Println("Server exiting")

}
