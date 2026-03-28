package controllers

import (
	"auto-myself-api/database"
	"auto-myself-api/helpers"
	"auto-myself-api/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

// Get all vehicles for the current user
func GetAllVehicles(c *gin.Context) {
	var user = c.MustGet("user").(*models.User)

	var vehicleIds []string

	database.DB.Model(&user).Association("OwnedVehicles").Find(&user.OwnedVehicles)
	database.DB.Model(&user).Association("AccessedVehicles").Find(&user.AccessedVehicles)

	for _, vehicle := range user.OwnedVehicles {
		vehicleIds = append(vehicleIds, vehicle.ID.String())
	}

	for _, vehicle_user_access := range user.AccessedVehicles {
		database.DB.Model(&vehicle_user_access).Association("Vehicle").Find(&vehicle_user_access.Vehicle)
		vehicleIds = append(vehicleIds, vehicle_user_access.Vehicle.ID.String())
	}

	c.JSON(http.StatusOK, vehicleIds)
}

// Get vehicle
func GetVehicleByID(c *gin.Context) {
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

	if !vehicle.CanRead(user) && !vehicle.CanPendingRead(user) {
		c.Status(http.StatusNotFound)
		return
	}

	c.JSON(http.StatusOK, vehicle.VehicleBase)
}

// Create vehicle
func CreateVehicle(c *gin.Context) {
	var user = c.MustGet("user").(*models.User)

	var newVehicle models.Vehicle
	if err := c.ShouldBindJSON(&newVehicle.VehicleBase); err != nil {
		c.Status(http.StatusUnprocessableEntity)
		return
	}

	newVehicle.CreatedBy = user.ID

	if err := database.DB.Create(&newVehicle).Error; err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Header("Location", "/v1/vehicle/"+newVehicle.ID.String())
	c.Status(http.StatusCreated)
}

// Delete vehicle
func DeleteVehicleByID(c *gin.Context) {
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
			c.Status(http.StatusInternalServerError)
		} else {
			c.Status(http.StatusNotFound)
		}
		return
	}

	if vehicle.CreatedBy != user.ID {
		if vehicle.CanRead(user) {
			c.Status(http.StatusForbidden)
		} else {
			c.Status(http.StatusNotFound)
		}
		return
	}

	if err := database.DB.Delete(&vehicle).Error; err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusNoContent)
}

// Update vehicle
func UpdateVehicleByID(c *gin.Context) {
	var user = c.MustGet("user").(*models.User)

	vehicleUUID, err := uuid.FromString(c.Param("uuid"))

	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	vehicle := models.Vehicle{
		DatabaseMetadata: helpers.DatabaseMetadata{
			ID: vehicleUUID,
		},
	}

	err = database.DB.First(&vehicle, "id = ?", vehicleUUID).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			database.LogError(err)
			c.Status(http.StatusInternalServerError)
		} else {
			c.Status(http.StatusNotFound)
		}
		return
	}

	if !vehicle.CanWrite(user) {
		if vehicle.CanRead(user) {
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

	if err := database.DB.Model(&vehicle).Updates(input).Error; err != nil {
		database.LogError(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update vehicle"})
		return
	}

	c.JSON(http.StatusOK, vehicle.VehicleBase)
}
