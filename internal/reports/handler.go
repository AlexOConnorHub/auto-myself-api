package reports

import (
	"auto-myself-api/internal/helpers"
	"auto-myself-api/internal/users"
	"context"
	"net/http"

	"github.com/gin-contrib/slog"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
)

type ReportHandler interface {
	Create(*gin.Context)
	List(*gin.Context)
	ListByDate(*gin.Context)
	FindByID(*gin.Context)
	UpdateByID(*gin.Context)
}

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) Create(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), helpers.UserCtxKey, c.MustGet(helpers.UserCtxKey).(*users.User))

	report := Report{}
	if err := c.ShouldBindJSON(&report.ReportPost); err != nil {
		slog.Get(c).Warn("Failed to bind JSON", "error", err)
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{"error": "Invalid request"})
		return
	}

	err := h.service.repository.Create(ctx, &report)
	if err != nil {
		slog.Get(c).Error("Failed to create report", "error", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to create report"})
		return
	}

	c.JSON(http.StatusCreated, report.ReportWeb)
}

func (h *Handler) List(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), helpers.UserCtxKey, c.MustGet(helpers.UserCtxKey).(*users.User))

	reports, err := h.service.List(ctx)
	if err != nil {
		slog.Get(c).Error("Failed to list reports", "error", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to list reports"})
		return
	}

	c.JSON(http.StatusOK, reports)
}

func (h *Handler) ListByDate(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), helpers.UserCtxKey, c.MustGet(helpers.UserCtxKey).(*users.User))

	var dateRange DateRange
	if err := c.ShouldBindJSON(&dateRange); err != nil {
		slog.Get(c).Warn("Failed to bind JSON", "error", err)
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{"error": "Invalid request"})
		return
	}

	reports, err := h.service.ListByDate(ctx, dateRange)
	if err != nil {
		slog.Get(c).Error("Failed to list reports by date", "error", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to list reports by date"})
		return
	}

	c.JSON(http.StatusOK, reports)
}

func (h *Handler) FindByID(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), helpers.UserCtxKey, c.MustGet(helpers.UserCtxKey).(*users.User))

	idParam := c.Param("id")
	id, err := uuid.FromString(idParam)
	if err != nil {
		slog.Get(c).Warn("Invalid report ID", "error", err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid report ID"})
		return
	}

	report, err := h.service.FindByID(ctx, id)
	if err != nil {
		slog.Get(c).Error("Failed to find report by ID", "error", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to find report by ID"})
		return
	}

	c.JSON(http.StatusOK, report.ReportWeb)
}

func (h *Handler) UpdateByID(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), helpers.UserCtxKey, c.MustGet(helpers.UserCtxKey).(*users.User))

	idParam := c.Param("id")
	id, err := uuid.FromString(idParam)
	if err != nil {
		slog.Get(c).Warn("Invalid report ID", "error", err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid report ID"})
		return
	}

	var reportPost ReportResolutionPost
	if err := c.ShouldBindJSON(&reportPost); err != nil {
		slog.Get(c).Warn("Failed to bind JSON", "error", err)
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{"error": "Invalid request"})
		return
	}

	report, err := h.service.UpdateByID(ctx, id, &reportPost)
	if err != nil {
		slog.Get(c).Error("Failed to update report by ID", "error", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to update report by ID"})
		return
	}

	c.JSON(http.StatusOK, report.ReportWeb)

}
