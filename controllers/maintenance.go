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

// @Summary Get maintenance
// @Description Retrieves all maintenance locations for a vehicle
// @Tags Maintenance
// @Produce json
// @Success 200 {object} []string
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
// @Router /vehicle/{uuid}/maintenance [get]
func GetAllMaintenance(c *gin.Context) {
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

	var maintenanceLocations []string

	database.DB.Model(&vehicle).Association("MaintenanceRecords").Find(&vehicle.MaintenanceRecords)

	for _, MaintenanceRecord := range vehicle.MaintenanceRecords {
		maintenanceLocations = append(maintenanceLocations, MaintenanceRecord.GetLocation())
	}

	c.JSON(http.StatusOK, maintenanceLocations)
}

// @Summary Get maintenance record
// @Description Retrieves a maintenance record by its UUID
// @Tags Maintenance
// @Produce json
// @Success 200 {object} models.MaintenanceRecordBase
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
// @Param uuid path string true "Maintenance Record UUID"
// @Param.examples uuid maintenance1 summary Maintenance 1
// @Param.examples uuid maintenance1 description Vehicle owned by User 1
// @Param.examples uuid maintenance1 value 01978640-1148-74f8-be64-59f2af568e59
// @Param.examples uuid maintenance2 summary Maintenance 2
// @Param.examples uuid maintenance2 description Vehicle shared by User 1 with User 2 and User 3
// @Param.examples uuid maintenance2 value 01978640-1148-74f8-be64-5e6b15475861
// @Param.examples uuid maintenance3 summary Maintenance 3
// @Param.examples uuid maintenance3 description Vehicle owned by User 4
// @Param.examples uuid maintenance3 value 01978640-1148-74f8-be64-600b58c80190
// @Param.examples uuid maintenance4 summary Maintenance 4
// @Param.examples uuid maintenance4 description Vehicle shared by User 6 with User 7
// @Param.examples uuid maintenance4 value 01978640-1148-74f8-be64-673c2bc659d3
// @Param.examples uuid maintenance5 summary Maintenance 5
// @Param.examples uuid maintenance5 description Vehicle owned by User 8
// @Param.examples uuid maintenance5 value 01978640-1148-74f8-be64-6b2a85c627a7
// @Param.examples uuid maintenance6 summary Maintenance 6
// @Param.examples uuid maintenance6 description Vehicle shared by User 1 with User 2 and User 3
// @Param.examples uuid maintenance6 value 01978640-1148-74f8-be64-6ce4acd8abcd
// @Param.examples uuid maintenance7 summary Maintenance 7
// @Param.examples uuid maintenance7 description Vehicle owned by User 4
// @Param.examples uuid maintenance7 value 01978640-1148-74f8-be64-70821e946a20
// @Param.examples uuid maintenance8 summary Maintenance 8
// @Param.examples uuid maintenance8 description Vehicle shared by User 6 with User
// @Param.examples uuid maintenance8 value 01978640-1148-74f8-be64-74b319513577
// @Param.examples uuid maintenance9 summary Maintenance 9
// @Param.examples uuid maintenance9 description Vehicle owned by User 8
// @Param.examples uuid maintenance9 value 01978640-1148-74f8-be64-7b8ece4dc40f
// @Param.examples uuid maintenance10 summary Maintenance 10
// @Param.examples uuid maintenance10 description Vehicle shared by User 1 with User 2 and User 3
// @Param.examples uuid maintenance10 value 01978640-1148-74f8-be64-7ea0e41d68d5
// @Param.examples uuid maintenance11 summary Maintenance 11
// @Param.examples uuid maintenance11 description Vehicle owned by User 4
// @Param.examples uuid maintenance11 value 01978640-1148-74f8-be64-bc7c09adc4c1
// @Param.examples uuid maintenance12 summary Maintenance 12
// @Param.examples uuid maintenance12 description Vehicle shared by User 6 with User 7
// @Param.examples uuid maintenance12 value 01978640-1148-74f8-be64-ac02bbbf11cf
// @Param.examples uuid maintenance13 summary Maintenance 13
// @Param.examples uuid maintenance13 description Vehicle owned by User 8
// @Param.examples uuid maintenance13 value 01978640-1148-74f8-be64-b446e827f938
// @Param.examples uuid maintenance14 summary Maintenance 14
// @Param.examples uuid maintenance14 description Vehicle shared by User 1 with User 2 and User 3
// @Param.examples uuid maintenance14 value 01978640-1148-74f8-be64-ba806fa103c7
// @Param.examples uuid maintenance15 summary Maintenance 15
// @Param.examples uuid maintenance15 description Vehicle owned by User 4
// @Param.examples uuid maintenance15 value 01978640-1149-7118-bada-9f77b4fa870a
// @Param.examples uuid maintenance16 summary Maintenance 16
// @Param.examples uuid maintenance16 description Vehicle shared by User 6 with User 7
// @Param.examples uuid maintenance16 value 01978640-1149-7118-bada-a16e466c1064
// @Param.examples uuid maintenance17 summary Maintenance 17
// @Param.examples uuid maintenance17 description Vehicle owned by User 8
// @Param.examples uuid maintenance17 value 01978640-1149-7118-bada-a7ce46886414
// @Param.examples uuid maintenance18 summary Maintenance 18
// @Param.examples uuid maintenance18 description Vehicle shared by User 1 with User 2 and User 3
// @Param.examples uuid maintenance18 value 01978640-1149-7118-bada-aa596985d112
// @Router /maintenance/{uuid} [get]
func GetMaintenanceByID(c *gin.Context) {
	var user = c.MustGet("user").(models.User)

	maintenanceUUID, err := uuid.FromString(c.Param("uuid"))

	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	var maintenanceRecord models.MaintenanceRecord
	err = database.DB.Where("id = ?", maintenanceUUID).First(&maintenanceRecord).Error
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

// @Summary Create maintenance record TODO: ADD HEADER
// @Description Create a new maintenance record for a vehicle.
// @Tags Maintenance
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
// @Param maintenance_record body models.MaintenanceRecordBase true "New maintenance record"
// @Param.examples maintenance_record maintenance_record1 summary Create a maintenance record
// @Param.examples maintenance_record maintenance_record1 description Create a new maintenance record with notes "A Fresh Vehicle"
// @Param.examples maintenance_record maintenance_record1 value { "notes": "A Fresh Vehicle" }
// @Router /maintenance [post]
func CreateMaintenance(c *gin.Context) {
	var user = c.MustGet("user").(models.User)

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

	c.Header("X-Object-Location", newMaintenanceRecord.GetLocation())
	c.Status(http.StatusCreated)
}

// @Summary Delete maintenance record
// @Description Delete a maintenance record.
// @Tags Maintenance
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
// @Router /maintenance/{uuid} [delete]
func DeleteMaintenanceByID(c *gin.Context) {
	var user = c.MustGet("user").(models.User)

	maintenanceRecordUUID, err := uuid.FromString(c.Param("uuid"))

	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	var maintenanceRecord models.MaintenanceRecord
	err = database.DB.Where("id = ?", maintenanceRecordUUID).First(&maintenanceRecord).Error
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

// @Summary Update maintenance record
// @Description Update a maintenance record by its UUID.
// @Tags Maintenance
// @Produce json
// @Success 200 {object} models.MaintenanceRecordBase
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
// @Param uuid path string true "Maintenance Record UUID"
// @Param.examples uuid maintenance1 summary Maintenance Record
// @Param.examples uuid maintenance1 description Maintenance record for vehicle 2, shared by User 1 with User 2 (write access) and User 3
// @Param.examples uuid maintenance1 value 01978640-1148-74f8-be64-600b58c80190
// @Param maintenance_record body models.MaintenanceRecordBase true "Maintenance record object"
// @Param.examples maintenance_record maintenance_record_modify summary Modify maintenance record
// @Param.examples maintenance_record maintenance_record_modify description Set notes to "NEW DATA"
// @Param.examples maintenance_record maintenance_record_modify value { "notes": "NEW DATA" }
// @Param.examples maintenance_record maintenance_record_reset summary Reset maintenance record
// @Param.examples maintenance_record maintenance_record_reset description Reset maintenance record to original state
// @Param.examples maintenance_record maintenance_record_reset value { "notes": "Brake inspection" }
// @Router /maintenance/{uuid} [patch]
func UpdateMaintenanceByID(c *gin.Context) {
	var user = c.MustGet("user").(models.User)

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

	err = database.DB.Where("id = ?", maintenanceUUID).First(&maintenanceRecord).Error
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
