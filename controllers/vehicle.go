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

// @Summary Get all vehicles for the current user
// @Description Returns a list of all vehicle's locations associated with the current user.
// @Tags Vehicles
// @Produce json
// @Success 200 {object} []string
// @Param auth_uuid header string true "User ID"
// @Param.examples auth_uuid user1 summary User 1
// @Param.examples auth_uuid user1 description User has One personal vehicle and one shared vehicle
// @Param.examples auth_uuid user1 value 019785fe-4eb4-766e-9c45-bec7780972a2
// @Param.examples auth_uuid user2 summary User 2
// @Param.examples auth_uuid user2 description User has vehicle shared FROM User 1 with write access
// @Param.examples auth_uuid user2 value 019785fe-4eb4-766e-9c45-c1f83e7c1f1f
// @Param.examples auth_uuid user3 summary User 3
// @Param.examples auth_uuid user3 description User has vehicle shared FROM User 1 with read access
// @Param.examples auth_uuid user3 value 019785fe-4eb4-766e-9c45-c497f2d9fe9e
// @Param.examples auth_uuid user4 summary User 4
// @Param.examples auth_uuid user4 description User has One personal vehicle
// @Param.examples auth_uuid user4 value 019785fe-4eb4-766e-9c45-c8578456b4df
// @Param.examples auth_uuid user5 summary User 5
// @Param.examples auth_uuid user5 description User has no vehicles, no vehicles shared
// @Param.examples auth_uuid user5 value 019785fe-4eb4-766e-9c45-cec136a9ad6f
// @Param.examples auth_uuid user6 summary User 6
// @Param.examples auth_uuid user6 description User has One vehicle to share
// @Param.examples auth_uuid user6 value 019785fe-4eb4-766e-9c45-f592a1187d0c
// @Param.examples auth_uuid user7 summary User 7
// @Param.examples auth_uuid user7 description User has vehicle shared FROM User 1 and User 6, both with write access
// @Param.examples auth_uuid user7 value 019785fe-4eb4-766e-9c45-f9cd4ee5c0b3
// @Param.examples auth_uuid user8 summary User 8
// @Param.examples auth_uuid user8 description User has One personal vehicle, vehicle shared FROM User 1 (write) and User 6 (read)
// @Param.examples auth_uuid user8 value 019785fe-4eb4-766e-9c45-fc6ed4a7407b
// @Router /vehicle [get]
func GetAllVehicles(c *gin.Context) {
	var user = c.MustGet("user").(models.User)

	var vehicleLocations []string

	database.DB.Model(&user).Association("OwnedVehicles").Find(&user.OwnedVehicles)
	database.DB.Model(&user).Association("AccessedVehicles").Find(&user.AccessedVehicles)

	for _, vehicle := range user.OwnedVehicles {
		vehicleLocations = append(vehicleLocations, vehicle.GetLocation())
	}

	for _, vehicle_user_access := range user.AccessedVehicles {
		database.DB.Model(&vehicle_user_access).Association("Vehicle").Find(&vehicle_user_access.Vehicle)
		vehicleLocations = append(vehicleLocations, vehicle_user_access.Vehicle.GetLocation())
	}

	c.JSON(http.StatusOK, vehicleLocations)
}

