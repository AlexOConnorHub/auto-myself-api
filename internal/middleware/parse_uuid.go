package middleware

import (
	"auto-myself-api/internal/helpers"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
)

func ParseUUIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		param := c.Param("uuid")
		if param != "" {
			parsedUUID, err := uuid.FromString(param)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Malformed UUID"})
				return
			}

			if _, err := uuid.TimestampFromV7(parsedUUID); err != nil {
				slog.WarnContext(c.Request.Context(), "UUID parameter is not v7", "uuid", parsedUUID, "error", err)
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "UUID must be v7 (with valid timestamp)"})
				return
			}

			c.Set(helpers.UUIDParamKey, parsedUUID)
		}
		c.Next()
	}
}
