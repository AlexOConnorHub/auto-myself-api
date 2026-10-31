package vehicles

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

type VehicleHandler interface {
	FindIDs(*gin.Context)
	FindByID(*gin.Context)
	UpdateByID(*gin.Context)
	Create(*gin.Context)
	DeleteByID(*gin.Context)
}

type Handler struct {
	vehicleService *Service
}

func NewVehicleHandler(vehicleService *Service) *Handler {
	return &Handler{
		vehicleService: vehicleService,
	}
}

func (h *Handler) FindIDs(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), helpers.UserCtxKey, c.MustGet(helpers.UserCtxKey).(*users.User))

	vehicleIDs, err := h.vehicleService.FindIDs(ctx)
	if err != nil {
		slog.Get(c).ErrorContext(ctx, "failed to find vehicle IDs", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, vehicleIDs)
}

func (h *Handler) FindByID(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), helpers.UserCtxKey, c.MustGet(helpers.UserCtxKey).(*users.User))
	vehicleUUID := c.MustGet(helpers.UUIDParamKey).(uuid.UUID)

	vehicle, err := h.vehicleService.FindByID(ctx, vehicleUUID)
	switch {
	case err == nil:
	case errors.Is(err, gorm.ErrRecordNotFound):
		slog.Get(c).InfoContext(ctx, "vehicle not found", "vehicle_id", vehicleUUID)
		c.JSON(http.StatusNotFound, gin.H{"error": "Vehicle not found"})
		return
	default:
		slog.Get(c).ErrorContext(ctx, "failed to find vehicle", "vehicle_id", vehicleUUID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, vehicle.VehicleWeb)
}

func (h *Handler) Create(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), helpers.UserCtxKey, c.MustGet(helpers.UserCtxKey).(*users.User))

	var vehiclePost VehiclePost
	if err := c.ShouldBindJSON(&vehiclePost); err != nil {
		slog.Get(c).WarnContext(ctx, "invalid vehicle body", "error", err)
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Invalid request"})
		return
	}

	newVehicle := Vehicle{VehicleWeb: VehicleWeb{VehiclePost: vehiclePost}}
	if suppliedID, exists := c.Get(helpers.UUIDParamKey); exists {
		newVehicle.ID = suppliedID.(uuid.UUID)
	}

	err := h.vehicleService.Create(ctx, &newVehicle)
	switch {
	case err == nil:
	case errors.Is(err, gorm.ErrDuplicatedKey):
		slog.Get(c).WarnContext(ctx, "vehicle already exists", "vehicle_id", newVehicle.ID)
		c.JSON(http.StatusConflict, gin.H{"error": "Vehicle already exists"})
		return
	default:
		slog.Get(c).ErrorContext(ctx, "failed to create vehicle", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.Header("Location", "/v1/vehicle/"+newVehicle.ID.String())
	c.Status(http.StatusCreated)
}

func (h *Handler) DeleteByID(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), helpers.UserCtxKey, c.MustGet(helpers.UserCtxKey).(*users.User))
	vehicleUUID := c.MustGet(helpers.UUIDParamKey).(uuid.UUID)

	err := h.vehicleService.DeleteByID(ctx, vehicleUUID)
	switch {
	case err == nil:
	case errors.Is(err, helpers.ErrAccessDenied):
		slog.Get(c).WarnContext(ctx, "access denied deleting vehicle", "vehicle_id", vehicleUUID)
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	case errors.Is(err, gorm.ErrRecordNotFound):
		slog.Get(c).InfoContext(ctx, "vehicle not found", "vehicle_id", vehicleUUID)
		c.JSON(http.StatusNotFound, gin.H{"error": "Vehicle not found"})
		return
	default:
		slog.Get(c).ErrorContext(ctx, "failed to delete vehicle", "vehicle_id", vehicleUUID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) UpdateByID(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), helpers.UserCtxKey, c.MustGet(helpers.UserCtxKey).(*users.User))
	vehicleUUID := c.MustGet(helpers.UUIDParamKey).(uuid.UUID)

	var input VehicleUpdatePost
	if err := c.ShouldBindJSON(&input); err != nil {
		slog.Get(c).WarnContext(ctx, "invalid vehicle body", "error", err)
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Invalid request"})
		return
	}

	vehicle := Vehicle{VehicleWeb: VehicleWeb{VehiclePost: input.VehiclePost, Version: input.Version}}

	err := h.vehicleService.UpdateByID(ctx, vehicleUUID, &vehicle)
	switch {
	case err == nil:
	case errors.Is(err, helpers.ErrVersionConflict):
		body := gin.H{"error": "Version conflict"}
		var conflict *helpers.VersionConflictError
		if errors.As(err, &conflict) {
			body["version"] = conflict.CurrentVersion
		}
		slog.Get(c).WarnContext(ctx, "version conflict updating vehicle", "vehicle_id", vehicleUUID)
		c.JSON(http.StatusConflict, body)
		return
	case errors.Is(err, helpers.ErrAccessDenied):
		slog.Get(c).WarnContext(ctx, "access denied updating vehicle", "vehicle_id", vehicleUUID)
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	case errors.Is(err, gorm.ErrRecordNotFound):
		slog.Get(c).InfoContext(ctx, "vehicle not found", "vehicle_id", vehicleUUID)
		c.JSON(http.StatusNotFound, gin.H{"error": "Vehicle not found"})
		return
	default:
		slog.Get(c).ErrorContext(ctx, "failed to update vehicle", "vehicle_id", vehicleUUID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, vehicle.VehicleWeb)
}