// @Summary Get vehicle
// @Description Retrieves a vehicle by its UUID.
// @Tags Vehicles
// @Produce json
// @Success 200 {object} models.VehicleBase
// @Failure 404
// @Param auth_uuid header string true "User ID"
// @Param.examples auth_uuid user1 summary User 1
// @Param.examples auth_uuid user1 description User has One personal vehicle and one shared vehicle
// @Param.examples auth_uuid user1 value 019785fe-4eb4-766e-9c45-bec7780972a2
// @Param.examples auth_uuid user2 summary User 2
// @Param.examples auth_uuid user2 description User has vehicle shared FROM User 1 with write access
// @Param.examples auth_uuid user2 value 019785fe-4eb4-766e-9c45-c1f83e7c1f1f
// @Param.examples auth_uuid user3 summary User 3
// @Param.examples auth_uuid user3 description User has vehicle shared FROM User 1 with read access
// @Param.examples auth_uuid user3 value 019785fe-4eb4-766e-9c45-c497f2d9fe9e
// @Param.examples auth_uuid user4 summary User 4
// @Param.examples auth_uuid user4 description User has One personal vehicle
// @Param.examples auth_uuid user4 value 019785fe-4eb4-766e-9c45-c8578456b4df
// @Param.examples auth_uuid user5 summary User 5
// @Param.examples auth_uuid user5 description User has no vehicles, no vehicles shared
// @Param.examples auth_uuid user5 value 019785fe-4eb4-766e-9c45-cec136a9ad6f
// @Param.examples auth_uuid user6 summary User 6
// @Param.examples auth_uuid user6 description User has One vehicle to share
// @Param.examples auth_uuid user6 value 019785fe-4eb4-766e-9c45-f592a1187d0c
// @Param.examples auth_uuid user7 summary User 7
// @Param.examples auth_uuid user7 description User has vehicle shared FROM User 1 and User 6, both with write access
// @Param.examples auth_uuid user7 value 019785fe-4eb4-766e-9c45-f9cd4ee5c0b3
// @Param.examples auth_uuid user8 summary User 8
// @Param.examples auth_uuid user8 description User has One personal vehicle, vehicle shared FROM User 1 (write) and User 6 (read)
// @Param.examples auth_uuid user8 value 019785fe-4eb4-766e-9c45-fc6ed4a7407b
// @Param uuid path string true "Vehicle UUID"
// @Param.examples uuid vehicle1 summary Vehicle 1
// @Param.examples uuid vehicle1 description Vehicle owned by User 1
// @Param.examples uuid vehicle1 value 019785fe-4eb4-766e-9c45-d0b2bb289b82
// @Param.examples uuid vehicle2 summary Vehicle 2
// @Param.examples uuid vehicle2 description Vehicle shared by User 1 with User 2
// @Param.examples uuid vehicle2 value 019785fe-4eb4-766e-9c45-d77f41aa8317
// @Param.examples uuid vehicle3 summary Vehicle 3
// @Param.examples uuid vehicle3 description Vehicle owned by User 4
// @Param.examples uuid vehicle3 value 019785fe-4eb4-766e-9c45-d9cc7ea628c1
// @Param.examples uuid vehicle4 summary Vehicle 4
// @Param.examples uuid vehicle4 description Vehicle shared by User 6 with User 7
// @Param.examples uuid vehicle4 value 019785fe-4eb4-766e-9c45-ddfb4b2e7210
// @Param.examples uuid vehicle5 summary Vehicle 5
// @Param.examples uuid vehicle5 description Vehicle owned by User 8
// @Param.examples uuid vehicle5 value 019785fe-4eb4-766e-9c45-e1af5010246b
// @Router /vehicle/{uuid} [get]
func GetVehicleByID(c *gin.Context) {
	var user = c.MustGet("user").(models.User)

	vehicleUUID, err := uuid.FromString(c.Param("uuid"))

	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	var vehicle models.Vehicle
	err = database.DB.Where("id = ?", vehicleUUID).First(&vehicle).Error
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

	c.JSON(http.StatusOK, vehicle.VehicleBase)
}

// @Summary Create vehicle TODO: ADD HEADER
// @Description Create a vehicle.
// @Tags Vehicles
// @Success 201
// @Failure 422
// @Param auth_uuid header string true "User ID"
// @Param.examples auth_uuid user1 summary User 1
// @Param.examples auth_uuid user1 description User has One personal vehicle and one shared vehicle
// @Param.examples auth_uuid user1 value 019785fe-4eb4-766e-9c45-bec7780972a2
// @Param.examples auth_uuid user2 summary User 2
// @Param.examples auth_uuid user2 description User has vehicle shared FROM User 1 with write access
// @Param.examples auth_uuid user2 value 019785fe-4eb4-766e-9c45-c1f83e7c1f1f
// @Param.examples auth_uuid user3 summary User 3
// @Param.examples auth_uuid user3 description User has vehicle shared FROM User 1 with read access
// @Param.examples auth_uuid user3 value 019785fe-4eb4-766e-9c45-c497f2d9fe9e
// @Param.examples auth_uuid user4 summary User 4
// @Param.examples auth_uuid user4 description User has One personal vehicle
// @Param.examples auth_uuid user4 value 019785fe-4eb4-766e-9c45-c8578456b4df
// @Param.examples auth_uuid user5 summary User 5
// @Param.examples auth_uuid user5 description User has no vehicles, no vehicles shared
// @Param.examples auth_uuid user5 value 019785fe-4eb4-766e-9c45-cec136a9ad6f
// @Param.examples auth_uuid user6 summary User 6
// @Param.examples auth_uuid user6 description User has One vehicle to share
// @Param.examples auth_uuid user6 value 019785fe-4eb4-766e-9c45-f592a1187d0c
// @Param.examples auth_uuid user7 summary User 7
// @Param.examples auth_uuid user7 description User has vehicle shared FROM User 1 and User 6, both with write access
// @Param.examples auth_uuid user7 value 019785fe-4eb4-766e-9c45-f9cd4ee5c0b3
// @Param.examples auth_uuid user8 summary User 8
// @Param.examples auth_uuid user8 description User has One personal vehicle, vehicle shared FROM User 1 (write) and User 6 (read)
// @Param.examples auth_uuid user8 value 019785fe-4eb4-766e-9c45-fc6ed4a7407b
// @Param vehicle body models.VehicleBase true "New Vehicle"
// @Param.examples vehicle vehicle1 summary Create a Vehicle
// @Param.examples vehicle vehicle1 description Create a new vehicle with nickname "A Fresh Vehicle"
// @Param.examples vehicle vehicle1 value { "nickname": "A Fresh Vehicle" }
// @Router /vehicle [post]
func CreateVehicle(c *gin.Context) {
	var user = c.MustGet("user").(models.User)

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

	c.Header("X-Object-Location", newVehicle.GetLocation())
	c.Status(http.StatusCreated)
}

