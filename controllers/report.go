package controllers

import (
	"auto-myself-api/app"
	"auto-myself-api/models"
	"net/http"

	"github.com/gin-contrib/slog"
	"github.com/gin-gonic/gin"
)

func CreateReport(c *gin.Context, a *app.App) {
	user := c.MustGet("user").(*models.User)

	reportInput := models.ReportBase{}
	if err := c.ShouldBindJSON(&reportInput); err != nil {
		slog.Get(c).Warn("Failed to bind JSON", "error", err)
		c.AbortWithStatus(http.StatusUnprocessableEntity)
		return
	}

	reportInput.CreatedBy = user.ID

	report := models.Report{
		ReportBase: reportInput,
	}

	if err := a.Gorm.Create(&report).Error; err != nil {
		slog.Get(c).Error("Failed to create report", "error", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusCreated, report)

}
