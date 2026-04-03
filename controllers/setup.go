package controllers

import (
	"auto-myself-api/app"
	"auto-myself-api/database"
	"auto-myself-api/middleware"
	"testing"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine, a *app.App) {
	if gin.Mode() == gin.ReleaseMode {
		r.Use(middleware.RateLimitMiddleware())
	}

	auth := r.Group("/auth")
	{
		auth.GET("/:provider", func(c *gin.Context) { LoginWebProvider(c, a) })
		auth.GET("/callback/:provider", func(c *gin.Context) { LoginWebCallback(c, a) })
		// auth.POST("/logout", )
		auth.POST("/exchange/:provider", func(c *gin.Context) { LoginExchangeProvider(c, a) })
		auth.POST("/development", func(c *gin.Context) { LoginDevelopment(c, a) })
		auth.POST("/refresh", func(c *gin.Context) { Refresh(c, a) })
	}

	v1 := r.Group("/v1")
	v1.Use(middleware.AuthMiddleware(a))
	v1.Use(middleware.ParseUUIDMiddleware())
	{
		uuidPath := "/:uuid"
		user := v1.Group("/user")
		{
			user.GET("", func(c *gin.Context) { GetCurrentUser(c, a) })
			user.PATCH("", func(c *gin.Context) { UpdateCurrentUser(c, a) })
			// user.DELETE("", func(c *gin.Context) { DeleteCurrentUser(c, a) })
			uuidGroup := user.Group(uuidPath)
			{

				uuidGroup.GET("", func(c *gin.Context) { GetUserByID(c, a) })
			}
		}

		vehicle := v1.Group("/vehicle")
		{
			vehicle.GET("", func(c *gin.Context) { GetAllVehicles(c, a) })
			vehicle.POST("", func(c *gin.Context) { CreateVehicle(c, a) })
			uuidGroup := vehicle.Group(uuidPath)
			{
				uuidGroup.GET("", func(c *gin.Context) { GetVehicleByID(c, a) })
				uuidGroup.POST("", func(c *gin.Context) { CreateVehicle(c, a) })
				uuidGroup.PATCH("", func(c *gin.Context) { UpdateVehicleByID(c, a) })
				uuidGroup.DELETE("", func(c *gin.Context) { DeleteVehicleByID(c, a) })
				uuidGroup.GET("/maintenance", func(c *gin.Context) { GetAllMaintenance(c, a) })
			}
		}

		maintenance := v1.Group("/maintenance")
		{
			maintenance.POST("", func(c *gin.Context) { CreateMaintenance(c, a) })
			uuidGroup := maintenance.Group(uuidPath)
			{
				uuidGroup.GET("", func(c *gin.Context) { GetMaintenanceByID(c, a) })
				uuidGroup.POST("", func(c *gin.Context) { CreateMaintenance(c, a) })
				uuidGroup.PATCH("", func(c *gin.Context) { UpdateMaintenanceByID(c, a) })
				uuidGroup.DELETE("", func(c *gin.Context) { DeleteMaintenanceByID(c, a) })
			}
		}

		share := v1.Group("/share")
		{
			share.GET("", func(c *gin.Context) { GetAllShares(c, a) })
			share.POST("", func(c *gin.Context) { CreateShare(c, a) })
			uuidGroup := share.Group(uuidPath)
			{
				uuidGroup.GET("", func(c *gin.Context) { GetShareByID(c, a) })
				uuidGroup.PATCH("", func(c *gin.Context) { AcceptShare(c, a) })
				uuidGroup.DELETE("", func(c *gin.Context) { DeleteShareByID(c, a) })
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
	db := database.TestConnectDB(t)
	gorm := database.ConnectGorm(db)
	gclient := database.ConnectGoogleClient()

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	a := &app.App{
		DB:      db,
		Gorm:    gorm,
		Gclient: gclient,
	}
	SetupRoutes(r, a)

	database.MigrateDB(t, db)
	database.ReseedDB(t, db)

	return r, a
}
