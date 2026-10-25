package vehicles

import (
	"auto-myself-api/internal/helpers"
	"auto-myself-api/internal/users"
	"context"
	"net/http"

	"github.com/gin-contrib/slog"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
)

type VehicleHandler interface {
	FindIDs(*gin.Context)
	FindByID(*gin.Context)
	UpdateByID(*gin.Context)
	Create(*gin.Context)
	DeleteByID(*gin.Context)
}

type Handler struct {
	Service *Service
}

func NewVehicleHandler(Service *Service) *Handler {
	return &Handler{
		Service: Service,
	}
}

func (h *Handler) FindIDs(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), helpers.UserCtxKey, c.Value(helpers.UserCtxKey).(*users.User))

	vehicleIDs, err := h.Service.FindIDs(ctx)

	if err != nil {
		slog.Get(c).ErrorContext(ctx, "Failed to find vehicle IDs", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"vehicle_ids": vehicleIDs})
}

func (h *Handler) FindByID(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), helpers.UserCtxKey, c.Value(helpers.UserCtxKey).(*users.User))
	vehicleUUID := c.MustGet("uuid_param").(uuid.UUID)

	vehicle, err := h.Service.FindByID(ctx, vehicleUUID)
	if err != nil {
		slog.Get(c).ErrorContext(ctx, "Failed to find vehicle by ID", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	c.JSON(http.StatusOK, vehicle.VehicleWeb)
}

func (h *Handler) Create(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), helpers.UserCtxKey, c.Value(helpers.UserCtxKey).(*users.User))

	var newVehiclePost VehiclePost
	if err := c.ShouldBindJSON(&newVehiclePost); err != nil {
		slog.Get(c).Warn("Failed to bind JSON", "error", err)
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Invalid request"})
		return
	}

	newVehicle := Vehicle{VehicleWeb: VehicleWeb{VehiclePost: newVehiclePost}}

	err := h.Service.Create(ctx, &newVehicle)

	if err != nil {
		slog.Get(c).ErrorContext(ctx, "Failed to create vehicle", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	c.Header("Location", "/v1/vehicle/"+newVehicle.ID.String())
	c.Status(http.StatusCreated)
}

func (h *Handler) DeleteByID(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), helpers.UserCtxKey, c.Value(helpers.UserCtxKey).(*users.User))
	vehicleUUID := c.MustGet("uuid_param").(uuid.UUID)

	err := h.Service.DeleteByID(ctx, vehicleUUID)
	if err != nil {
		slog.Get(c).Warn("Failed to delete vehicle", "error", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) UpdateByID(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), helpers.UserCtxKey, c.Value(helpers.UserCtxKey).(*users.User))
	vehicleUUID := c.MustGet("uuid_param").(uuid.UUID)

	var vehiclePost VehiclePost
	if err := c.ShouldBindJSON(&vehiclePost); err != nil {
		slog.Get(c).Warn("Failed to bind JSON", "error", err)
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Invalid request"})
		return
	}

	vehicle := Vehicle{VehicleWeb: VehicleWeb{VehiclePost: vehiclePost}}

	err := h.Service.UpdateByID(ctx, vehicleUUID, &vehicle)

	if err != nil {
		slog.Get(c).Warn("Failed to update vehicle", "error", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, vehicle.VehicleWeb)
}
