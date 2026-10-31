package maintenance

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

type MaintenanceHandler interface {
	FindIDs(*gin.Context)
	FindByID(*gin.Context)
	Create(*gin.Context)
	DeleteByID(*gin.Context)
	UpdateByID(*gin.Context)
}

type Handler struct {
	maintenanceService *Service
}

func NewMaintenanceHandler(maintenanceService *Service) *Handler {
	return &Handler{
		maintenanceService: maintenanceService,
	}
}

func (h *Handler) FindIDs(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), helpers.UserCtxKey, c.MustGet(helpers.UserCtxKey).(*users.User))
	vehicleUUID := c.MustGet(helpers.UUIDParamKey).(uuid.UUID)

	maintenanceUUIDs, err := h.maintenanceService.FindIDs(ctx, vehicleUUID)
	switch {
	case err == nil:
	case errors.Is(err, gorm.ErrRecordNotFound):
		slog.Get(c).InfoContext(ctx, "vehicle not found", "vehicle_id", vehicleUUID)
		c.JSON(http.StatusNotFound, gin.H{"error": "Vehicle not found"})
		return
	default:
		slog.Get(c).ErrorContext(ctx, "failed to find maintenance record IDs", "vehicle_id", vehicleUUID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, maintenanceUUIDs)
}

func (h *Handler) FindByID(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), helpers.UserCtxKey, c.MustGet(helpers.UserCtxKey).(*users.User))
	maintenanceUUID := c.MustGet(helpers.UUIDParamKey).(uuid.UUID)

	maintenanceRecord, err := h.maintenanceService.FindByID(ctx, maintenanceUUID)
	switch {
	case err == nil:
	case errors.Is(err, gorm.ErrRecordNotFound):
		slog.Get(c).InfoContext(ctx, "maintenance record not found", "maintenance_id", maintenanceUUID)
		c.JSON(http.StatusNotFound, gin.H{"error": "Maintenance record not found"})
		return
	default:
		slog.Get(c).ErrorContext(ctx, "failed to find maintenance record", "maintenance_id", maintenanceUUID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, maintenanceRecord.MaintenanceRecordWeb)
}

func (h *Handler) Create(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), helpers.UserCtxKey, c.MustGet(helpers.UserCtxKey).(*users.User))

	var maintenanceRecord MaintenanceRecord
	if err := c.ShouldBindJSON(&maintenanceRecord.MaintenanceRecordWeb.MaintenanceRecordPost); err != nil {
		slog.Get(c).WarnContext(ctx, "invalid maintenance record body", "error", err)
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Invalid request"})
		return
	}

	if suppliedID, exists := c.Get(helpers.UUIDParamKey); exists {
		maintenanceRecord.ID = suppliedID.(uuid.UUID)
	}

	err := h.maintenanceService.Create(ctx, &maintenanceRecord)
	switch {
	case err == nil:
	case errors.Is(err, helpers.ErrAccessDenied):
		slog.Get(c).WarnContext(ctx, "access denied creating maintenance record", "vehicle_id", maintenanceRecord.VehicleID)
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	case errors.Is(err, gorm.ErrRecordNotFound):
		slog.Get(c).InfoContext(ctx, "vehicle not found", "vehicle_id", maintenanceRecord.VehicleID)
		c.JSON(http.StatusNotFound, gin.H{"error": "Vehicle not found"})
		return
	case errors.Is(err, gorm.ErrDuplicatedKey):
		slog.Get(c).WarnContext(ctx, "maintenance record already exists", "maintenance_id", maintenanceRecord.ID)
		c.JSON(http.StatusConflict, gin.H{"error": "Maintenance record already exists"})
		return
	default:
		slog.Get(c).ErrorContext(ctx, "failed to create maintenance record", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.Header("Location", "/v1/maintenance/"+maintenanceRecord.ID.String())
	c.Status(http.StatusCreated)
}

func (h *Handler) DeleteByID(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), helpers.UserCtxKey, c.MustGet(helpers.UserCtxKey).(*users.User))
	maintenanceRecordUUID := c.MustGet(helpers.UUIDParamKey).(uuid.UUID)

	err := h.maintenanceService.DeleteByID(ctx, maintenanceRecordUUID)
	switch {
	case err == nil:
	case errors.Is(err, helpers.ErrAccessDenied):
		slog.Get(c).WarnContext(ctx, "access denied deleting maintenance record", "maintenance_id", maintenanceRecordUUID)
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	case errors.Is(err, gorm.ErrRecordNotFound):
		slog.Get(c).InfoContext(ctx, "maintenance record not found", "maintenance_id", maintenanceRecordUUID)
		c.JSON(http.StatusNotFound, gin.H{"error": "Maintenance record not found"})
		return
	default:
		slog.Get(c).ErrorContext(ctx, "failed to delete maintenance record", "maintenance_id", maintenanceRecordUUID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) UpdateByID(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), helpers.UserCtxKey, c.MustGet(helpers.UserCtxKey).(*users.User))
	maintenanceUUID := c.MustGet(helpers.UUIDParamKey).(uuid.UUID)

	var input MaintenanceRecordUpdatePost
	if err := c.ShouldBindJSON(&input); err != nil {
		slog.Get(c).WarnContext(ctx, "invalid maintenance record body", "error", err)
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Invalid request"})
		return
	}

	maintenanceRecord := MaintenanceRecord{
		MaintenanceRecordWeb: MaintenanceRecordWeb{
			MaintenanceRecordPost: MaintenanceRecordPost{
				MaintenanceRecordFields: input.MaintenanceRecordFields,
			},
		},
	}

	err := h.maintenanceService.UpdateByID(ctx, maintenanceUUID, &maintenanceRecord)
	switch {
	case err == nil:
	case errors.Is(err, helpers.ErrAccessDenied):
		slog.Get(c).WarnContext(ctx, "access denied updating maintenance record", "maintenance_id", maintenanceUUID)
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	case errors.Is(err, gorm.ErrRecordNotFound):
		slog.Get(c).InfoContext(ctx, "maintenance record not found", "maintenance_id", maintenanceUUID)
		c.JSON(http.StatusNotFound, gin.H{"error": "Maintenance record not found"})
		return
	default:
		slog.Get(c).ErrorContext(ctx, "failed to update maintenance record", "maintenance_id", maintenanceUUID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, maintenanceRecord.MaintenanceRecordWeb)
}
