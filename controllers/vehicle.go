package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetAllVehicles(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Get all vehicles"})
}

func GetVehicle(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Get a vehicle"})
}

func CreateVehicle(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{"message": "Create a vehicle"})
}

func DeleteVehicle(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{"message": "Delete a vehicle"})
}

func UpdateVehicle(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Update a vehicle"})
	// uuid := c.Param("uuid")

	// var requestData struct {
	// 	Color string `json:"color"`
	// 	Miles int    `json:"miles"`
	// }

	// if err := c.BindJSON(&requestData); err != nil {
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
	// 	return
	// }

	// c.JSON(http.StatusOK, gin.H{
	// 	"uuid":    uuid,
	// 	"updated": requestData,
	// })
}
