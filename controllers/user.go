package controllers

import (
	"auto-myself-api/database"
	"auto-myself-api/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Returns user record by ID
func GetUserByID(c *gin.Context) {
	user := c.MustGet("user").(*models.User)

	uuid := c.Param("uuid")

	var requestedUser models.User
	if err := database.DB.First(&requestedUser, "id = ?", uuid).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			database.LogError(err)
			c.Status(http.StatusInternalServerError)
		} else {
			c.Status(http.StatusNotFound)
		}
		return
	}

	if !user.CanRead(requestedUser) {
		c.Status(http.StatusNotFound)
		return
	}

	c.JSON(http.StatusOK, requestedUser.UserBase)
}

// Modify current user's record
func UpdateCurrentUser(c *gin.Context) {
	var user = c.MustGet("user").(*models.User)

	var input models.UserBase
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	if err := database.DB.Model(&user).Updates(input).Error; err != nil {
		database.LogError(err)
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, user.UserBase)
}

// Get current user's record
func GetCurrentUser(c *gin.Context) {
	var user = c.MustGet("user").(*models.User)

	c.JSON(http.StatusOK, user)
}
