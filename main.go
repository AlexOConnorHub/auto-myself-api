package main

import (
	"auto-myself-api/database"
	"auto-myself-api/routes"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

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
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	database.Init()

	r := gin.Default()
	r.TrustedPlatform = gin.PlatformCloudflare
	r.SetTrustedProxies(nil)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
	routes.SetupRoutes(r)

	r.Run(":8080")
}
