package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	ginSlog "github.com/gin-contrib/slog"
	"github.com/gin-gonic/gin"
)

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}
			requestLogger(c).ErrorContext(c.Request.Context(), "panic recovered",
				"error", recovered, "stack", string(debug.Stack()))
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		}()

		c.Next()
	}
}

func requestLogger(c *gin.Context) (logger *slog.Logger) {
	defer func() {
		if recover() != nil {
			logger = slog.Default()
		}
	}()
	return ginSlog.Get(c)
}
