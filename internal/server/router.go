package server

import (
	"time"

	"auto-myself-api/internal/auth"
	"auto-myself-api/internal/database"
	"auto-myself-api/internal/export"
	"auto-myself-api/internal/files"
	"auto-myself-api/internal/health"
	"auto-myself-api/internal/httperr"
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
	Database          *database.Database
	Secrets           *secrets.SecretStore
	Storage           files.StorageRepository
	IdentityVerifiers auth.IdentityVerifiers
}

type Handlers struct {
	Auth        *auth.Handler
	User        *users.Handler
	Vehicle     *vehicles.Handler
	Maintenance *maintenance.Handler
	File        *files.Handler
	Share       *shares.Handler
	Report      *reports.Handler
	Export      *export.Handler
	Health      *health.Handler
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
	tokenService := auth.NewTokenService(authRepository, d.Secrets)
	authService := auth.NewAuthService(authRepository, tokenService, d.IdentityVerifiers)
	authHandler := auth.NewAuthHandler(authService)

	exportService := export.NewService(authRepository, vehicleRepository, maintenanceRepository, fileRepository, reportRepository, d.Storage)
	exportHandler := export.NewExportHandler(exportService)

	healthHandler := health.NewHandler(d.Database)

	return &Handlers{
		Auth:        authHandler,
		User:        userHandler,
		Vehicle:     vehicleHandler,
		Maintenance: maintenanceHandler,
		File:        fileHandler,
		Share:       shareHandler,
		Report:      reportHandler,
		Export:      exportHandler,
		Health:      healthHandler,
	}
}

func NewEngine() *gin.Engine {
	r := gin.New()
	r.Use(middleware.Logger())
	r.Use(middleware.Recovery())
	r.TrustedPlatform = gin.PlatformGoogleAppEngine

	return r
}

const (
	DefaultRequestTimeout = 5 * time.Second
	UploadRequestTimeout  = 2 * time.Minute
)

type timeoutDurations struct {
	defaultTimeout time.Duration
	uploadTimeout  time.Duration
}

type routeConfig struct {
	timeouts *timeoutDurations
}

type RouteOption func(*routeConfig)

func WithTimeouts(defaultTimeout, uploadTimeout time.Duration) RouteOption {
	if defaultTimeout <= 0 || uploadTimeout <= 0 {
		panic("server: WithTimeouts requires positive durations")
	}
	return func(c *routeConfig) {
		c.timeouts = &timeoutDurations{
			defaultTimeout: defaultTimeout,
			uploadTimeout:  uploadTimeout,
		}
	}
}

func newTimeoutMiddleware(d time.Duration) gin.HandlerFunc {
	return timeout.New(
		timeout.WithTimeout(d),
		timeout.WithResponse(httperr.WriteRequestTimeout),
	)
}

func resolveTimeouts(mode string, override *timeoutDurations) *timeoutDurations {
	if override != nil {
		return override
	}
	if mode == gin.ReleaseMode {
		return &timeoutDurations{
			defaultTimeout: DefaultRequestTimeout,
			uploadTimeout:  UploadRequestTimeout,
		}
	}
	return nil
}

func RegisterRoutes(r *gin.Engine, h *Handlers, d Dependencies, opts ...RouteOption) {
	r.GET("/healthz", h.Health.Liveness)
	r.GET("/readyz", h.Health.Readiness)

	cfg := routeConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	timeouts := resolveTimeouts(gin.Mode(), cfg.timeouts)
	isRelease := gin.Mode() == gin.ReleaseMode

	var defaultTimeoutMiddleware, uploadTimeoutMiddleware gin.HandlerFunc
	if timeouts != nil {
		defaultTimeoutMiddleware = newTimeoutMiddleware(timeouts.defaultTimeout)
		uploadTimeoutMiddleware = newTimeoutMiddleware(timeouts.uploadTimeout)
	}

	authGroup := r.Group("/auth")
	authGroup.Use(func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		c.Next()
	})
	if defaultTimeoutMiddleware != nil {
		authGroup.Use(defaultTimeoutMiddleware)
	}
	if isRelease {
		authGroup.Use(middleware.RateLimitMiddleware())
	}
	{
		authGroup.POST("/mobile/:provider", h.Auth.LoginMobile)
		authGroup.POST("/development", h.Auth.LoginDevelopment)
		authGroup.POST("/refresh", h.Auth.RefreshToken)
	}

	v1Middleware := []gin.HandlerFunc{
		middleware.AuthMiddleware(d.Secrets, users.NewUserRepository(d.Database)),
		middleware.ParseUUIDMiddleware(),
	}
	if isRelease {
		v1Middleware = append(v1Middleware, middleware.UserRateLimitMiddleware())
	}

	v1 := r.Group("/v1")
	if defaultTimeoutMiddleware != nil {
		v1.Use(defaultTimeoutMiddleware)
	}
	v1.Use(v1Middleware...)
	{
		userGroup := v1.Group("/user")
		{
			userGroup.GET("", h.User.GetCurrentUser)
			userGroup.PATCH("", h.User.UpdateUser)
			userGroup.DELETE("", h.User.DeleteAccount)
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
			// POST is registered on the upload group below
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
			accessGroup.PATCH("/:uuid", h.Share.UpdateAccessByID)
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

	uploadGroup := r.Group("/v1")
	if uploadTimeoutMiddleware != nil {
		uploadGroup.Use(uploadTimeoutMiddleware)
	}
	uploadGroup.Use(v1Middleware...)
	uploadGroup.POST("/file/:record_type/:uuid", h.File.CreateForRecord)

	exportGroup := r.Group("/v1")
	exportGroup.Use(v1Middleware...) // Does not include same timeout
	exportGroup.GET("/user/export", h.Export.Download)
}
