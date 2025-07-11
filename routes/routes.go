package routes

import (
	"auto-myself-api/controllers"
	"auto-myself-api/middleware"

	"github.com/fufuok/favicon"
	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.UserIDHeaderMiddleware())

	r.Use(favicon.New(favicon.Config{
		File: "favicon.ico",
	}))

	public := r.Group("/public")
	{
		public.GET("/ping", controllers.PublicPing)
	}

	private := r.Group("/")
	// private.Use(middleware.CheckJWT())
	{
		user := private.Group("/user")
		{
			user.GET("", controllers.GetCurrentUser)
			user.PATCH("", controllers.UpdateCurrentUser)
			// user.DELETE("", controllers.DeleteCurrentUser)
			user.GET("/:uuid", controllers.GetUserByID)
		}

		vehicle := private.Group("/vehicle")
		{
			vehicle.POST("", controllers.CreateVehicle)
			vehicle.GET("", controllers.GetAllVehicles)
			vehicle.GET("/:uuid", controllers.GetVehicleByID)
			vehicle.GET("/:uuid/maintenance", controllers.GetAllMaintenance)
			vehicle.PATCH("/:uuid", controllers.UpdateVehicleByID)
			vehicle.DELETE("/:uuid", controllers.DeleteVehicleByID)
		}

		maintenance := private.Group("/maintenance")
		{
			maintenance.POST("", controllers.CreateMaintenance)
			maintenance.GET("/:uuid", controllers.GetMaintenanceByID)
			maintenance.PATCH("/:uuid", controllers.UpdateMaintenanceByID)
			maintenance.DELETE("/:uuid", controllers.DeleteMaintenanceByID)
		}
	}
}
