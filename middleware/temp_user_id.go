package middleware

import (
	"auto-myself-api/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

func UserIDHeaderMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		uuidFromHeader := c.Request.Header.Get("uuid")
		if uuidFromHeader == "" {
			c.Next()
			return
		}
		parsedUUID, err := models.ParseUUID(uuidFromHeader)
		if err != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Invalid UUID"})
			return
		}

		c.Set("USER_ID", parsedUUID)
		c.Next()
	}
}
