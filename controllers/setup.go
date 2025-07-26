package controllers

import (
	"auto-myself-api/middleware"

	"github.com/fufuok/favicon"
	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	switch gin.Mode() {
	case gin.DebugMode:
		r.Use(middleware.CORSAllowAllMiddleware())
		r.Use(middleware.ContextGetUserHeaderMiddleware())
	case gin.TestMode:
		r.Use(middleware.RateLimitMiddleware())
		r.Use(middleware.ContextGetUserHeaderMiddleware())
	case gin.ReleaseMode:
		r.Use(middleware.RateLimitMiddleware())
		r.Use(middleware.ContextGetUserJWTMiddleware())
	}

	r.Use(favicon.New(favicon.Config{
		File: "favicon.ico",
	}))

	private := r.Group("/")
	{
		user := private.Group("/user")
		{
			user.GET("", GetCurrentUser)
			user.PATCH("", UpdateCurrentUser)
			// user.DELETE("", DeleteCurrentUser)
			user.GET("/:uuid", GetUserByID)
		}

		vehicle := private.Group("/vehicle")
		{
			vehicle.POST("", CreateVehicle)
			vehicle.GET("", GetAllVehicles)
			vehicle.GET("/:uuid", GetVehicleByID)
			vehicle.GET("/:uuid/maintenance", GetAllMaintenance)
			vehicle.PATCH("/:uuid", UpdateVehicleByID)
			vehicle.DELETE("/:uuid", DeleteVehicleByID)
		}

		maintenance := private.Group("/maintenance")
		{
			maintenance.POST("", CreateMaintenance)
			maintenance.GET("/:uuid", GetMaintenanceByID)
			maintenance.PATCH("/:uuid", UpdateMaintenanceByID)
			maintenance.DELETE("/:uuid", DeleteMaintenanceByID)
		}
	}
}
