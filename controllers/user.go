package controllers

import (
	"auto-myself-api/app"
	"auto-myself-api/database"
	"auto-myself-api/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

func GetUserByID(c *gin.Context, a *app.App) {
	user := c.MustGet("user").(*models.User)

	userUUID := c.MustGet("uuid_param").(uuid.UUID)

	var requestedUser models.User
	if err := a.Gorm.First(&requestedUser, "id = ?", userUUID).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			database.LogError(err)
			c.Status(http.StatusInternalServerError)
		} else {
			c.Status(http.StatusNotFound)
		}
		return
	}

	if !user.CanRead(a, requestedUser) {
		c.Status(http.StatusNotFound)
		return
	}

	c.JSON(http.StatusOK, requestedUser.UserBase)
}

func UpdateCurrentUser(c *gin.Context, a *app.App) {
	var user = c.MustGet("user").(*models.User)

	var input models.UserBase
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	if err := a.Gorm.Model(&user).Updates(input).Error; err != nil {
		database.LogError(err)
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, user.UserBase)
}

func GetCurrentUser(c *gin.Context, a *app.App) {
	var user = c.MustGet("user").(*models.User)

	c.JSON(http.StatusOK, user)
}
