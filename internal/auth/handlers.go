package auth

import (
	"errors"
	"net/http"

	"github.com/gin-contrib/slog"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AuthHandler interface {
	LoginWebInitiate(c *gin.Context)
	LoginWebCallback(c *gin.Context)
	RefreshToken(c *gin.Context)
	LoginDevelopment(c *gin.Context)
}

type Handler struct {
	authService *Service
}

func NewAuthHandler(authService *Service) *Handler {
	return &Handler{
		authService: authService,
	}
}

func (h *Handler) LoginWebInitiate(c *gin.Context) {
	ctx := c.Request.Context()
	provider := c.Param("provider")

	url, err := h.authService.LoginWebInitiate(ctx, provider)
	if err != nil {
		if errors.Is(err, ErrUnsupportedProvider) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported provider"})
			return
		}
		slog.Get(c).ErrorContext(ctx, "failed to start oauth login", "provider", provider, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, url)
}

func (h *Handler) LoginWebCallback(c *gin.Context) {
	ctx := c.Request.Context()
	provider := c.Param("provider")

	code := c.Query("code")
	if code == "" {
		slog.Get(c).WarnContext(ctx, "oauth callback missing code", "provider", provider)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	userAuth, err := h.authService.LoginWebCallback(ctx, provider, code)
	if err != nil {
		if errors.Is(err, ErrUnsupportedProvider) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported provider"})
			return
		}
		slog.Get(c).ErrorContext(ctx, "oauth callback failed", "provider", provider, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"jwt":     userAuth.JWT,
		"refresh": userAuth.Refresh,
	})
}

func (h *Handler) RefreshToken(c *gin.Context) {
	ctx := c.Request.Context()

	var request RefreshTokenRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		slog.Get(c).WarnContext(ctx, "invalid refresh token body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	userAuth, err := h.authService.RefreshToken(ctx, request)
	switch {
	case err == nil:
	case errors.Is(err, gorm.ErrRecordNotFound), errors.Is(err, ErrInvalidRefreshToken):
		slog.Get(c).WarnContext(ctx, "invalid or expired refresh token")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	default:
		slog.Get(c).ErrorContext(ctx, "failed to refresh token", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"jwt": userAuth.JWT, "refresh": userAuth.Refresh})
}

// LoginDevelopment is a development-only helper.
func (h *Handler) LoginDevelopment(c *gin.Context) {
	ctx := c.Request.Context()

	if gin.Mode() != gin.DebugMode && gin.Mode() != gin.TestMode {
		c.Status(http.StatusTeapot)
		return
	}

	var request DevelopmentLoginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		slog.Get(c).WarnContext(ctx, "invalid development login body", "error", err)
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Invalid request"})
		return
	}

	userAuth, err := h.authService.DevelopmentLogin(ctx, request)
	switch {
	case err == nil:
	case errors.Is(err, gorm.ErrRecordNotFound):
		slog.Get(c).InfoContext(ctx, "development user not found", "user_id", request.UserID)
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	default:
		slog.Get(c).ErrorContext(ctx, "development login failed", "user_id", request.UserID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"authentication": "Bearer " + userAuth.JWT,
		"refresh":        userAuth.Refresh,
	})
}
