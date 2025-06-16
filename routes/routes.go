package routes

import (
	"auto-myself-server/controllers"

	"github.com/fufuok/favicon"
	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
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
			user.GET("/", controllers.GetCurrentUser)
			user.PATCH("/", controllers.UpdateCurrentUser)
			user.DELETE("/", controllers.DeleteCurrentUser)
			user.GET("/:uuid", controllers.GetUserById)
		}

		vehicle := private.Group("/vehicle")
		{
			vehicle.POST("/", controllers.CreateVehicle)
			vehicle.GET("/all", controllers.GetAllVehicles)
			vehicle.GET("/:uuid", controllers.GetVehicle)
			vehicle.PATCH("/:uuid", controllers.UpdateVehicle)
			vehicle.DELETE("/:uuid", controllers.DeleteVehicle)
		}

		maintenance := private.Group("/maintenance")
		{
			maintenance.POST("/", controllers.CreateMaintenance)
			maintenance.GET("/:uuid", controllers.GetMaintenance)
			maintenance.PATCH("/:uuid", controllers.UpdateMaintenance)
			maintenance.DELETE("/:uuid", controllers.DeleteMaintenance)
		}
	}
}
