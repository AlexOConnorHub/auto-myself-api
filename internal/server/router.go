package server

import (
	"auto-myself-api/internal/auth"
	"auto-myself-api/internal/database"
	"auto-myself-api/internal/files"
	"auto-myself-api/internal/maintenance"
	"auto-myself-api/internal/middleware"
	"auto-myself-api/internal/reports"
	"auto-myself-api/internal/secrets"
	"auto-myself-api/internal/shares"
	"auto-myself-api/internal/users"
	"auto-myself-api/internal/vehicles"

	"github.com/gin-contrib/timeout"
	"github.com/gin-gonic/gin"
)

type Dependencies struct {
	Database *database.Database
	Secrets  *secrets.SecretStore
	Storage  files.StorageRepository
}

type Handlers struct {
	Auth        *auth.Handler
	User        *users.Handler
	Vehicle     *vehicles.Handler
	Maintenance *maintenance.Handler
	File        *files.Handler
	Share       *shares.Handler
	Report      *reports.Handler
}

func Build(d Dependencies) *Handlers {
	userRepository := users.NewUserRepository(d.Database)
	userService := users.NewUserService(userRepository)
	userHandler := users.NewUserHandler(userService)

	vehicleRepository := vehicles.NewVehicleRepository(d.Database)
	vehicleService := vehicles.NewVehicleService(vehicleRepository)
	vehicleHandler := vehicles.NewVehicleHandler(vehicleService)

	maintenanceRepository := maintenance.NewMaintenanceRepository(d.Database)
	maintenanceService := maintenance.NewMaintenanceService(maintenanceRepository, vehicleRepository)
	maintenanceHandler := maintenance.NewMaintenanceHandler(maintenanceService)

	fileRepository := files.NewFileRepository(d.Database)
	fileService := files.NewFileService(fileRepository, d.Storage, maintenanceRepository, vehicleRepository)
	fileHandler := files.NewFileHandler(fileService)

	shareRepository := shares.NewShareRepository(d.Database)
	shareService := shares.NewShareService(shareRepository, vehicleRepository, userRepository)
	shareHandler := shares.NewShareHandler(shareService)

	reportRepository := reports.NewReportRepository(d.Database)
	reportService := reports.NewReportService(reportRepository)
	reportHandler := reports.NewReportHandler(reportService)

	authRepository := auth.NewAuthRepository(d.Database)
	oauthRepository := auth.NewGoogleOAuthRepository()
	tokenService := auth.NewTokenService(authRepository, d.Secrets)
	authService := auth.NewAuthService(authRepository, tokenService, oauthRepository, d.Secrets)
	authHandler := auth.NewAuthHandler(authService)

	return &Handlers{
		Auth:        authHandler,
		User:        userHandler,
		Vehicle:     vehicleHandler,
		Maintenance: maintenanceHandler,
		File:        fileHandler,
		Share:       shareHandler,
		Report:      reportHandler,
	}
}

func NewEngine() *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.Logger())
	r.TrustedPlatform = gin.PlatformGoogleAppEngine

	if gin.Mode() == gin.ReleaseMode {
		r.Use(timeout.New())
	}

	return r
}

func RegisterRoutes(r *gin.Engine, h *Handlers, d Dependencies) {
	authGroup := r.Group("/auth")
	if gin.Mode() == gin.ReleaseMode {
		authGroup.Use(middleware.RateLimitMiddleware())
	}
	{
		authGroup.GET("/web/:provider", h.Auth.LoginWebInitiate)
		authGroup.GET("/web/callback/:provider", h.Auth.LoginWebCallback)
		authGroup.POST("/development", h.Auth.LoginDevelopment)
		authGroup.POST("/refresh", h.Auth.RefreshToken)
	}

	v1 := r.Group("/v1")
	v1.Use(middleware.AuthMiddleware(d.Secrets, users.NewUserRepository(d.Database)))
	v1.Use(middleware.ParseUUIDMiddleware())
	if gin.Mode() == gin.ReleaseMode {
		v1.Use(middleware.UserRateLimitMiddleware())
	}
	{
		userGroup := v1.Group("/user")
		{
			userGroup.GET("", h.User.GetCurrentUser)
			userGroup.PATCH("", h.User.UpdateUser)
			userGroup.GET("/:uuid", h.User.GetUserByID)
		}

		vehicleGroup := v1.Group("/vehicle")
		{
			vehicleGroup.GET("", h.Vehicle.FindIDs)
			vehicleGroup.POST("", h.Vehicle.Create)
			vehicleGroup.POST("/:uuid", h.Vehicle.Create)
			vehicleGroup.GET("/:uuid", h.Vehicle.FindByID)
			vehicleGroup.PATCH("/:uuid", h.Vehicle.UpdateByID)
			vehicleGroup.DELETE("/:uuid", h.Vehicle.DeleteByID)
			vehicleGroup.GET("/:uuid/maintenance", h.Maintenance.FindIDs)
			vehicleGroup.GET("/:uuid/access", h.Share.GetVehicleAccess)
		}

		maintenanceGroup := v1.Group("/maintenance")
		{
			maintenanceGroup.POST("", h.Maintenance.Create)
			maintenanceGroup.POST("/:uuid", h.Maintenance.Create)
			maintenanceGroup.GET("/:uuid", h.Maintenance.FindByID)
			maintenanceGroup.PATCH("/:uuid", h.Maintenance.UpdateByID)
			maintenanceGroup.DELETE("/:uuid", h.Maintenance.DeleteByID)
		}

		fileGroup := v1.Group("/file")
		{
			fileGroup.GET("/:record_type/:uuid", h.File.GetForRecord)
			fileGroup.POST("/:record_type/:uuid", h.File.CreateForRecord)
			fileGroup.DELETE("/:uuid", h.File.DeleteByID)
		}

		shareGroup := v1.Group("/share")
		{
			shareGroup.GET("", h.Share.GetAllShares)
			shareGroup.POST("", h.Share.CreateShare)
			shareGroup.GET("/:uuid", h.Share.GetShareByID)
			shareGroup.PATCH("/:uuid", h.Share.AcceptShare)
			shareGroup.DELETE("/:uuid", h.Share.DeleteShareByID)
		}

		accessGroup := v1.Group("/access")
		{
			accessGroup.DELETE("/:uuid", h.Share.DeleteAccessByID)
		}

		reportGroup := v1.Group("/report")
		{
			reportGroup.POST("", h.Report.Create)
			reportGroup.GET("", h.Report.List)
			reportGroup.GET("/unresolved", h.Report.ListUnresolved)
			reportGroup.POST("/by-date", h.Report.ListByDate)
			reportGroup.GET("/:uuid", h.Report.FindByID)
			reportGroup.PATCH("/:uuid", h.Report.UpdateByID)
		}
	}
}
