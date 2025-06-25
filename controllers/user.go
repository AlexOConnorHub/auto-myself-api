package controllers

import (
	"auto-myself-api/database"
	"auto-myself-api/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// @Summary Returns user record by ID
// @Description Useful for finding a user's username
// @Tags Users
// @Produce json
// @Success 200 {object} models.User
// @Failure 404 {object} map[string]string
// @Param uuid path string true "User ID"
// @Param.examples uuid user1 summary User 1
// @Param.examples uuid user1 description User has One personal vehicle and one shared vehicle
// @Param.examples uuid user1 value 019785fe-4eb4-766e-9c45-bec7780972a2
// @Param.examples uuid user2 summary User 2
// @Param.examples uuid user2 description User has vehicle shared FROM User 1 with write access
// @Param.examples uuid user2 value 019785fe-4eb4-766e-9c45-c1f83e7c1f1f
// @Param.examples uuid user3 summary User 3
// @Param.examples uuid user3 description User has vehicle shared FROM User 1 with read access
// @Param.examples uuid user3 value 019785fe-4eb4-766e-9c45-c497f2d9fe9e
// @Param.examples uuid user4 summary User 4
// @Param.examples uuid user4 description User has One personal vehicle
// @Param.examples uuid user4 value 019785fe-4eb4-766e-9c45-c8578456b4df
// @Param.examples uuid user5 summary User 5
// @Param.examples uuid user5 description User has no vehicles, no vehicles shared
// @Param.examples uuid user5 value 019785fe-4eb4-766e-9c45-cec136a9ad6f
// @Param.examples uuid user6 summary User 6
// @Param.examples uuid user6 description User has One vehicle to share
// @Param.examples uuid user6 value 019785fe-4eb4-766e-9c45-f592a1187d0c
// @Param.examples uuid user7 summary User 7
// @Param.examples uuid user7 description User has vehicle shared FROM User 1 and User 6, both with write access
// @Param.examples uuid user7 value 019785fe-4eb4-766e-9c45-f9cd4ee5c0b3
// @Param.examples uuid user8 summary User 8
// @Param.examples uuid user8 description User has One personal vehicle, vehicle shared FROM User 1 (write) and User 6 (read)
// @Param.examples uuid user8 value 019785fe-4eb4-766e-9c45-fc6ed4a7407b
// @Router /user/{uuid} [get]
func GetUserById(c *gin.Context) {
	var user = models.User{}

	parsedUUID, err := models.ParseUUID(c.Param("uuid"))
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Invalid UUID"})
		return
	}

	err = database.DB.Where("id = ?", parsedUUID).First(&user).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user"})
		return
	}

	c.Header("Location", "/user/"+user.DatabaseMetadata.ID.String())
	response := user
	c.JSON(http.StatusOK, response)
}

// @Summary Modify current user's record
// @Description Useful for modifying the user's username
// @Tags Users
// @Produce json
// @Success 200 {object} models.User
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 422 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Param uuid header string true "User ID"
// @Param.examples uuid user1 summary User 1
// @Param.examples uuid user1 description User has One personal vehicle and one shared vehicle
// @Param.examples uuid user1 value 019785fe-4eb4-766e-9c45-bec7780972a2
// @Param user body models.UserBase true "User object"
// @Param.examples user user1_modify summary Modify User 1
// @Param.examples user user1_modify description Set username to "Modified User 1"
// @Param.examples user user1_modify value { "username": "Modified User 1" }
// @Param.examples user user1_reset summary Reset User 1
// @Param.examples user user1_reset description Reset User 1 to original state
// @Param.examples user user1_reset value { "username": "User 1" }
// @Router /user [patch]
func UpdateCurrentUser(c *gin.Context) {
	var user = models.User{}

	currentUser, _ := c.Get("USER_ID")

	err := database.DB.Where("id = ?", currentUser).First(&user).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user"})
		return
	}

	if err = c.ShouldBindJSON(&user.UserBase); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	if err = database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	c.Header("Location", "/user/"+user.DatabaseMetadata.ID.String())
	response := user
	c.JSON(http.StatusOK, response)
}

func GetCurrentUser(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{"message": "Get your user"})
}

func DeleteCurrentUser(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{"message": "Delete your user"})
}
