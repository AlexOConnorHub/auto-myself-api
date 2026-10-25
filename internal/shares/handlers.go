package shares

import (
	"auto-myself-api/internal/helpers"
	"auto-myself-api/internal/server"
	"auto-myself-api/internal/users"
	"auto-myself-api/internal/vehicles"
	"net/http"

	"github.com/gin-contrib/slog"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
)

func GetAllShares(c *gin.Context, a *server.App) {
	user := c.MustGet("user").(*users.User)

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

func GetShareByID(c *gin.Context, a *server.App) {
	user := c.MustGet("user").(*users.User)

	shareUUID := c.MustGet("uuid_param").(uuid.UUID)

	var share vehicles.VehicleUserAccessPending
	if err := a.Gorm.First(&share, "id = ?", shareUUID).Error; err != nil {
		helpers.DatabaseFetchError(c, err, "id", shareUUID)
		c.Abort()
		return
	}

	if share.UserID != user.ID && share.CreatedBy != user.ID {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	c.JSON(http.StatusOK, share.VehicleUserAccessBase)
}

func CreateShare(c *gin.Context, a *server.App) {
	user := c.MustGet("user").(*users.User)

	var share vehicles.VehicleUserAccessBase
	if err := c.ShouldBindJSON(&share); err != nil {
		slog.Get(c).Warn("Failed to bind JSON", "error", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	record := vehicles.VehicleUserAccessPending{
		VehicleUserAccessBase: share,
		CreatedBy:             user.ID,
	}

	if err := a.Gorm.Create(&record).Error; err != nil {
		slog.Get(c).Warn("Failed to create record", "error", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.Header("Location", "/v1/share/"+record.ID.String())
	c.Status(http.StatusCreated)
}

func AcceptShare(c *gin.Context, a *server.App) {
	user := c.MustGet("user").(*users.User)
	var share vehicles.VehicleUserAccessPending

	err := a.Gorm.First(&share, "id = ?", c.MustGet("uuid_param").(uuid.UUID)).Error
	if err != nil {
		helpers.DatabaseFetchError(c, err, "id", c.MustGet("uuid_param").(uuid.UUID))
		c.Abort()
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

	access := vehicles.VehicleUserAccess{
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

func DeleteShareByID(c *gin.Context, a *server.App) {
	ctx := c.Request.Context()
	user := c.MustGet("user").(*users.User)

	shareUUID := c.MustGet("uuid_param").(uuid.UUID)

	var share vehicles.VehicleUserAccessPending
	if err := a.Gorm.WithContext(ctx).First(&share, "id = ?", shareUUID).Error; err != nil {
		helpers.DatabaseFetchError(c, err, "id", shareUUID)
		c.Abort()
		return
	}

	if user.ID != share.UserID && user.ID != share.CreatedBy {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	if err := a.Gorm.WithContext(ctx).Delete(&share).Error; err != nil {
		slog.Get(c).Warn("Failed to delete record", "error", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusNoContent)
}
