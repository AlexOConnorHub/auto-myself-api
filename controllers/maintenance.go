package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetMaintenance(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Get a maintenance"})
}

func DeleteMaintenance(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Delete a maintenance"})
}

func CreateMaintenance(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{"message": "Create a maintenance"})
}

func UpdateMaintenance(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Update a maintenance"})
	// uuid := c.Param("uuid")

	// var requestData struct {
	// 	Color string `json:"color"`
	// 	Miles int    `json:"miles"`
	// }

	// if err := c.BindJSON(&requestData); err != nil {
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
	// 	return
	// }

	// c.JSON(http.StatusNoContent, gin.H{})
	// // , gin.H{
	// // 	"uuid":    uuid,
	// // 	"updated": requestData,
	// // })
}
