package controllers

import (
	"auto-myself-server/database"
	"auto-myself-server/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetUserById(c *gin.Context) {
	var final = models.User{
		ID: c.Param("uuid"),
	}
	database.DB.Limit(1).Find(&final)

	if final.CreatedAt != "" {
		c.Request.Response.StatusCode = http.StatusNotFound
		return
	}

	response := map[string]interface{}{
		"location":   "/user/" + final.ID,
		"username":   final.Username,
		"created_at": final.CreatedAt,
		"updated_at": final.UpdatedAt,
		"deleted_at": final.DeletedAt,
		"public_key": final.PublicKey,
	}
	c.JSON(http.StatusOK, response)
}

func UpdateCurrentUser(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{"message": "Update your user"})
}

func GetCurrentUser(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{"message": "Get your user"})
}

func DeleteCurrentUser(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{"message": "Delete your user"})
}
