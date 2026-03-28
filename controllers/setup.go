package controllers

import (
	"auto-myself-api/database"
	"auto-myself-api/middleware"
	"testing"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	if gin.Mode() == gin.ReleaseMode {
		r.Use(middleware.RateLimitMiddleware())
	}

	auth := r.Group("/auth")
	{
		auth.GET("/:provider", LoginWebProvider)
		auth.GET("/callback/:provider", LoginWebCallback)
		// auth.POST("/logout", )
		auth.POST("/exchange/:provider", LoginExchangeProvider)
		auth.POST("/development", LoginDevelopment)
		auth.POST("/refresh", Refresh)
	}

	v1 := r.Group("/v1")
	v1.Use(middleware.AuthMiddleware())
	{
		user := v1.Group("/user")
		{
			user.GET("", GetCurrentUser)
			user.PATCH("", UpdateCurrentUser)
			// user.DELETE("", DeleteCurrentUser)
			user.GET("/:uuid", GetUserByID)
		}

		vehicle := v1.Group("/vehicle")
		{
			vehicle.POST("", CreateVehicle)
			vehicle.GET("", GetAllVehicles)
			vehicle.GET("/:uuid", GetVehicleByID)
			vehicle.GET("/:uuid/maintenance", GetAllMaintenance)
			vehicle.PATCH("/:uuid", UpdateVehicleByID)
			vehicle.DELETE("/:uuid", DeleteVehicleByID)
		}

		maintenance := v1.Group("/maintenance")
		{
			maintenance.POST("", CreateMaintenance)
			maintenance.GET("/:uuid", GetMaintenanceByID)
			maintenance.PATCH("/:uuid", UpdateMaintenanceByID)
			maintenance.DELETE("/:uuid", DeleteMaintenanceByID)
		}

		share := v1.Group("/share")
		{
			share.GET("", GetAllShares)
			share.GET("/:uuid", GetShareByID)
			share.POST("", CreateShare)
			share.PATCH("/:uuid", AcceptShare)
			share.DELETE("/:uuid", DeleteShareByID)
		}
	}
}

var (
	NO_ACCESS = 0
	READ_ONLY = 1
	WRITE     = 2
)

func setupTest(t *testing.T) *gin.Engine {
	database.InitTest(t)
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	SetupRoutes(r)

	return r
}
