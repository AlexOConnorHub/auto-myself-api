package maintenance

import (
	"auto-myself-api/internal/helpers"
	"auto-myself-api/internal/users"
	"context"
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
	vehicleUUID := c.MustGet("uuid_param").(uuid.UUID)

	maintenanceUUIDs, err := h.maintenanceService.FindIDs(ctx, vehicleUUID)

	switch err {
	case nil:
		// No error, continue
	case helpers.ErrAccessDenied:
		slog.Get(c).WarnContext(ctx, "Access denied", "error", err)
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	case gorm.ErrRecordNotFound:
		slog.Get(c).WarnContext(ctx, "Vehicle not found", "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Vehicle not found"})
		return
	default:
		slog.Get(c).ErrorContext(ctx, "Unexpected error", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, maintenanceUUIDs)
	return

}

func (h *Handler) FindByID(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), helpers.UserCtxKey, c.MustGet(helpers.UserCtxKey).(*users.User))
	maintenanceUUID := c.MustGet("uuid_param").(uuid.UUID)

	maintenanceRecord, err := h.maintenanceService.FindByID(ctx, maintenanceUUID)
	switch err {
	case nil:
		// No error, continue
	case helpers.ErrAccessDenied:
		slog.Get(c).WarnContext(ctx, "Access denied", "error", err)
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	case gorm.ErrRecordNotFound:
		slog.Get(c).WarnContext(ctx, "Maintenance record not found", "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Maintenance record not found"})
		return
	default:
		slog.Get(c).ErrorContext(ctx, "Unexpected error", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, maintenanceRecord.MaintenanceRecordWeb)
}

func (h *Handler) Create(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), helpers.UserCtxKey, c.MustGet(helpers.UserCtxKey).(*users.User))

	var maintenanceRecord MaintenanceRecord
	if err := c.ShouldBindJSON(&maintenanceRecord.MaintenanceRecordWeb.MaintenanceRecordPost); err != nil {
		slog.Get(c).Warn("Failed to bind JSON", "error", err)
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Invalid request"})
		return
	}

	if err := h.maintenanceService.Create(ctx, &maintenanceRecord); err != nil {
		slog.Get(c).ErrorContext(ctx, "Failed to create maintenance record", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusCreated, maintenanceRecord.MaintenanceRecordWeb)
}

func (h *Handler) DeleteByID(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), helpers.UserCtxKey, c.MustGet(helpers.UserCtxKey).(*users.User))
	maintenanceRecordUUID := c.MustGet("uuid_param").(uuid.UUID)

	if err := h.maintenanceService.DeleteByID(ctx, maintenanceRecordUUID); err != nil {
		switch err {
		case nil:
			// No error, continue
		case helpers.ErrAccessDenied:
			slog.Get(c).WarnContext(ctx, "Access denied", "error", err)
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		case gorm.ErrRecordNotFound:
			slog.Get(c).WarnContext(ctx, "Maintenance record not found", "error", err)
			c.JSON(http.StatusNotFound, gin.H{"error": "Maintenance record not found"})
			return
		default:
			slog.Get(c).ErrorContext(ctx, "Unexpected error", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) UpdateByID(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), helpers.UserCtxKey, c.MustGet(helpers.UserCtxKey).(*users.User))
	maintenanceUUID := c.MustGet("uuid_param").(uuid.UUID)

	var maintenanceRecord MaintenanceRecord
	if err := c.ShouldBindJSON(&maintenanceRecord.MaintenanceRecordWeb.MaintenanceRecordPost); err != nil {
		slog.Get(c).Warn("Failed to bind JSON", "error", err)
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Invalid request"})
		return
	}

	err := h.maintenanceService.UpdateByID(ctx, maintenanceUUID, &maintenanceRecord)

	switch err {
	case nil:
		// No error, continue
	case helpers.ErrAccessDenied:
		slog.Get(c).WarnContext(ctx, "Access denied", "error", err)
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	case gorm.ErrRecordNotFound:
		slog.Get(c).WarnContext(ctx, "Maintenance record not found", "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Maintenance record not found"})
		return
	default:
		slog.Get(c).ErrorContext(ctx, "Unexpected error", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, maintenanceRecord.MaintenanceRecordWeb)
}
