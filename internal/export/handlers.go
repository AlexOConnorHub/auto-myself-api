package export

import (
	"archive/zip"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"auto-myself-api/internal/helpers"
	"auto-myself-api/internal/users"

	"github.com/gin-contrib/slog"
	"github.com/gin-gonic/gin"
)

//go:embed README.md
var exportReadme []byte

type ExportHandler interface {
	Download(*gin.Context)
}

type Handler struct {
	service *Service
}

func NewExportHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) Download(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), helpers.UserCtxKey, c.MustGet(helpers.UserCtxKey).(*users.User))

	archive, err := h.service.Build(ctx)
	if err != nil {
		slog.Get(c).ErrorContext(ctx, "failed to build user export", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	filename := fmt.Sprintf("auto-myself-export-%s-%s.zip", archive.User.ID, time.Now().UTC().Format("20060102"))
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	c.Header("Cache-Control", "no-store")

	zipWriter := zip.NewWriter(c.Writer)
	if err := writeArchive(zipWriter, archive); err != nil {
		slog.Get(c).ErrorContext(ctx, "failed to write user export", "error", err)
	}
	if err := zipWriter.Close(); err != nil {
		slog.Get(c).ErrorContext(ctx, "failed to close user export archive", "error", err)
	}
}

func writeArchive(zipWriter *zip.Writer, archive *Archive) error {
	if err := writeFile(zipWriter, "README.md", exportReadme); err != nil {
		return err
	}

	if err := writeJSON(zipWriter, "user.json", archive.User); err != nil {
		return err
	}

	if err := createDirectory(zipWriter, "vehicles/"); err != nil {
		return err
	}
	for _, vehicle := range archive.Vehicles {
		prefix := "vehicles/" + vehicle.Vehicle.ID.String() + "/"
		if err := createDirectory(zipWriter, prefix); err != nil {
			return err
		}
		if err := writeJSON(zipWriter, prefix+"vehicle.json", vehicle.Vehicle); err != nil {
			return err
		}
		if err := writeJSON(zipWriter, prefix+"maintenance_records.json", vehicle.MaintenanceRecords); err != nil {
			return err
		}
		if err := createDirectory(zipWriter, prefix+"files/"); err != nil {
			return err
		}
		for _, file := range vehicle.Files {
			if err := writeJSON(zipWriter, prefix+"files/"+file.ID.String()+".json", file); err != nil {
				return err
			}
		}
	}

	if len(archive.Reports) > 0 {
		if err := createDirectory(zipWriter, "reports/"); err != nil {
			return err
		}
		for _, report := range archive.Reports {
			if err := writeJSON(zipWriter, "reports/"+report.ID.String()+".json", report); err != nil {
				return err
			}
		}
	}

	return nil
}

func createDirectory(zipWriter *zip.Writer, name string) error {
	_, err := zipWriter.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Store})
	return err
}

func writeJSON(zipWriter *zip.Writer, name string, value any) error {
	writer, err := zipWriter.Create(name)
	if err != nil {
		return fmt.Errorf("creating %s: %w", name, err)
	}
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return fmt.Errorf("encoding %s: %w", name, err)
	}
	return nil
}

func writeFile(zipWriter *zip.Writer, name string, data []byte) error {
	writer, err := zipWriter.Create(name)
	if err != nil {
		return fmt.Errorf("creating %s: %w", name, err)
	}
	if _, err := writer.Write(data); err != nil {
		return fmt.Errorf("writing %s: %w", name, err)
	}
	return nil
}
