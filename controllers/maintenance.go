package controllers

import (
	"auto-myself-api/database"
	"auto-myself-api/helpers"
	"auto-myself-api/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

// Get maintenance
func GetAllMaintenance(c *gin.Context) {
	var user = c.MustGet("user").(*models.User)

	vehicleUUID, err := uuid.FromString(c.Param("uuid"))

	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	var vehicle models.Vehicle
	err = database.DB.First(&vehicle, "id = ?", vehicleUUID).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			database.LogError(err)
		}
		c.Status(http.StatusNotFound)
		return
	}

	if !vehicle.CanRead(user) {
		c.Status(http.StatusNotFound)
		return
	}

	var maintenanceIdss []string

	database.DB.Model(&vehicle).Association("MaintenanceRecords").Find(&vehicle.MaintenanceRecords)

	for _, MaintenanceRecord := range vehicle.MaintenanceRecords {
		maintenanceIdss = append(maintenanceIdss, MaintenanceRecord.ID.String())
	}

	c.JSON(http.StatusOK, maintenanceIdss)
}

// Get maintenance record
func GetMaintenanceByID(c *gin.Context) {
	var user = c.MustGet("user").(*models.User)

	maintenanceUUID, err := uuid.FromString(c.Param("uuid"))

	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	var maintenanceRecord models.MaintenanceRecord
	err = database.DB.First(&maintenanceRecord, "id = ?", maintenanceUUID).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			database.LogError(err)
		}
		c.Status(http.StatusNotFound)
		return
	}
	database.DB.Model(&maintenanceRecord).Association("Vehicle").Find(&maintenanceRecord.Vehicle)

	if !maintenanceRecord.Vehicle.CanRead(user) {
		c.Status(http.StatusNotFound)
		return
	}

	c.JSON(http.StatusOK, maintenanceRecord.MaintenanceRecordBase)
}

// Create maintenance record
func CreateMaintenance(c *gin.Context) {
	var user = c.MustGet("user").(*models.User)

	var newMaintenanceRecord models.MaintenanceRecord
	if err := c.ShouldBindJSON(&newMaintenanceRecord.MaintenanceRecordBase); err != nil {
		c.Status(http.StatusUnprocessableEntity)
		return
	}

	newMaintenanceRecord.CreatedBy = user.ID

	if err := database.DB.Create(&newMaintenanceRecord).Error; err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Header("Location", "/v1/maintenance/"+newMaintenanceRecord.ID.String())
	c.Status(http.StatusCreated)
}

// Delete maintenance record
func DeleteMaintenanceByID(c *gin.Context) {
	var user = c.MustGet("user").(*models.User)

	maintenanceRecordUUID, err := uuid.FromString(c.Param("uuid"))

	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	var maintenanceRecord models.MaintenanceRecord
	err = database.DB.First(&maintenanceRecord, "id = ?", maintenanceRecordUUID).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			database.LogError(err)
			c.Status(http.StatusInternalServerError)
		} else {
			c.Status(http.StatusNotFound)
		}
		return
	}
	database.DB.Model(&maintenanceRecord).Association("Vehicle").Find(&maintenanceRecord.Vehicle)

	if maintenanceRecord.Vehicle.CreatedBy != user.ID ||
		(maintenanceRecord.CreatedAt.Add(time.Hour*24).After(time.Now()) && maintenanceRecord.CreatedBy != user.ID) {
		if maintenanceRecord.Vehicle.CanRead(user) {
			c.Status(http.StatusForbidden)
		} else {
			c.Status(http.StatusNotFound)
		}
		return
	}

	if err := database.DB.Delete(&maintenanceRecord).Error; err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusNoContent)
}

// Update maintenance record
func UpdateMaintenanceByID(c *gin.Context) {
	var user = c.MustGet("user").(*models.User)

	maintenanceUUID, err := uuid.FromString(c.Param("uuid"))

	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	maintenanceRecord := models.MaintenanceRecord{
		DatabaseMetadata: helpers.DatabaseMetadata{
			ID: maintenanceUUID,
		},
	}

	err = database.DB.First(&maintenanceRecord, "id = ?", maintenanceUUID).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			database.LogError(err)
			c.Status(http.StatusInternalServerError)
		} else {
			c.Status(http.StatusNotFound)
		}
		return
	}
	database.DB.Model(&maintenanceRecord).Association("Vehicle").Find(&maintenanceRecord.Vehicle)

	if !maintenanceRecord.Vehicle.CanWrite(user) {
		if maintenanceRecord.Vehicle.CanRead(user) {
			c.Status(http.StatusForbidden)
		} else {
			c.Status(http.StatusNotFound)
		}
		return
	}

	var input models.MaintenanceRecordBase
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	if err := database.DB.Model(&maintenanceRecord).Updates(input).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update maintenance record"})
		return
	}

	c.JSON(http.StatusOK, maintenanceRecord.MaintenanceRecordBase)
}
