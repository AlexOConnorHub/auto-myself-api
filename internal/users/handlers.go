package users

import (
	"auto-myself-api/internal/helpers"
	"context"
	"errors"
	"net/http"

	"github.com/gin-contrib/slog"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

type UserHandler interface {
	GetCurrentUser(*gin.Context)
	UpdateUser(*gin.Context)
	GetUserByID(*gin.Context)
}

type Handler struct {
	userService *Service
}

func NewUserHandler(userService *Service) *Handler {
	return &Handler{
		userService: userService,
	}
}

func (h *Handler) GetCurrentUser(c *gin.Context) {
	var user = c.MustGet(helpers.UserCtxKey).(*User)

	c.JSON(http.StatusOK, user.UserWeb)
}

func (h *Handler) UpdateUser(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), helpers.UserCtxKey, c.MustGet(helpers.UserCtxKey).(*User))

	var input UserUpdatePost
	if err := c.ShouldBindJSON(&input); err != nil {
		slog.Get(c).WarnContext(ctx, "invalid user update body", "error", err)
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Invalid request"})
		return
	}

	user := User{UserWeb: UserWeb{UserPost: UserPost{Username: input.Username}, Version: input.Version}}

	if err := h.userService.UpdateUser(ctx, &user); err != nil {
		switch {
		case errors.Is(err, helpers.ErrVersionConflict):
			body := gin.H{"error": "Version conflict"}
			var conflict *helpers.VersionConflictError
			if errors.As(err, &conflict) {
				body["version"] = conflict.CurrentVersion
			}
			slog.Get(c).WarnContext(ctx, "version conflict updating user")
			c.JSON(http.StatusConflict, body)
			return
		default:
			slog.Get(c).ErrorContext(ctx, "failed to update user", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}
	}

	c.JSON(http.StatusOK, user.UserWeb)
}

func (h *Handler) GetUserByID(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), helpers.UserCtxKey, c.MustGet(helpers.UserCtxKey).(*User))
	userUUID := c.MustGet(helpers.UUIDParamKey).(uuid.UUID)

	requestedUser, err := h.userService.GetUserByID(ctx, userUUID)
	switch {
	case err == nil:
	case errors.Is(err, gorm.ErrRecordNotFound):
		slog.Get(c).InfoContext(ctx, "user not found", "user_id", userUUID)
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	default:
		slog.Get(c).ErrorContext(ctx, "failed to find user", "user_id", userUUID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, requestedUser.UserWeb)
}
