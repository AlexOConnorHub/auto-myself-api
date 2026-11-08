package files

import (
	"auto-myself-api/internal/helpers"
	"auto-myself-api/internal/httperr"
	"auto-myself-api/internal/users"
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/gin-contrib/slog"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

// MaxUploadBytes caps a single upload request body at 10 MiB.
const MaxUploadBytes = 10 << 20

type FileHandler interface {
	GetForRecord(*gin.Context)
	CreateForRecord(*gin.Context)
	DeleteByID(*gin.Context)
}

type Handler struct {
	fileService *Service
}

func NewFileHandler(fileService *Service) *Handler {
	return &Handler{
		fileService: fileService,
	}
}

func (h *Handler) GetForRecord(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), helpers.UserCtxKey, c.MustGet(helpers.UserCtxKey).(*users.User))
	recordType := c.Param("record_type")
	recordUUID := c.MustGet(helpers.UUIDParamKey).(uuid.UUID)

	entries, err := h.fileService.GetForRecord(ctx, recordType, recordUUID)
	switch {
	case err == nil:
	case errors.Is(err, ErrUnsupportedRecordType):
		slog.Get(c).WarnContext(ctx, "unsupported file record type", "record_type", recordType)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file type"})
		return
	case errors.Is(err, gorm.ErrRecordNotFound):
		slog.Get(c).InfoContext(ctx, "record not found", "record_type", recordType, "record_id", recordUUID)
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	default:
		slog.Get(c).ErrorContext(ctx, "failed to list files", "record_type", recordType, "record_id", recordUUID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, entries)
}

func (h *Handler) CreateForRecord(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), helpers.UserCtxKey, c.MustGet(helpers.UserCtxKey).(*users.User))
	recordType := c.Param("record_type")
	recordUUID := c.MustGet(helpers.UUIDParamKey).(uuid.UUID)

	body := http.MaxBytesReader(c.Writer, c.Request.Body, MaxUploadBytes)
	c.Request.Body = body

	data, err := readUploadBody(ctx, body)
	if err != nil {
		if ctx.Err() != nil {
			// The timeout middleware may race its own timer: when the handler finishes first
			// it flushes the buffer instead of writing its 408. Writing the same body here
			// makes either branch return the canonical 408 instead of an empty 200.
			httperr.WriteRequestTimeout(c)
			return
		}
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			slog.Get(c).WarnContext(ctx, "file upload too large", "record_type", recordType, "record_id", recordUUID)
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "File too large"})
			return
		}
		slog.Get(c).WarnContext(ctx, "failed to read upload body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file data"})
		return
	}

	if ctx.Err() != nil {
		httperr.WriteRequestTimeout(c)
		return
	}

	var link MaintenanceRecordFile
	err = h.fileService.CreateForRecord(ctx, recordType, recordUUID, data, &link)
	switch {
	case err == nil:
	case errors.Is(err, ErrUnsupportedRecordType):
		slog.Get(c).WarnContext(ctx, "unsupported file record type", "record_type", recordType)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file type"})
		return
	case errors.Is(err, helpers.ErrAccessDenied):
		slog.Get(c).WarnContext(ctx, "access denied uploading file", "record_type", recordType, "record_id", recordUUID)
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	case errors.Is(err, gorm.ErrRecordNotFound):
		slog.Get(c).InfoContext(ctx, "record not found", "record_type", recordType, "record_id", recordUUID)
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	default:
		slog.Get(c).ErrorContext(ctx, "failed to upload file", "record_type", recordType, "record_id", recordUUID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.Header("Location", "/v1/file/"+link.FileID.String())
	c.JSON(http.StatusCreated, link)
}

func (h *Handler) DeleteByID(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), helpers.UserCtxKey, c.MustGet(helpers.UserCtxKey).(*users.User))
	fileUUID := c.MustGet(helpers.UUIDParamKey).(uuid.UUID)

	err := h.fileService.DeleteByID(ctx, fileUUID)
	switch {
	case err == nil:
	case errors.Is(err, helpers.ErrAccessDenied):
		slog.Get(c).WarnContext(ctx, "access denied deleting file", "file_id", fileUUID)
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	case errors.Is(err, gorm.ErrRecordNotFound):
		slog.Get(c).InfoContext(ctx, "file not found", "file_id", fileUUID)
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	default:
		slog.Get(c).ErrorContext(ctx, "failed to delete file", "file_id", fileUUID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.Status(http.StatusNoContent)
}

type uploadReadResult struct {
	data []byte
	err  error
}

func readUploadBody(ctx context.Context, body io.ReadCloser) ([]byte, error) {
	done := make(chan uploadReadResult, 1)
	go func() {
		data, err := io.ReadAll(body)
		done <- uploadReadResult{data: data, err: err}
	}()

	select {
	case <-ctx.Done():
		go func() { _ = body.Close() }()
		return nil, ctx.Err()
	case result := <-done:
		return result.data, result.err
	}
}
