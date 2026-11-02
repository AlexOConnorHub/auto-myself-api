package shares

import (
	"auto-myself-api/internal/helpers"
	"auto-myself-api/internal/users"
	"context"
	"errors"
	"net/http"

	"github.com/gin-contrib/slog"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

type ShareHandler interface {
	GetAllShares(*gin.Context)
	GetShareByID(*gin.Context)
	CreateShare(*gin.Context)
	AcceptShare(*gin.Context)
	DeleteShareByID(*gin.Context)
	GetVehicleAccess(*gin.Context)
	UpdateAccessByID(*gin.Context)
	DeleteAccessByID(*gin.Context)
}

type Handler struct {
	shareService *Service
}

func NewShareHandler(shareService *Service) *Handler {
	return &Handler{
		shareService: shareService,
	}
}

func (h *Handler) GetAllShares(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), helpers.UserCtxKey, c.MustGet(helpers.UserCtxKey).(*users.User))

	shareIDs, err := h.shareService.GetAllShares(ctx)
	if err != nil {
		slog.Get(c).ErrorContext(ctx, "failed to list shares", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, shareIDs)
}

func (h *Handler) GetShareByID(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), helpers.UserCtxKey, c.MustGet(helpers.UserCtxKey).(*users.User))
	shareUUID := c.MustGet(helpers.UUIDParamKey).(uuid.UUID)

	share, err := h.shareService.GetShareByID(ctx, shareUUID)
	switch {
	case err == nil:
	case errors.Is(err, gorm.ErrRecordNotFound):
		slog.Get(c).InfoContext(ctx, "share not found", "share_id", shareUUID)
		c.JSON(http.StatusNotFound, gin.H{"error": "Share not found"})
		return
	default:
		slog.Get(c).ErrorContext(ctx, "failed to find share", "share_id", shareUUID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, share.VehicleUserAccessBase)
}

func (h *Handler) CreateShare(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), helpers.UserCtxKey, c.MustGet(helpers.UserCtxKey).(*users.User))

	var share VehicleUserAccessPending
	if err := c.ShouldBindJSON(&share.VehicleUserAccessBase); err != nil {
		slog.Get(c).WarnContext(ctx, "invalid share body", "error", err)
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Invalid request"})
		return
	}

	err := h.shareService.CreateShare(ctx, &share)
	switch {
	case err == nil:
	case errors.Is(err, helpers.ErrAccessDenied):
		slog.Get(c).WarnContext(ctx, "access denied creating share", "vehicle_id", share.VehicleID)
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	case errors.Is(err, gorm.ErrRecordNotFound), errors.Is(err, gorm.ErrForeignKeyViolated):
		slog.Get(c).InfoContext(ctx, "vehicle or user not found while creating share", "vehicle_id", share.VehicleID, "user_id", share.UserID)
		c.JSON(http.StatusNotFound, gin.H{"error": "Vehicle or user not found"})
		return
	case errors.Is(err, ErrShareAlreadyExists), errors.Is(err, gorm.ErrDuplicatedKey):
		slog.Get(c).WarnContext(ctx, "share already exists", "vehicle_id", share.VehicleID, "user_id", share.UserID)
		c.JSON(http.StatusConflict, gin.H{"error": "Share already exists"})
		return
	case errors.Is(err, ErrCannotShareWithSelf):
		slog.Get(c).WarnContext(ctx, "cannot share with self", "user_id", share.UserID)
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Cannot share a vehicle with yourself"})
		return
	default:
		slog.Get(c).ErrorContext(ctx, "failed to create share", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.Header("Location", "/v1/share/"+share.ID.String())
	c.Status(http.StatusCreated)
}

func (h *Handler) AcceptShare(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), helpers.UserCtxKey, c.MustGet(helpers.UserCtxKey).(*users.User))
	shareUUID := c.MustGet(helpers.UUIDParamKey).(uuid.UUID)

	access, err := h.shareService.AcceptShare(ctx, shareUUID)
	switch {
	case err == nil:
	case errors.Is(err, helpers.ErrAccessDenied):
		slog.Get(c).WarnContext(ctx, "creator may not accept their own share", "share_id", shareUUID)
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	case errors.Is(err, gorm.ErrRecordNotFound):
		slog.Get(c).InfoContext(ctx, "share not found", "share_id", shareUUID)
		c.JSON(http.StatusNotFound, gin.H{"error": "Share not found"})
		return
	case errors.Is(err, gorm.ErrDuplicatedKey):
		slog.Get(c).WarnContext(ctx, "access already exists", "share_id", shareUUID)
		c.JSON(http.StatusConflict, gin.H{"error": "Access already exists"})
		return
	default:
		slog.Get(c).ErrorContext(ctx, "failed to accept share", "share_id", shareUUID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.Header("Location", "/v1/vehicle/"+access.VehicleID.String())
	c.Status(http.StatusNoContent)
}

func (h *Handler) DeleteShareByID(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), helpers.UserCtxKey, c.MustGet(helpers.UserCtxKey).(*users.User))
	shareUUID := c.MustGet(helpers.UUIDParamKey).(uuid.UUID)

	err := h.shareService.DeleteShareByID(ctx, shareUUID)
	switch {
	case err == nil:
	case errors.Is(err, gorm.ErrRecordNotFound):
		slog.Get(c).InfoContext(ctx, "share not found", "share_id", shareUUID)
		c.JSON(http.StatusNotFound, gin.H{"error": "Share not found"})
		return
	default:
		slog.Get(c).ErrorContext(ctx, "failed to delete share", "share_id", shareUUID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) GetVehicleAccess(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), helpers.UserCtxKey, c.MustGet(helpers.UserCtxKey).(*users.User))
	vehicleUUID := c.MustGet(helpers.UUIDParamKey).(uuid.UUID)

	accesses, err := h.shareService.GetAccessForVehicle(ctx, vehicleUUID)
	switch {
	case err == nil:
	case errors.Is(err, gorm.ErrRecordNotFound):
		slog.Get(c).InfoContext(ctx, "vehicle not found", "vehicle_id", vehicleUUID)
		c.JSON(http.StatusNotFound, gin.H{"error": "Vehicle not found"})
		return
	default:
		slog.Get(c).ErrorContext(ctx, "failed to list vehicle access", "vehicle_id", vehicleUUID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, accesses)
}

