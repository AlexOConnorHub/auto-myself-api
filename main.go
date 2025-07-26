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

	_ "auto-myself-api/docs"

	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

//	@title									Auto Myself API
//	@version								1.0
//	@description							This API is used to store and share vehicle maintenance history.
//	@termsOfService							https://github.com/AlexOConnorHub/auto-myself-api/blob/main/TERMS_OF_SERVICE.md

//	@contact.name							AlexOConnorHub
//	@contact.url							https://automyself.com
//	@contact.email							api@automyself.com

//	@license.name							CC BY-NC-SA 4.0
//	@license.url							https://creativecommons.org/licenses/by-nc-sa/4.0/

//	@servers.url							http://localhost:8080/
//	@servers.description					Local development server
//	@servers.url							https://api.automyself.com/v1/
//	@servers.description					Production server

//	// @securitydefinitions.oauth2.application	OAuth2Application
//	// @tokenUrl								https://auth0.automyself.com/oauth/token
//	// @in 									header
//	// @name									Authorization
//	// @scope.admin							Grants read and write access to administrative information
//	// @scope.user_id							Provides information to associate the authenticated user and data for that user
//
//	// @securitydefinitions.oauth2.implicit	OAuth2Implicit
//	// @authorizationUrl						https://auth0.automyself.com/oauth/authorize
//	// @in 									header
//	// @name									Authorization
//	// @scope.admin							Grants read and write access to administrative information
//	// @scope.user_id							Provides information to associate the authenticated user and data for that user

func main() {
	database.Init()

	r := gin.Default()
	r.TrustedPlatform = gin.PlatformCloudflare

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
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
