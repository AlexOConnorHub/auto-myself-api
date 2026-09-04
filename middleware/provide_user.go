package middleware

import (
	"auto-myself-api/app"
	"auto-myself-api/models"
	"errors"
	"log"
	"net/http"
	"strings"

	_ "github.com/joho/godotenv/autoload"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

var secret []byte

func getBearerFromHeader(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}

	return strings.TrimSpace(header[len(prefix):])
}

func AuthMiddleware(a *app.App) gin.HandlerFunc {
	value, err := a.Secrets.Get("JWT_SIGNING_SECRET")
	if err != nil {
		log.Fatal("JWT_SIGNING_SECRET not found in secrets: ", err)
	}
	secret = []byte(value.Value)
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
			jwt.WithAudience("auto-myself-auth"),
			jwt.WithIssuer("auto-myself-auth"),
			jwt.WithIssuedAt(),
			jwt.WithExpirationRequired(),
		)
		if err != nil {
			c.AbortWithError(http.StatusUnauthorized, err)
			return
		}

		if !token.Valid {
			c.AbortWithError(http.StatusUnauthorized, errors.New("Bearer token: Not valid"))
			return
		}

		parsedUUID, err := uuid.FromString(claims.Subject)
		if err != nil {
			c.AbortWithError(http.StatusUnauthorized, errors.New("Bearer token: bad UUID"))
			return
		}

		user, err := gorm.G[models.User](a.Gorm).Where("id = ?", parsedUUID).First(c)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.AbortWithError(http.StatusUnauthorized, errors.New("Bearer token: user not found"))
			return
		} else if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		c.Set("user", &user)
		c.Next()
	}
}
