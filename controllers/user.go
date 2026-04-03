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

func GetUserByID(c *gin.Context, a *app.App) {
	user := c.MustGet("user").(*models.User)

	userUUID := c.MustGet("uuid_param").(uuid.UUID)

	var requestedUser models.User
	if err := a.Gorm.First(&requestedUser, "id = ?", userUUID).Error; err != nil {
		database.DatabaseFetchError(c, err, "id", userUUID)
		c.Abort()
		return
	}

	if !user.CanRead(a, requestedUser) {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	c.JSON(http.StatusOK, requestedUser.UserBase)
}

func UpdateCurrentUser(c *gin.Context, a *app.App) {
	var user = c.MustGet("user").(*models.User)

	var input models.UserBase
	if err := c.ShouldBindJSON(&input); err != nil {
		slog.Get(c).Warn("Failed to bind JSON", "error", err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	if err := a.Gorm.Model(&user).Updates(input).Error; err != nil {
		slog.Get(c).Warn("Failed to update user", "error", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, user.UserBase)
}

func GetCurrentUser(c *gin.Context, a *app.App) {
	var user = c.MustGet("user").(*models.User)

	c.JSON(http.StatusOK, user)
}
