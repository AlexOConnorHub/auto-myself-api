package controllers

import (
	"auto-myself-api/database"
	"auto-myself-api/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Get all shares for the current user
func GetAllShares(c *gin.Context) {
	user := c.MustGet("user").(*models.User)

	var shareIds []string
	database.DB.Model(&user).Association("PendingVehicleSharesReceived").Find(&user.PendingVehicleSharesReceived)

	for _, share := range user.PendingVehicleSharesReceived {
		shareIds = append(shareIds, share.ID.String())
	}

	database.DB.Model(&user).Association("PendingVehicleSharesSent").Find(&user.PendingVehicleSharesSent)

	for _, share := range user.PendingVehicleSharesSent {
		shareIds = append(shareIds, share.ID.String())
	}

	c.JSON(http.StatusOK, shareIds)
}

func GetShareByID(c *gin.Context) {
	user := c.MustGet("user").(*models.User)

	var share models.VehicleUserAccessPending
	if err := database.DB.First(&share, "id = ?", c.Param("uuid")).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			c.Status(http.StatusInternalServerError)
		} else {
			c.Status(http.StatusNotFound)
		}
		return
	}

	if share.UserID != user.ID && share.CreatedBy != user.ID {
		c.Status(http.StatusNotFound)
		return
	}

	c.JSON(http.StatusOK, share.VehicleUserAccessBase)
}

// Create a share
func CreateShare(c *gin.Context) {
	var share models.VehicleUserAccessBase
	if err := c.ShouldBindJSON(&share); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	record := models.VehicleUserAccessPending{
		VehicleUserAccessBase: share,
	}

	if err := database.DB.Create(&record).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusCreated)
}

// Accept a share
func AcceptShare(c *gin.Context) {
	user := c.MustGet("user").(*models.User)
	var share models.VehicleUserAccessPending
	if err := c.ShouldBindJSON(&share); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if user.ID != share.UserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You do not have permission to accept this share"})
		return
	}

	access := models.VehicleUserAccess{
		VehicleUserAccessBase: share.VehicleUserAccessBase,
		CreatedBy:             share.UserID,
	}

	if err := database.DB.Create(&access).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := database.DB.Delete(&share).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	database.DB.Model(&access).Association("Vehicle").Find(&access.Vehicle)

	c.Header("Location", "/v1/share/"+access.Vehicle.ID.String())
	c.Status(http.StatusNoContent)
}

// Delete a share
func DeleteShareByID(c *gin.Context) {
	user := c.MustGet("user").(*models.User)

	var share models.VehicleUserAccessPending
	if err := database.DB.First(&share, "id = ?", c.Param("uuid")).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			database.LogError(err)
			c.Status(http.StatusInternalServerError)
		} else {
			c.Status(http.StatusNotFound)
		}
		return
	}

	if user.ID != share.UserID && user.ID != share.CreatedBy {
		c.Status(http.StatusNotFound)
		return
	}

	if err := database.DB.Delete(&share).Error; err != nil {
		database.LogError(err)
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusNoContent)
}
