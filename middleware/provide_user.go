package middleware

import (
	"auto-myself-api/app"
	"auto-myself-api/helpers"
	"auto-myself-api/models"
	"log"
	"net/http"
	"strings"

	_ "github.com/joho/godotenv/autoload"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

func getBearerFromHeader(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}

	return strings.TrimSpace(header[len(prefix):])
}

func AuthMiddleware(a *app.App) gin.HandlerFunc {
	secret := []byte(helpers.GetSecret("JWT_SIGNING_SECRET"))
	if len(secret) == 0 {
		log.Fatal("JWT_SIGNING_SECRET environment variable is not set")
	}
	return func(c *gin.Context) {
		tokenString := getBearerFromHeader(c.GetHeader("Authorization"))

		claims := &jwt.RegisteredClaims{}

		token, err := jwt.ParseWithClaims(
			tokenString,
			claims,
			func(token *jwt.Token) (any, error) {
				return secret, nil
			},
			jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
			jwt.WithAudience("auto-myself-api"),
			jwt.WithIssuer("auto-myself-api"),
			jwt.WithIssuedAt(),
			jwt.WithExpirationRequired(),
		)
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		if !token.Valid {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		parsedUUID, err := uuid.FromString(claims.Subject)
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		var user = models.User{}
		err = a.Gorm.First(&user, "id = ?", parsedUUID).Error

		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}

			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		c.Set("user", &user)
		c.Next()
	}
}
