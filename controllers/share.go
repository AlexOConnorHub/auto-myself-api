package controllers

import (
	"auto-myself-api/app"
	"auto-myself-api/database"
	"auto-myself-api/models"
	"net/http"

	"github.com/gin-contrib/slog"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
)

func GetAllShares(c *gin.Context, a *app.App) {
	user := c.MustGet("user").(*models.User)

	var shareIds []string
	a.Gorm.Model(&user).Association("PendingVehicleSharesReceived").Find(&user.PendingVehicleSharesReceived)

	for _, share := range user.PendingVehicleSharesReceived {
		shareIds = append(shareIds, share.ID.String())
	}

	a.Gorm.Model(&user).Association("PendingVehicleSharesSent").Find(&user.PendingVehicleSharesSent)

	for _, share := range user.PendingVehicleSharesSent {
		shareIds = append(shareIds, share.ID.String())
	}

	c.JSON(http.StatusOK, shareIds)
}

func GetShareByID(c *gin.Context, a *app.App) {
	user := c.MustGet("user").(*models.User)

	shareUUID := c.MustGet("uuid_param").(uuid.UUID)

	var share models.VehicleUserAccessPending
	if err := a.Gorm.First(&share, "id = ?", shareUUID).Error; err != nil {
		database.DatabaseFetchError(c, err, "id", shareUUID)
		c.Abort()
		return
	}

	if share.UserID != user.ID && share.CreatedBy != user.ID {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	c.JSON(http.StatusOK, share.VehicleUserAccessBase)
}

func CreateShare(c *gin.Context, a *app.App) {
	var share models.VehicleUserAccessBase
	if err := c.ShouldBindJSON(&share); err != nil {
		slog.Get(c).Warn("Failed to bind JSON", "error", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	record := models.VehicleUserAccessPending{
		VehicleUserAccessBase: share,
	}

	if err := a.Gorm.Create(&record).Error; err != nil {
		slog.Get(c).Warn("Failed to create record", "error", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.Header("Location", "/v1/share/"+record.ID.String())
	c.Status(http.StatusCreated)
}

func AcceptShare(c *gin.Context, a *app.App) {
	user := c.MustGet("user").(*models.User)
	var share models.VehicleUserAccessPending
	if err := c.ShouldBindJSON(&share); err != nil {
		slog.Get(c).Warn("Failed to bind JSON", "error", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	if user.ID != share.UserID {
		var status int
		if user.ID == share.CreatedBy {
			status = http.StatusForbidden
		} else {
			status = http.StatusNotFound
		}
		c.AbortWithStatus(status)
		return
	}

	access := models.VehicleUserAccess{
		VehicleUserAccessBase: share.VehicleUserAccessBase,
		CreatedBy:             share.UserID,
	}

	if err := a.Gorm.Create(&access).Error; err != nil {
		slog.Get(c).Warn("Failed to create record", "error", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	if err := a.Gorm.Delete(&share).Error; err != nil {
		slog.Get(c).Warn("Failed to delete record", "error", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	a.Gorm.Model(&access).Association("Vehicle").Find(&access.Vehicle)

	c.Header("Location", "/v1/vehicle/"+access.Vehicle.ID.String())
	c.Status(http.StatusNoContent)
}

func DeleteShareByID(c *gin.Context, a *app.App) {
	user := c.MustGet("user").(*models.User)

	shareUUID := c.MustGet("uuid_param").(uuid.UUID)

	var share models.VehicleUserAccessPending
	if err := a.Gorm.First(&share, "id = ?", shareUUID).Error; err != nil {
		database.DatabaseFetchError(c, err, "id", shareUUID)
		c.Abort()
		return
	}

	if user.ID != share.UserID && user.ID != share.CreatedBy {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	if err := a.Gorm.Delete(&share).Error; err != nil {
		slog.Get(c).Warn("Failed to delete record", "error", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusNoContent)
}
