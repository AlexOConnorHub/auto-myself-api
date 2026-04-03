package controllers

import (
	"auto-myself-api/app"
	"auto-myself-api/database"
	"auto-myself-api/helpers"
	"auto-myself-api/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

func GetAllVehicles(c *gin.Context, a *app.App) {
	var user = c.MustGet("user").(*models.User)

	var vehicleIds []string

	a.Gorm.Model(&user).Association("OwnedVehicles").Find(&user.OwnedVehicles)
	a.Gorm.Model(&user).Association("AccessedVehicles").Find(&user.AccessedVehicles)

	for _, vehicle := range user.OwnedVehicles {
		vehicleIds = append(vehicleIds, vehicle.ID.String())
	}

	for _, vehicle_user_access := range user.AccessedVehicles {
		a.Gorm.Model(&vehicle_user_access).Association("Vehicle").Find(&vehicle_user_access.Vehicle)
		vehicleIds = append(vehicleIds, vehicle_user_access.Vehicle.ID.String())
	}

	c.JSON(http.StatusOK, vehicleIds)
}

func GetVehicleByID(c *gin.Context, a *app.App) {
	var user = c.MustGet("user").(*models.User)

	vehicleUUID := c.MustGet("uuid_param").(uuid.UUID)

	var vehicle models.Vehicle
	err := a.Gorm.First(&vehicle, "id = ?", vehicleUUID).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			database.LogError(err)
		}
		c.Status(http.StatusNotFound)
		return
	}

	if !vehicle.CanRead(a, user) && !vehicle.CanPendingRead(a, user) {
		c.Status(http.StatusNotFound)
		return
	}

	c.JSON(http.StatusOK, vehicle.VehicleBase)
}

func CreateVehicle(c *gin.Context, a *app.App) {
	var user = c.MustGet("user").(*models.User)

	var newVehicle models.Vehicle
	if err := c.ShouldBindJSON(&newVehicle.VehicleBase); err != nil {
		c.Status(http.StatusUnprocessableEntity)
		return
	}

	newVehicle.CreatedBy = user.ID

	vehicleUUID, exists := c.Get("uuid_param")
	if exists {
		newVehicle.ID = vehicleUUID.(uuid.UUID)
	}

	if err := a.Gorm.Create(&newVehicle).Error; err != nil {
		database.LogError(err)
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Header("Location", "/v1/vehicle/"+newVehicle.ID.String())
	c.Status(http.StatusCreated)
}

func DeleteVehicleByID(c *gin.Context, a *app.App) {
	var user = c.MustGet("user").(*models.User)

	vehicleUUID := c.MustGet("uuid_param").(uuid.UUID)

	var vehicle models.Vehicle
	err := a.Gorm.First(&vehicle, "id = ?", vehicleUUID).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			database.LogError(err)
			c.Status(http.StatusInternalServerError)
		} else {
			c.Status(http.StatusNotFound)
		}
		return
	}

	if vehicle.CreatedBy != user.ID {
		if vehicle.CanRead(a, user) {
			c.Status(http.StatusForbidden)
		} else {
			c.Status(http.StatusNotFound)
		}
		return
	}

	if err := a.Gorm.Delete(&vehicle).Error; err != nil {
		database.LogError(err)
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusNoContent)
}

func UpdateVehicleByID(c *gin.Context, a *app.App) {
	var user = c.MustGet("user").(*models.User)

	vehicleUUID := c.MustGet("uuid_param").(uuid.UUID)

	vehicle := models.Vehicle{
		DatabaseMetadata: helpers.DatabaseMetadata{
			ID: vehicleUUID,
		},
	}

	err := a.Gorm.First(&vehicle, "id = ?", vehicleUUID).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			database.LogError(err)
			c.Status(http.StatusInternalServerError)
		} else {
			c.Status(http.StatusNotFound)
		}
		return
	}

	if !vehicle.CanWrite(a, user) {
		if vehicle.CanRead(a, user) {
			c.Status(http.StatusForbidden)
		} else {
			c.Status(http.StatusNotFound)
		}
		return
	}

	var input models.VehicleBase
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	if err := a.Gorm.Model(&vehicle).Updates(input).Error; err != nil {
		database.LogError(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update vehicle"})
		return
	}

	c.JSON(http.StatusOK, vehicle.VehicleBase)
}