// @Summary Delete vehicle
// @Description Delete a vehicle.
// @Tags Vehicles
// @Success 204
// @Failure 403
// @Failure 404
// @Param auth_uuid header string true "User ID"
// @Param.examples auth_uuid user1 summary User 1
// @Param.examples auth_uuid user1 description User has One personal vehicle and one shared vehicle
// @Param.examples auth_uuid user1 value 019785fe-4eb4-766e-9c45-bec7780972a2
// @Param.examples auth_uuid user2 summary User 2
// @Param.examples auth_uuid user2 description User has vehicle shared FROM User 1 with write access
// @Param.examples auth_uuid user2 value 019785fe-4eb4-766e-9c45-c1f83e7c1f1f
// @Param.examples auth_uuid user3 summary User 3
// @Param.examples auth_uuid user3 description User has vehicle shared FROM User 1 with read access
// @Param.examples auth_uuid user3 value 019785fe-4eb4-766e-9c45-c497f2d9fe9e
// @Param.examples auth_uuid user4 summary User 4
// @Param.examples auth_uuid user4 description User has One personal vehicle
// @Param.examples auth_uuid user4 value 019785fe-4eb4-766e-9c45-c8578456b4df
// @Param.examples auth_uuid user5 summary User 5
// @Param.examples auth_uuid user5 description User has no vehicles, no vehicles shared
// @Param.examples auth_uuid user5 value 019785fe-4eb4-766e-9c45-cec136a9ad6f
// @Param.examples auth_uuid user6 summary User 6
// @Param.examples auth_uuid user6 description User has One vehicle to share
// @Param.examples auth_uuid user6 value 019785fe-4eb4-766e-9c45-f592a1187d0c
// @Param.examples auth_uuid user7 summary User 7
// @Param.examples auth_uuid user7 description User has vehicle shared FROM User 1 and User 6, both with write access
// @Param.examples auth_uuid user7 value 019785fe-4eb4-766e-9c45-f9cd4ee5c0b3
// @Param.examples auth_uuid user8 summary User 8
// @Param.examples auth_uuid user8 description User has One personal vehicle, vehicle shared FROM User 1 (write) and User 6 (read)
// @Param.examples auth_uuid user8 value 019785fe-4eb4-766e-9c45-fc6ed4a7407b
// @Param uuid path string true "Vehicle UUID"
// @Router /vehicle/{uuid} [delete]
func DeleteVehicleByID(c *gin.Context) {
	var user = c.MustGet("user").(models.User)

	vehicleUUID, err := uuid.FromString(c.Param("uuid"))

	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	var vehicle models.Vehicle
	err = database.DB.Where("id = ?", vehicleUUID).First(&vehicle).Error
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

// @Summary Update vehicle
// @Description Update a vehicle by its UUID.
// @Tags Vehicles
// @Produce json
// @Success 200 {object} models.VehicleBase
// @Failure 403
// @Failure 404
// @Param auth_uuid header string true "User ID"
// @Param.examples auth_uuid user1 summary User 1
// @Param.examples auth_uuid user1 description User has One personal vehicle and one shared vehicle
// @Param.examples auth_uuid user1 value 019785fe-4eb4-766e-9c45-bec7780972a2
// @Param.examples auth_uuid user2 summary User 2
// @Param.examples auth_uuid user2 description User has vehicle shared FROM User 1 with write access
// @Param.examples auth_uuid user2 value 019785fe-4eb4-766e-9c45-c1f83e7c1f1f
// @Param.examples auth_uuid user3 summary User 3
// @Param.examples auth_uuid user3 description User has vehicle shared FROM User 1 with read access
// @Param.examples auth_uuid user3 value 019785fe-4eb4-766e-9c45-c497f2d9fe9e
// @Param.examples auth_uuid user4 summary User 4
// @Param.examples auth_uuid user4 description User has One personal vehicle
// @Param.examples auth_uuid user4 value 019785fe-4eb4-766e-9c45-c8578456b4df
// @Param uuid path string true "Vehicle UUID"
// @Param.examples uuid vehicle2 summary Vehicle 2
// @Param.examples uuid vehicle2 description Vehicle shared by User 1 with User 2 (write access) and User 3
// @Param.examples uuid vehicle2 value 019785fe-4eb4-766e-9c45-d77f41aa8317
// @Param user body models.VehicleBase true "Vehicle object"
// @Param.examples user vehicle_modify summary Modify vehicle
// @Param.examples user vehicle_modify description Set nickname to "Modified Vehicle 2"
// @Param.examples user vehicle_modify value { "nickname": "Modified Vehicle 2" }
// @Param.examples user vehicle_reset summary Reset Vehicle
// @Param.examples user vehicle_reset description Reset vehicle to original state
// @Param.examples user vehicle_reset value { "Nickname": "Vehicle 2" }
// @Router /vehicle/{uuid} [patch]
func UpdateVehicleByID(c *gin.Context) {
	var user = c.MustGet("user").(models.User)

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

	err = database.DB.Where("id = ?", vehicleUUID).First(&vehicle).Error
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
