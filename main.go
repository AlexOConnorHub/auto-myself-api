package main

import (
	"auto-myself-api/components"
	"auto-myself-api/database"
	"auto-myself-api/routes"
	"log"
	"time"

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

	// if err := components.SendEmailRaw("no-reply@automyself.com", "Test email from Auto Myself API", "<h1>This is a test email from the Auto Myself API</h1><p>If you received this email, the email system is working!</p>"); err != nil {
	// 	log.Fatalf("Error sending test email: %v", err)
	// }
	// time.Sleep(time.Second)

	if err := components.ListTemplates(); err != nil {
		log.Fatalf("Error listing templates: %v", err)
	}
	time.Sleep(time.Second)

	// if err := components.CreateTemplate("confirm_email", "Auto Myself - Confirm your email",
	// 	"<div><p>Thank you for signing up! Please confirm your email address by entering the code below:</p><h1 style=\"font-size: 36px; font-weight: bold;\">{{code}}</h1><p>If you didn't sign up for this account, you can safely ignore this email.</p></div>"); err != nil {
	// 	log.Fatalf("Error creating template: %v", err)
	// }
	// time.Sleep(time.Second)

	// if err := components.ListTemplates(); err != nil {
	// 	log.Fatalf("Error listing templates: %v", err)
	// }
	// time.Sleep(time.Second)

	// if err := components.SendEmailTemplate("no-reply@automyself.com", "confirm_email", `{"code":"123456"}`); err != nil {
	// 	log.Fatalf("Error sending email: %v", err)
	// }
	// time.Sleep(time.Second)

	// if err := components.UpdateTemplate("test_email", "Auto Myself - Test Email",
	// 	"<h1>This is a test email from the Auto Myself API</h1><p>If you received this email, the email system is working!</p>"); err != nil {
	// 	log.Fatalf("Error creating template: %v", err)
	// }
	// time.Sleep(time.Second)

	// if err := components.DeleteTemplate("test_email"); err != nil {
	// 	log.Fatalf("Error deleting template: %v", err)
	// }
	// time.Sleep(time.Second)

	// if err := components.ListTemplates(); err != nil {
	// 	log.Fatalf("Error listing templates: %v", err)
	// }
	// time.Sleep(time.Second)

	if err := components.SendEmailTemplate("no-reply@automyself.com", "test_email", `{}`); err != nil {
		log.Fatalf("Error sending email: %v", err)
	}
	return

	database.Init()

	r := gin.Default()
	r.TrustedPlatform = gin.PlatformCloudflare
	r.SetTrustedProxies(nil)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
	routes.SetupRoutes(r)

	r.Run(":8080")
}
