package auth

import (
	"auto-myself-api/internal/helpers"
	"auto-myself-api/internal/users"
	"context"
	"errors"
	"net/http"

	"github.com/gin-contrib/slog"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AuthHandler interface {
	LoginMobile(c *gin.Context)
	RefreshToken(c *gin.Context)
	LoginDevelopment(c *gin.Context)
	SignOutEverywhere(c *gin.Context)
}

type Handler struct {
	authService *Service
}

func NewAuthHandler(authService *Service) *Handler {
	return &Handler{
		authService: authService,
	}
}

func (h *Handler) LoginMobile(c *gin.Context) {
	ctx := c.Request.Context()
	provider := c.Param("provider")

	if !IsSupportedMobileProvider(provider) {
		slog.Get(c).WarnContext(ctx, "unsupported mobile login provider", "provider", provider)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported provider"})
		return
	}

	var request MobileLoginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		slog.Get(c).WarnContext(ctx, "invalid mobile login body", "provider", provider, "error", err)
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Invalid request"})
		return
	}

	userAuth, err := h.authService.LoginMobile(ctx, provider, request)
	h.respondWithLogin(c, provider, userAuth, err)
}

func (h *Handler) respondWithLogin(c *gin.Context, provider string, userAuth *UserAuthResponse, err error) {
	ctx := c.Request.Context()

	switch {
	case err == nil:
		c.JSON(http.StatusOK, gin.H{"jwt": userAuth.JWT, "refresh": userAuth.Refresh})
	case errors.Is(err, ErrUnsupportedProvider):
		slog.Get(c).WarnContext(ctx, "unsupported mobile login provider", "provider", provider)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported provider"})
	case errors.Is(err, ErrInvalidIdentityToken):
		slog.Get(c).WarnContext(ctx, "invalid identity token", "provider", provider)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid identity token"})
	default:
		slog.Get(c).ErrorContext(ctx, "mobile login failed", "provider", provider, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
	}
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

func (h *Handler) SignOutEverywhere(c *gin.Context) {
	user := c.MustGet(helpers.UserCtxKey).(*users.User)
	ctx := context.WithValue(c.Request.Context(), helpers.UserCtxKey, user)

	if err := h.authService.SignOutEverywhere(ctx); err != nil {
		slog.Get(c).ErrorContext(ctx, "failed to sign out everywhere", "user_id", user.ID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	slog.Get(c).InfoContext(ctx, "signed out everywhere", "user_id", user.ID)
	c.Status(http.StatusNoContent)
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
