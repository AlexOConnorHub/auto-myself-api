package controllers

import (
	"auto-myself-api/database"
	"auto-myself-api/helpers"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestUserHandler(t *testing.T) {
	database.InitTest(t)

	r := gin.Default()
	SetupRoutes(r)

	t.Run("GetCurrentUser", func(t *testing.T) {
		w := helpers.PerformRequest(r, "GET", "/user", map[string]string{"auth_uuid": "Bearer token"})
		if w.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
		}
	})
}
