package middleware

import (
	"auto-myself-api/database"
	"auto-myself-api/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func UserIDHeaderMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		uuidFromHeader := c.Request.Header.Get("auth_uuid")
		if uuidFromHeader == "" {
			c.Next()
			return
		}
		parsedUUID, err := models.ParseUUID(uuidFromHeader)
		if err != nil {
			c.AbortWithError(http.StatusUnprocessableEntity, err)
			return
		}

		var user = models.User{}

		err = database.DB.Where("id = ?", parsedUUID).First(&user).Error

		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}

			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		c.Set("user", user)
		c.Next()
	}
}
