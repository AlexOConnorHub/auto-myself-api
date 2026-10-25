package middleware

import (
	"log/slog"
	"os"

	ginSlog "github.com/gin-contrib/slog"
	"github.com/gin-gonic/gin"
)

func Logger() gin.HandlerFunc {
	loggerInstance := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelInfo,
	}))

	loggerFunc := func(_ *gin.Context, _ *slog.Logger) *slog.Logger {
		return loggerInstance
	}

	return ginSlog.SetLogger(ginSlog.WithLogger(loggerFunc))
}
