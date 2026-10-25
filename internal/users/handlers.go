package users

import (
	"auto-myself-api/internal/helpers"
	"context"
	"net/http"

	"github.com/gin-contrib/slog"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
)

type UserHandler interface {
	GetCurrentUser(*gin.Context)
	UpdateUser(*gin.Context)
	GetUserByID(*gin.Context)
}

type userHandler struct {
	userService *userService
}

func NewUserHandler(userService *userService) *userHandler {
	return &userHandler{
		userService: userService,
	}
}

func (h *userHandler) GetCurrentUser(c *gin.Context) {
	var user = c.MustGet(helpers.UserCtxKey).(*User)

	c.JSON(http.StatusOK, user.UserWeb)
}

func (h *userHandler) UpdateUser(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), helpers.UserCtxKey, c.MustGet(helpers.UserCtxKey).(*User))

	var input UserPost
	if err := c.ShouldBindJSON(&input); err != nil {
		slog.Get(c).WarnContext(ctx, "users.UpdateCurrentUser: Failed to bind JSON", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	if err := h.userService.UpdateUser(ctx, input); err != nil {
		slog.Get(c).ErrorContext(ctx, "Failed to update user", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, ctx.Value(helpers.UserCtxKey).(*User).UserWeb)
}

func (h *userHandler) GetUserByID(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), helpers.UserCtxKey, c.MustGet(helpers.UserCtxKey).(*User))
	userUUID := c.MustGet("uuid_param").(uuid.UUID)

	requestedUser, err := h.userService.GetUserByID(ctx, userUUID)

	if err != nil {
		slog.Get(c).ErrorContext(ctx, "userHandler.GetUserByID: Failed to get user by ID", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if requestedUser == nil {
		slog.Get(c).InfoContext(ctx, "userHandler.GetUserByID: User not found for ID", "user_id", userUUID)
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, requestedUser.UserWeb)
}
