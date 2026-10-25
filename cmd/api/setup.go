package api

import (
	"auto-myself-api/internal/auth"
	"auto-myself-api/internal/database"
	"auto-myself-api/internal/maintenance"
	"auto-myself-api/internal/middleware"
	"auto-myself-api/internal/reports"
	"auto-myself-api/internal/secrets"
	"auto-myself-api/internal/users"
	"auto-myself-api/internal/vehicles"
	"context"
	"fmt"

	"github.com/gin-contrib/timeout"
	"github.com/gin-gonic/gin"
)

func Setup(ctx context.Context, r *gin.Engine) {
	r.Use(gin.Recovery())
	r.Use(middleware.Logger())
	r.TrustedPlatform = gin.PlatformGoogleAppEngine

	if gin.Mode() == gin.ReleaseMode {
		r.Use(middleware.RateLimitMiddleware())
		r.Use(timeout.New())
	}

	secretStore, err := secrets.SetupSecrets(ctx, []string{
		"DOMAIN",
		"KEYRING",
		"POSTGRES_DSN",
		"GOOGLE_OAUTH2_CLIENT_ID",
		"GOOGLE_OAUTH2_CLIENT_SECRET",
	})
	if err != nil {
		panic("failed to setup secrets: " + err.Error())
	}

	sql, err := CreateSelfUpdatingDatabase(ctx, secretStore)
	if err != nil {
		panic("failed to create self-updating database: " + err.Error())
	}

	tokenRepository := auth.NewTokenRepository(secretStore)
	tokenService := auth.NewTokenService(tokenRepository)

	authRepository := auth.NewAuthRepository(sql, secretStore)
	authService := auth.NewAuthService(authRepository, tokenService)
	authHandler := auth.NewAuthHandler(authService)

	userRepository := users.NewUserRepository(sql)
	userService := users.NewUserService(userRepository)
	userHandler := users.NewUserHandler(userService)

	vehicleRepository := vehicles.NewVehicleRepository(sql)
	vehicleService := vehicles.NewVehicleService(vehicleRepository)
	vehicleHandler := vehicles.NewVehicleHandler(vehicleService)

	maintenanceRepository := maintenance.NewMaintenanceRepository(sql)
	maintenanceService := maintenance.NewMaintenanceService(maintenanceRepository, vehicleRepository)
	maintenanceHandler := maintenance.NewMaintenanceHandler(maintenanceService)

	fileRepository := file.NewFileRepository(sql)
	fileService := file.NewFileService(fileRepository)
	fileHandler := file.NewFileHandler(fileService)

	shareRepository := share.NewShareRepository(sql)
	shareService := share.NewShareService(shareRepository)
	shareHandler := share.NewShareHandler(shareService)

	reportRepository := reports.NewRepository(sql)
	reportService := reports.NewService(reportRepository)
	reportHandler := reports.NewHandler(reportService)

	authGroup := r.Group("/auth")
	{
		authGroup.GET("/web/:provider", authHandler.LoginWebInitiate)
		authGroup.GET("/web/callback/:provider", authHandler.LoginWebCallback)
		// auth.POST("/logout", )
		// authGroup.POST("/exchange/:provider", WithApp(a, authGroup.LoginExchangeProvider))
		authGroup.POST("/development", authHandler.LoginDevelopment)
		authGroup.POST("/refresh", authHandler.RefreshToken)
	}

	v1 := r.Group("/v1")
	v1.Use(middleware.AuthMiddleware(secretStore, sql))
	v1.Use(middleware.ParseUUIDMiddleware())
	{
		uuidPath := "/:uuid"
		userGroup := v1.Group("/user")
		{
			userGroup.GET("", userHandler.GetCurrentUser)
			userGroup.PATCH("", userHandler.UpdateUser)
			// user.DELETE("", WithApp(a, DeleteCurrentUser))
			uuidGroup := userGroup.Group(uuidPath)
			{

				uuidGroup.GET("", userHandler.GetUserByID)
			}
		}

		vehicleGroup := v1.Group("/vehicle")
		{
			vehicleGroup.GET("", vehicleHandler.FindIDs)
			vehicleGroup.POST("", vehicleHandler.Create)
			uuidGroup := vehicleGroup.Group(uuidPath)
			{
				uuidGroup.GET("", vehicleHandler.FindByID)
				uuidGroup.PATCH("", vehicleHandler.UpdateByID)
				uuidGroup.DELETE("", vehicleHandler.DeleteByID)
				uuidGroup.GET("/maintenance", maintenanceHandler.FindIDs)
			}
		}

		maintenanceGroup := v1.Group("/maintenance")
		{
			maintenanceGroup.POST("", maintenanceHandler.Create)
			uuidGroup := maintenanceGroup.Group(uuidPath)
			{
				uuidGroup.GET("", maintenanceHandler.FindByID)
				uuidGroup.PATCH("", maintenanceHandler.UpdateByID)
				uuidGroup.DELETE("", maintenanceHandler.DeleteByID)
			}
		}

		fileGroup := v1.Group("/file")
		{
			recordInfo := fileGroup.Group("/:record_type/:uuid")
			{
				recordInfo.GET("", fileHandler.GetFilesForTypeByID)
				recordInfo.POST("", fileHandler.CreateFileForTypeByID)
			}
			uuidGroup := fileGroup.Group(uuidPath)
			{
				uuidGroup.DELETE("", fileHandler.DeleteFileByID)
			}
		}

		shareGroup := v1.Group("/share")
		{
			shareGroup.GET("", shareHandler.GetAllShares)
			shareGroup.POST("", shareHandler.CreateShare)
			uuidGroup := shareGroup.Group(uuidPath)
			{
				uuidGroup.GET("", shareHandler.GetShareByID)
				uuidGroup.PATCH("", shareHandler.AcceptShare)
				uuidGroup.DELETE("", shareHandler.DeleteShareByID)
			}
		}

		reportGroup := v1.Group("/report")
		{
			reportGroup.POST("", reportHandler.Create)
			reportGroup.GET("", reportHandler.List)
			reportGroup.POST("/by-date", reportHandler.ListByDate)
			uuidGroup := reportGroup.Group(uuidPath)
			{
				uuidGroup.GET("", reportHandler.FindByID)
				uuidGroup.PATCH("", reportHandler.UpdateByID)
			}
		}
	}
}

func CreateSelfUpdatingDatabase(ctx context.Context, s *secrets.SecretStore) (*database.Database, error) {
	DSN_SECRET := "POSTGRES_DSN"

	db, err := database.ConnectPostgres(ctx, s.Get(DSN_SECRET).Value)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	sql, err := database.NewDatabase(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create database wrapper: %w", err)
	}

	s.Watch(DSN_SECRET, func(value secrets.SecretValue) {
		newDB, err := database.ConnectPostgres(ctx, value.Value)
		if err != nil {
			panic(err)
		}

		if err := newDB.PingContext(ctx); err != nil {
			panic(err)
		}

		_, err = sql.Replace(newDB)
		if err != nil {
			panic(err)
		}
	})

	return sql, nil
}
