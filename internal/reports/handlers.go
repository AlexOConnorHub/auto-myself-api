package reports

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

type ReportHandler interface {
	Create(*gin.Context)
	List(*gin.Context)
	ListUnresolved(*gin.Context)
	ListByDate(*gin.Context)
	FindByID(*gin.Context)
	UpdateByID(*gin.Context)
}

type Handler struct {
	service *Service
}

func NewReportHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) Create(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), helpers.UserCtxKey, c.MustGet(helpers.UserCtxKey).(*users.User))

	var report Report
	if err := c.ShouldBindJSON(&report.ReportPost); err != nil {
		slog.Get(c).WarnContext(ctx, "invalid report body", "error", err)
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Invalid request"})
		return
	}

	if err := h.service.Create(ctx, &report); err != nil {
		slog.Get(c).ErrorContext(ctx, "failed to create report", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusCreated, report.ReportWeb)
}

func (h *Handler) List(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), helpers.UserCtxKey, c.MustGet(helpers.UserCtxKey).(*users.User))

	reports, err := h.service.List(ctx)
	if err != nil {
		slog.Get(c).ErrorContext(ctx, "failed to list reports", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, reports)
}

func (h *Handler) ListUnresolved(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), helpers.UserCtxKey, c.MustGet(helpers.UserCtxKey).(*users.User))

	reports, err := h.service.ListUnresolved(ctx)
	switch {
	case err == nil:
	case errors.Is(err, helpers.ErrAccessDenied):
		slog.Get(c).WarnContext(ctx, "access denied listing unresolved reports")
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	default:
		slog.Get(c).ErrorContext(ctx, "failed to list unresolved reports", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, reports)
}

func (h *Handler) ListByDate(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), helpers.UserCtxKey, c.MustGet(helpers.UserCtxKey).(*users.User))

	var dateRange DateRange
	if err := c.ShouldBindJSON(&dateRange); err != nil {
		slog.Get(c).WarnContext(ctx, "invalid date range body", "error", err)
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Invalid request"})
		return
	}

	reports, err := h.service.ListByDate(ctx, dateRange)
	switch {
	case err == nil:
	case errors.Is(err, helpers.ErrAccessDenied):
		slog.Get(c).WarnContext(ctx, "access denied listing reports by date")
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	default:
		slog.Get(c).ErrorContext(ctx, "failed to list reports by date", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, reports)
}

func (h *Handler) FindByID(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), helpers.UserCtxKey, c.MustGet(helpers.UserCtxKey).(*users.User))
	reportUUID := c.MustGet(helpers.UUIDParamKey).(uuid.UUID)

	report, err := h.service.FindByID(ctx, reportUUID)
	switch {
	case err == nil:
	case errors.Is(err, gorm.ErrRecordNotFound):
		slog.Get(c).InfoContext(ctx, "report not found", "report_id", reportUUID)
		c.JSON(http.StatusNotFound, gin.H{"error": "Report not found"})
		return
	default:
		slog.Get(c).ErrorContext(ctx, "failed to find report", "report_id", reportUUID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, report.ReportWeb)
}

func (h *Handler) UpdateByID(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), helpers.UserCtxKey, c.MustGet(helpers.UserCtxKey).(*users.User))
	reportUUID := c.MustGet(helpers.UUIDParamKey).(uuid.UUID)

	var input ReportResolutionPost
	if err := c.ShouldBindJSON(&input); err != nil {
		slog.Get(c).WarnContext(ctx, "invalid report resolution body", "error", err)
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Invalid request"})
		return
	}

	report := Report{
		ReportWeb: ReportWeb{
			Resolved:    input.Resolved,
			PublicNotes: input.PublicNotes,
			Version:     input.Version,
		},
		InternalNotes: input.InternalNotes,
	}

	err := h.service.UpdateByID(ctx, reportUUID, &report)
	switch {
	case err == nil:
	case errors.Is(err, helpers.ErrVersionConflict):
		body := gin.H{"error": "Version conflict"}
		var conflict *helpers.VersionConflictError
		if errors.As(err, &conflict) {
			body["version"] = conflict.CurrentVersion
		}
		slog.Get(c).WarnContext(ctx, "version conflict updating report", "report_id", reportUUID)
		c.JSON(http.StatusConflict, body)
		return
	case errors.Is(err, helpers.ErrAccessDenied):
		slog.Get(c).WarnContext(ctx, "access denied updating report", "report_id", reportUUID)
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	case errors.Is(err, gorm.ErrRecordNotFound):
		slog.Get(c).InfoContext(ctx, "report not found", "report_id", reportUUID)
		c.JSON(http.StatusNotFound, gin.H{"error": "Report not found"})
		return
	default:
		slog.Get(c).ErrorContext(ctx, "failed to update report", "report_id", reportUUID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, report.ReportWeb)
}
