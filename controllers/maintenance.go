package controllers

import (
	"auto-myself-api/app"
	"auto-myself-api/database"
	"auto-myself-api/helpers"
	"auto-myself-api/models"
	"net/http"
	"time"

	"github.com/gin-contrib/slog"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
)

func GetAllMaintenance(c *gin.Context, a *app.App) {
	var user = c.MustGet("user").(*models.User)

	vehicleUUID := c.MustGet("uuid_param").(uuid.UUID)

	var vehicle models.Vehicle
	err := a.Gorm.First(&vehicle, "id = ?", vehicleUUID).Error
	if err != nil {
		database.DatabaseFetchError(c, err, "id", vehicleUUID)
		c.Abort()
		return
	}

	if !vehicle.CanRead(a, user) {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	var maintenanceIdss []string

	a.Gorm.Model(&vehicle).Association("MaintenanceRecords").Find(&vehicle.MaintenanceRecords)

	for _, MaintenanceRecord := range vehicle.MaintenanceRecords {
		maintenanceIdss = append(maintenanceIdss, MaintenanceRecord.ID.String())
	}

	c.JSON(http.StatusOK, maintenanceIdss)
}

func GetMaintenanceByID(c *gin.Context, a *app.App) {
	var user = c.MustGet("user").(*models.User)

	maintenanceUUID := c.MustGet("uuid_param").(uuid.UUID)

	var maintenanceRecord models.MaintenanceRecord
	err := a.Gorm.First(&maintenanceRecord, "id = ?", maintenanceUUID).Error
	if err != nil {
		database.DatabaseFetchError(c, err, "id", maintenanceUUID)
		c.Abort()
		return
	}
	a.Gorm.Model(&maintenanceRecord).Association("Vehicle").Find(&maintenanceRecord.Vehicle)

	if !maintenanceRecord.Vehicle.CanRead(a, user) {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	c.JSON(http.StatusOK, maintenanceRecord.MaintenanceRecordBase)
}

func CreateMaintenance(c *gin.Context, a *app.App) {
	var user = c.MustGet("user").(*models.User)

	var newMaintenanceRecord models.MaintenanceRecord
	if err := c.ShouldBindJSON(&newMaintenanceRecord.MaintenanceRecordBase); err != nil {
		slog.Get(c).Warn("Failed to bind JSON", "error", err)
		c.AbortWithStatus(http.StatusUnprocessableEntity)
		return
	}

	newMaintenanceRecord.CreatedBy = user.ID

	userProvidedUUID, exists := c.Get("uuid_param")
	if exists {
		newMaintenanceRecord.ID = userProvidedUUID.(uuid.UUID)
	}

	if err := a.Gorm.Create(&newMaintenanceRecord).Error; err != nil {
		slog.Get(c).Warn("Failed to create record", "error", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.Header("Location", "/v1/maintenance/"+newMaintenanceRecord.ID.String())
	c.Status(http.StatusCreated)
}

func DeleteMaintenanceByID(c *gin.Context, a *app.App) {
	var user = c.MustGet("user").(*models.User)

	maintenanceRecordUUID := c.MustGet("uuid_param").(uuid.UUID)

	var maintenanceRecord models.MaintenanceRecord
	err := a.Gorm.First(&maintenanceRecord, "id = ?", maintenanceRecordUUID).Error
	if err != nil {
		database.DatabaseFetchError(c, err, "id", maintenanceRecordUUID)
		c.Abort()
		return
	}
	a.Gorm.Model(&maintenanceRecord).Association("Vehicle").Find(&maintenanceRecord.Vehicle)

	if maintenanceRecord.Vehicle.CreatedBy != user.ID ||
		(maintenanceRecord.CreatedAt.Add(time.Hour*24).After(time.Now()) && maintenanceRecord.CreatedBy != user.ID) {
		var status int
		if maintenanceRecord.Vehicle.CanRead(a, user) {
			status = http.StatusForbidden
		} else {
			status = http.StatusNotFound
		}
		c.AbortWithStatus(status)
		return
	}

	if err := a.Gorm.Delete(&maintenanceRecord).Error; err != nil {
		slog.Get(c).Warn("Failed to delete record", "error", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusNoContent)
}

func UpdateMaintenanceByID(c *gin.Context, a *app.App) {
	var user = c.MustGet("user").(*models.User)

	maintenanceUUID := c.MustGet("uuid_param").(uuid.UUID)

	maintenanceRecord := models.MaintenanceRecord{
		DatabaseMetadata: helpers.DatabaseMetadata{
			ID: maintenanceUUID,
		},
	}

	err := a.Gorm.First(&maintenanceRecord, "id = ?", maintenanceUUID).Error
	if err != nil {
		database.DatabaseFetchError(c, err, "id", maintenanceUUID)
		c.Abort()
		return
	}
	a.Gorm.Model(&maintenanceRecord).Association("Vehicle").Find(&maintenanceRecord.Vehicle)

	if !maintenanceRecord.Vehicle.CanWrite(a, user) {
		var status int
		if maintenanceRecord.Vehicle.CanRead(a, user) {
			status = http.StatusForbidden
		} else {
			status = http.StatusNotFound
		}
		c.AbortWithStatus(status)
		return
	}

	var input models.MaintenanceRecordBase
	if err := c.ShouldBindJSON(&input); err != nil {
		slog.Get(c).Warn("Failed to bind JSON", "error", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	if err := a.Gorm.Model(&maintenanceRecord).Updates(input).Error; err != nil {
		slog.Get(c).Warn("Failed to update record", "error", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, maintenanceRecord.MaintenanceRecordBase)
}