func (h *Handler) UpdateAccessByID(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), helpers.UserCtxKey, c.MustGet(helpers.UserCtxKey).(*users.User))
	accessUUID := c.MustGet(helpers.UUIDParamKey).(uuid.UUID)

	var input VehicleUserAccessUpdatePost
	if err := c.ShouldBindJSON(&input); err != nil {
		slog.Get(c).WarnContext(ctx, "invalid access body", "error", err)
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Invalid request"})
		return
	}

	access := VehicleUserAccess{
		VehicleUserAccessBase: VehicleUserAccessBase{WriteAccess: *input.WriteAccess},
		Version:               input.Version,
	}

	err := h.shareService.UpdateAccessByID(ctx, accessUUID, &access)
	switch {
	case err == nil:
	case errors.Is(err, helpers.ErrVersionConflict):
		body := gin.H{"error": "Version conflict"}
		var conflict *helpers.VersionConflictError
		if errors.As(err, &conflict) {
			body["version"] = conflict.CurrentVersion
		}
		slog.Get(c).WarnContext(ctx, "version conflict updating access", "access_id", accessUUID)
		c.JSON(http.StatusConflict, body)
		return
	case errors.Is(err, helpers.ErrAccessDenied):
		slog.Get(c).WarnContext(ctx, "access denied updating access", "access_id", accessUUID)
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	case errors.Is(err, gorm.ErrRecordNotFound):
		slog.Get(c).InfoContext(ctx, "access not found", "access_id", accessUUID)
		c.JSON(http.StatusNotFound, gin.H{"error": "Access not found"})
		return
	default:
		slog.Get(c).ErrorContext(ctx, "failed to update access", "access_id", accessUUID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, NewVehicleUserAccessWeb(access))
}

func (h *Handler) DeleteAccessByID(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), helpers.UserCtxKey, c.MustGet(helpers.UserCtxKey).(*users.User))
	accessUUID := c.MustGet(helpers.UUIDParamKey).(uuid.UUID)

	err := h.shareService.DeleteAccessByID(ctx, accessUUID)
	switch {
	case err == nil:
	case errors.Is(err, gorm.ErrRecordNotFound):
		slog.Get(c).InfoContext(ctx, "access not found", "access_id", accessUUID)
		c.JSON(http.StatusNotFound, gin.H{"error": "Access not found"})
		return
	default:
		slog.Get(c).ErrorContext(ctx, "failed to revoke access", "access_id", accessUUID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.Status(http.StatusNoContent)
}
