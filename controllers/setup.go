package controllers

import (
	"auto-myself-api/database"
	"auto-myself-api/helpers"
	"auto-myself-api/middleware"
	"testing"

	"github.com/fufuok/favicon"
	"github.com/gin-gonic/gin"

	_ "auto-myself-api/docs"

	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func SetupRoutes(r *gin.Engine) {
	switch gin.Mode() {
	case gin.DebugMode:
		r.Use(middleware.CORSAllowAllMiddleware())
		r.Use(middleware.ContextGetUserHeaderMiddleware())
	case gin.TestMode:
		r.Use(middleware.ContextGetUserHeaderMiddleware())
	case gin.ReleaseMode:
		r.Use(middleware.RateLimitMiddleware())
		r.Use(middleware.ContextGetUserJWTMiddleware())
	}

	cwd := helpers.GetRelativeRootPath(nil)

	r.Use(favicon.New(favicon.Config{
		File: cwd + "/favicon.ico",
	}))

	r.GET("/docs/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	user := r.Group("/user")
	{
		user.GET("", GetCurrentUser)
		user.PATCH("", UpdateCurrentUser)
		// user.DELETE("", DeleteCurrentUser)
		user.GET("/:uuid", GetUserByID)
	}

	vehicle := r.Group("/vehicle")
	{
		vehicle.POST("", CreateVehicle)
		vehicle.GET("", GetAllVehicles)
		vehicle.GET("/:uuid", GetVehicleByID)
		vehicle.GET("/:uuid/maintenance", GetAllMaintenance)
		vehicle.PATCH("/:uuid", UpdateVehicleByID)
		vehicle.DELETE("/:uuid", DeleteVehicleByID)
	}

	maintenance := r.Group("/maintenance")
	{
		maintenance.POST("", CreateMaintenance)
		maintenance.GET("/:uuid", GetMaintenanceByID)
		maintenance.PATCH("/:uuid", UpdateMaintenanceByID)
		maintenance.DELETE("/:uuid", DeleteMaintenanceByID)
	}
}

func setupTest(t *testing.T) *gin.Engine {
	database.InitTest(t)
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	SetupRoutes(r)

	return r
}
