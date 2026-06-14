package controllers

import (
	"auto-myself-api/app"
	"auto-myself-api/helpers"
	"auto-myself-api/models"
	"crypto/sha256"
	"encoding/hex"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

func GetFilesForTypeByID(c *gin.Context, a *app.App) {
	fileType := c.Param("record_type")
	uuid := c.MustGet("uuid_param")
	user := c.MustGet("user").(*models.User)

	type fileReturnStruct struct {
		ID         string `json:"id"`
		CreatedAt  string `json:"created_at"`
		UpdatedAt  string `json:"updated_at"`
		Sha256Hash string `json:"sha256_hash"`
		FileSize   int64  `json:"file_size"`
		SignedUrl  string `json:"signed_url"`
	}

	switch fileType {
	case "maintenance":
		var maintenance models.MaintenanceRecord
		err := a.Gorm.
			Preload("Vehicle").
			Preload("FileLink").
			Preload("FileLink.File").
			Where("id = ?", uuid).
			First(&maintenance).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(404, gin.H{"error": "Maintenance record not found"})
			} else {
				c.JSON(500, gin.H{"error": "Database error"})
			}
			return
		}

		if !maintenance.Vehicle.CanRead(a, user) {
			c.JSON(404, gin.H{"error": "Maintenance record not found"})
			return
		}

		var final []fileReturnStruct

		for _, file_link := range maintenance.FileLink {
			signedUrl, err := helpers.GetFileUrl(file_link.File.StorageKey)
			if err != nil {
				c.JSON(500, gin.H{"error": "Failed to generate signed URL"})
				println(err.Error())
				println("Failed to generate signed URL for file ID: " + file_link.File.StorageKey)
				return
			}

			final = append(final, fileReturnStruct{
				ID:         file_link.File.ID.String(),
				CreatedAt:  file_link.File.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
				UpdatedAt:  file_link.File.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
				Sha256Hash: file_link.File.Sha256Hash,
				FileSize:   file_link.File.FileSize,
				SignedUrl:  signedUrl,
			})
		}

		c.JSON(200, final)
		return

	default:
		c.JSON(400, gin.H{"error": "Invalid file type"})
		return
	}
}

func CreateFileForTypeByID(c *gin.Context, a *app.App) {
	fileType := c.Param("record_type")
	uuid := c.MustGet("uuid_param").(uuid.UUID)
	user := c.MustGet("user").(*models.User)

	if fileType == "maintenance" {
		var maintenance models.MaintenanceRecord
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

		var file models.File
		file.CreatedBy = user.ID
		file.Sha256Hash = hex.EncodeToString(hash[:])
		file.FileSize = int64(len(data))

		if err := a.Gorm.Create(&file).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to create file record"})
			return
		}
		file.StorageKey = models.RelationString + "/" + file.ID.String()
		if err := a.Gorm.Save(&file).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to create file record"})
			return
		}

		var mrFile models.MaintenanceRecordFile
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

func DeleteFileForTypeByID(c *gin.Context, a *app.App) {
	fileType := c.Param("record_type")
	uuid := c.MustGet("uuid_param").(uuid.UUID)
	user := c.MustGet("user").(*models.User)

	if fileType == "direct" {
		var file models.File
		if err := a.Gorm.Where("id = ?", uuid).First(&file).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(404, gin.H{"error": "File not found"})
			} else {
				c.JSON(500, gin.H{"error": "Database error"})
			}
			return
		}

		var maintenanceRecordLink models.MaintenanceRecordFile
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

	c.JSON(400, gin.H{"error": "Invalid file type"})
}
