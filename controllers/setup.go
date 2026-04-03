package controllers

import (
	"auto-myself-api/app"
	"auto-myself-api/database"
	"auto-myself-api/helpers"
	"auto-myself-api/middleware"
	"testing"

	"github.com/gin-gonic/gin"
)

func WithApp(a *app.App, h func(*gin.Context, *app.App)) gin.HandlerFunc {
	return func(c *gin.Context) {
		h(c, a)
	}
}

func SetupRoutes(r *gin.Engine, a *app.App) {
	if gin.Mode() == gin.ReleaseMode {
		r.Use(middleware.RateLimitMiddleware())
	}

	auth := r.Group("/auth")
	{
		auth.GET("/:provider", WithApp(a, LoginWebProvider))
		auth.GET("/callback/:provider", WithApp(a, LoginWebCallback))
		// auth.POST("/logout", )
		auth.POST("/exchange/:provider", WithApp(a, LoginExchangeProvider))
		auth.POST("/development", WithApp(a, LoginDevelopment))
		auth.POST("/refresh", WithApp(a, Refresh))
	}

	v1 := r.Group("/v1")
	v1.Use(middleware.AuthMiddleware(a))
	v1.Use(middleware.ParseUUIDMiddleware())
	{
		uuidPath := "/:uuid"
		user := v1.Group("/user")
		{
			user.GET("", WithApp(a, GetCurrentUser))
			user.PATCH("", WithApp(a, UpdateCurrentUser))
			// user.DELETE("", WithApp(a, DeleteCurrentUser))
			uuidGroup := user.Group(uuidPath)
			{

				uuidGroup.GET("", WithApp(a, GetUserByID))
			}
		}

		vehicle := v1.Group("/vehicle")
		{
			vehicle.GET("", WithApp(a, GetAllVehicles))
			vehicle.POST("", WithApp(a, CreateVehicle))
			uuidGroup := vehicle.Group(uuidPath)
			{
				uuidGroup.GET("", WithApp(a, GetVehicleByID))
				uuidGroup.POST("", WithApp(a, CreateVehicle))
				uuidGroup.PATCH("", WithApp(a, UpdateVehicleByID))
				uuidGroup.DELETE("", WithApp(a, DeleteVehicleByID))
				uuidGroup.GET("/maintenance", WithApp(a, GetAllMaintenance))
			}
		}

		maintenance := v1.Group("/maintenance")
		{
			maintenance.POST("", WithApp(a, CreateMaintenance))
			uuidGroup := maintenance.Group(uuidPath)
			{
				uuidGroup.GET("", WithApp(a, GetMaintenanceByID))
				uuidGroup.POST("", WithApp(a, CreateMaintenance))
				uuidGroup.PATCH("", WithApp(a, UpdateMaintenanceByID))
				uuidGroup.DELETE("", WithApp(a, DeleteMaintenanceByID))
			}
		}

		share := v1.Group("/share")
		{
			share.GET("", WithApp(a, GetAllShares))
			share.POST("", WithApp(a, CreateShare))
			uuidGroup := share.Group(uuidPath)
			{
				uuidGroup.GET("", WithApp(a, GetShareByID))
				uuidGroup.PATCH("", WithApp(a, AcceptShare))
				uuidGroup.DELETE("", WithApp(a, DeleteShareByID))
			}
		}
	}
}

var (
	NO_ACCESS      = 0
	READ_BY_INVITE = 1
	READ_ONLY      = 2
	WRITE          = 3
)

func setupTest(t *testing.T) (*gin.Engine, *app.App) {
	app := helpers.MakeApp()
	gin.SetMode(gin.TestMode)
	r := helpers.MakeGin()

	SetupRoutes(r, app)

	database.MigrateDB(t, app.DB)
	database.ReseedDB(t, app.DB)

	return r, app
}
