package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetUserById(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Get a user"})
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
