package middleware

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
)

func ParseUUIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userUUID := c.Param("uuid")
		if userUUID != "" {
			parsedUUID, err := uuid.FromString(userUUID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Malformed UUID"})
				c.Abort()
				return
			}

			_, err = uuid.TimestampFromV7(parsedUUID)
			if err != nil {
				fmt.Printf("Error parsing timestamp from UUID %s: %v\n", parsedUUID, err)
				c.JSON(http.StatusBadRequest, gin.H{"error": "UUID must be v7 (with valid timestamp)"})
				c.Abort()
				return
			}

			c.Set("uuid_param", parsedUUID)
		}
		c.Next()
	}
}
