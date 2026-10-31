package middleware

import (
	"auto-myself-api/internal/helpers"
	"auto-myself-api/internal/secrets"
	"auto-myself-api/internal/users"
	"errors"
	"log/slog"
	"net/http"
	"strings"

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

func AuthMiddleware(secretStore *secrets.SecretStore, userRepository *users.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		secretValue, err := secretStore.Get("JWT_SIGNING_SECRET")
		if err != nil {
			slog.ErrorContext(ctx, "failed to read JWT signing secret", "error", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid request"})
			return
		}

		tokenString := getBearerFromHeader(c.GetHeader("Authorization"))
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid request"})
			return
		}

		claims := &jwt.RegisteredClaims{}
		secret := []byte(secretValue.Value)
		token, err := jwt.ParseWithClaims(
			tokenString,
			claims,
			func(token *jwt.Token) (any, error) {
				return secret, nil
			},
			jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
			jwt.WithAudience("auto-myself-api"),
			jwt.WithIssuer("auto-myself-auth"),
			jwt.WithIssuedAt(),
			jwt.WithExpirationRequired(),
		)
		if err != nil || !token.Valid {
			slog.WarnContext(ctx, "invalid JWT", "error", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid request"})
			return
		}

		subject, err := uuid.FromString(claims.Subject)
		if err != nil {
			slog.WarnContext(ctx, "JWT subject is not a UUID", "error", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid request"})
			return
		}

		user, err := userRepository.FindByID(ctx, subject)
		switch {
		case err == nil:
		case errors.Is(err, gorm.ErrRecordNotFound):
			slog.WarnContext(ctx, "JWT user not found", "user_id", subject)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid request"})
			return
		default:
			slog.ErrorContext(ctx, "failed to load user for JWT", "user_id", subject, "error", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		c.Set(helpers.UserCtxKey, user)
		c.Next()
	}
}
