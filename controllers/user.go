package controllers

import (
	"auto-myself-server/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetUserById(c *gin.Context) {
	final := models.User{
		ID:         c.Param("uuid"),
		Username:   "testuser",
		CreatedAt:  "2023-10-01T12:00:00Z",
		UpdatedAt:  "2023-10-01T12:00:00Z",
		DeletedAt:  "",
		PublicKey:  "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC3...",
		PrivateKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC3...",
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
