package controllers

import (
	"auto-myself-api/database"
	"auto-myself-api/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

func CreateShare(c *gin.Context) {
	var share models.VehicleUserAccessBase
	if err := c.ShouldBindJSON(&share); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	record := models.VehicleUserAccess{
		VehicleUserAccessBase: share,
	}

	record.Pending = true

	if err := database.DB.Create(&record).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func AcceptShare(c *gin.Context) {
	var share models.Share
	if err := c.ShouldBindJSON(&share); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user := c.MustGet("user").(models.User)
	if !user.CanWrite(share.VehicleID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You do not have permission to accept this share"})
		return
	}

	if err := database.DB.Model(&share).Update("accepted", true).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, share)
}
