package controllers

import (
	"auto-myself-api/database"
	"auto-myself-api/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// @Summary Get all vehicles for the current user
// @Description Returns a list of all vehicle's locations associated with the current user.
// @Tags Vehicles
// @Produce json
// @Success 200 {object} []string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 422 {object} map[string]string
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
// @Success 200 {object} []string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 422 {object} map[string]string
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

	vehiceUUID, err := models.ParseUUID(c.Param("uuid"))

	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Invalid UUID format"})
		return
	}

	var vehicle models.Vehicle
	err = database.DB.Where("id = ?", vehiceUUID).First(&vehicle).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			database.LogError(err)
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if !vehicle.CanRead(user) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vehicle not found"})
		return
	}

	c.JSON(http.StatusOK, vehicle.VehicleBase)
}

func CreateVehicle(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{"message": "Create a vehicle"})
}

func DeleteVehicle(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{"message": "Delete a vehicle"})
}

func UpdateVehicle(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Update a vehicle"})
	// uuid := c.Param("uuid")

	// var requestData struct {
	// 	Color string `json:"color"`
	// 	Miles int    `json:"miles"`
	// }

	// if err := c.BindJSON(&requestData); err != nil {
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
	// 	return
	// }

	// c.JSON(http.StatusOK, gin.H{
	// 	"uuid":    uuid,
	// 	"updated": requestData,
	// })
}
