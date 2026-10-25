package auth

import (
	"net/http"

	"github.com/gin-contrib/slog"
	"github.com/gin-gonic/gin"
)

type AuthHandler interface {
	LoginWebProvider(c *gin.Context)
	LoginWebCallback(c *gin.Context)
	RefreshToken(c *gin.Context)
	LoginDevelopment(c *gin.Context)
}

type authHandler struct {
	authService *authService
}

func NewAuthHandler(authService *authService) *authHandler {
	return &authHandler{
		authService: authService,
	}
}

func (h *authHandler) LoginWebInitiate(c *gin.Context) {
	provider := c.Param("provider")

	url, err := h.authService.LoginWebInitiate(c, provider)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported provider"})
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, url)
}

func (h *authHandler) LoginWebCallback(c *gin.Context) {
	ctx := c.Request.Context()
	provider := c.Param("provider")

	code := c.Query("code")
	if code == "" {
		slog.Get(c).ErrorContext(ctx, "authHandler.LoginWebCallback: code not found in query parameters (Google requirements)")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	userAuth, err := h.authService.LoginWebCallback(ctx, provider, code)

	if err != nil {
		slog.Get(c).ErrorContext(ctx, "authHandler.LoginWebCallback: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if userAuth.JWT == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"jwt":     userAuth.JWT,
		"refresh": userAuth.Refresh,
	})
}

func (h *authHandler) RefreshToken(c *gin.Context) {
	ctx := c.Request.Context()

	var userProvidedToken RefreshTokenRequest
	if err := c.ShouldBindJSON(&userProvidedToken); err != nil {
		slog.Get(c).WarnContext(ctx, "authHandler.RefreshToken: Failed to bind JSON", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	userAuth, err := h.authService.RefreshToken(ctx, userProvidedToken)
	if err != nil {
		slog.Get(c).ErrorContext(ctx, "authHandler.RefreshToken: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if userAuth.JWT == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"jwt": userAuth.JWT, "refresh": userAuth.Refresh})
	return
}

// DEVELOPMENT ONLY Login
func (h *authHandler) LoginDevelopment(c *gin.Context) {
	ctx := c.Request.Context()

	if gin.Mode() != gin.DebugMode && gin.Mode() != gin.TestMode {
		c.Status(http.StatusTeapot)
		return
	}

	var developmentLoginRequest DevelopmentLoginRequest
	if err := c.ShouldBindJSON(&developmentLoginRequest); err != nil {
		slog.Get(c).WarnContext(ctx, "authHandler.LoginDevelopment: Failed to bind JSON", "error", err)
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Invalid request"})
		return
	}

	userAuth, err := h.authService.DevelopmentLogin(ctx, developmentLoginRequest)

	if err != nil {
		slog.Get(c).ErrorContext(ctx, "authHandler.LoginDevelopment: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if userAuth.JWT == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"authentication": "Bearer " + userAuth.JWT,
		"refresh":        userAuth.Refresh,
	})
}
