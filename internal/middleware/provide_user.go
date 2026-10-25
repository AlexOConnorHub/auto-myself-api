package middleware

import (
	"auto-myself-api/internal/database"
	"auto-myself-api/internal/secrets"
	"auto-myself-api/internal/users"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	_ "github.com/joho/godotenv/autoload"
	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
	"github.com/golang-jwt/jwt/v5"
)

var secret []byte

func getBearerFromHeader(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}

	return strings.TrimSpace(header[len(prefix):])
}

func AuthMiddleware(secrets *secrets.SecretStore, database *database.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		value := secrets.Get("JWT_SIGNING_SECRET")
		secret = []byte(value.Value)
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
			slog.Error("Failed to parse JWT token: ", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid request"})
			return
		}

		if !token.Valid {
			slog.Error("JWT token is not valid")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid request"})
			return
		}

		parsedUUID, err := uuid.FromString(claims.Subject)
		if err != nil {
			slog.Error("Failed to parse UUID from JWT subject: ", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid request"})
			return
		}

		user, err := gorm.G[users.User](database.GetGorm()).Where("id = ?", parsedUUID).First(ctx)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			slog.Error("User not found for UUID: ", parsedUUID)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid request"})
			return
		} else if err != nil {
			slog.Error("Failed to retrieve user from database: ", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		c.Set("user", &user)
		c.Next()
	}
}
