package files

import (
	"auto-myself-api/internal/helpers"
	"auto-myself-api/internal/server"
	"auto-myself-api/internal/users"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/gin-contrib/slog"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

type FileHandler interface {
	GetIDsForRecord(c *gin.Context)
	GetByID(c *gin.Context)
	CreateForRecordByID(c *gin.Context)
	DeleteByID(c *gin.Context)
}

type Handler struct {
	service Service
}

func NewFileHandler(service Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) GetIDsForRecord(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), helpers.UserCtxKey, c.MustGet(helpers.UserCtxKey).(*users.User))
	fileType := c.Param("record_type")
	uuid := c.MustGet("uuid_param")

	ids, err := h.service.GetIDsForRecord(ctx, fileType, uuid)

	switch err {
	case helpers.ErrAccessDenied:
		slog.Get(c).Warn("Access denied for user attempting to get IDs for record")
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	case gorm.ErrRecordNotFound:
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	case nil:
		// no error, continue
	default:
		slog.Get(c).Error("Database error while getting IDs for record: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"file_ids": ids})
}

func CreateFileForTypeByID(c *gin.Context, a *server.App) {
	fileType := c.Param("record_type")
	uuid := c.MustGet("uuid_param").(uuid.UUID)
	user := c.MustGet("user").(*users.User)

	if fileType == "maintenance" {
		var maintenance MaintenanceRecord
		if err := a.Gorm.Where("id = ?", uuid).First(&maintenance).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(404, gin.H{"error": "Maintenance record not found"})
			} else {
				c.JSON(500, gin.H{"error": "Database error"})
			}
			return
		}

		a.Gorm.Model(&maintenance).Association("Vehicle").Find(&maintenance.Vehicle)

		if !maintenance.Vehicle.CanWrite(a, user) {
			if !maintenance.Vehicle.CanRead(a, user) {
				c.JSON(404, gin.H{"error": "Maintenance record not found"})
			} else {
				c.JSON(403, gin.H{"error": "Forbidden"})
			}
			return
		}

		data, err := c.GetRawData()
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid file data"})
			return
		}

		hash := sha256.Sum256(data)

		var file File
		file.CreatedBy = user.ID
		file.Sha256Hash = hex.EncodeToString(hash[:])
		file.FileSize = int64(len(data))

		if err := a.Gorm.Create(&file).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to create file record"})
			return
		}
		file.StorageKey = fmt.Sprintf("maintenance_record/%s", file.ID.String())
		if err := a.Gorm.Save(&file).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to create file record"})
			return
		}

		var mrFile MaintenanceRecordFile
		mrFile.MaintenanceRecordID = uuid
		mrFile.FileID = file.ID
		if err := a.Gorm.Create(&mrFile).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to create file record"})
			return
		}

		if err := file.UploadFile(a, &data); err != nil {
			c.JSON(500, gin.H{"error": "Failed to upload file"})
			println("Failed to upload file: " + err.Error())
			return
		}

		c.JSON(201, mrFile)
		return
	} else {
		c.JSON(400, gin.H{"error": "Invalid file type"})
		println("Invalid file type: " + fileType)
		return
	}
}

func DeleteFileByID(c *gin.Context, a *server.App) {
	uuid := c.MustGet("uuid_param").(uuid.UUID)
	user := c.MustGet("user").(*users.User)

	var file File
	if err := a.Gorm.Where("id = ?", uuid).First(&file).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(404, gin.H{"error": "File not found"})
		} else {
			c.JSON(500, gin.H{"error": "Database error"})
		}
		return
	}

	var maintenanceRecordLink MaintenanceRecordFile
	if err := a.Gorm.Where("file_id = ?", file.ID).First(&maintenanceRecordLink).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			c.JSON(500, gin.H{"error": "Database error"})
			return
		}
	}
	if !maintenanceRecordLink.ID.IsNil() { // Is maintenance record file
		a.Gorm.Model(&maintenanceRecordLink).Association("MaintenanceRecord").Find(&maintenanceRecordLink.MaintenanceRecord)
		a.Gorm.Model(&maintenanceRecordLink.MaintenanceRecord).Association("Vehicle").Find(&maintenanceRecordLink.MaintenanceRecord.Vehicle)

		if !maintenanceRecordLink.MaintenanceRecord.Vehicle.CanWrite(a, user) {
			if !maintenanceRecordLink.MaintenanceRecord.Vehicle.CanRead(a, user) {
				c.JSON(404, gin.H{"error": "File not found"})
			} else {
				c.JSON(403, gin.H{"error": "Forbidden"})
			}
			return
		}

		if err := maintenanceRecordLink.DeleteWithFile(a); err != nil {
			c.JSON(500, gin.H{"error": "Failed to delete file from storage"})
			return
		}

		c.Status(204)
		return
	}
}
