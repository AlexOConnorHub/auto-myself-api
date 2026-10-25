package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
)

var testBadUUID = "123e4567-e89b-12d3-a456-426614174000"
var testGoodUUID = "019d50bf-776b-7860-a5b2-354fe9fd7a7d"

func TestParseUUIDMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ParseUUIDMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	router.GET("/test/:uuid", func(c *gin.Context) {
		userUUID, exists := c.Get("uuid_param")
		if !exists {
			c.String(http.StatusOK, "no uuid")
			return
		}
		c.String(http.StatusOK, userUUID.(uuid.UUID).String())
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test/invalid-uuid", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, `{"error":"Malformed UUID"}`, w.Body.String())

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test/"+testBadUUID, nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, `{"error":"UUID must be v7 (with valid timestamp)"}`, w.Body.String())

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test/"+testGoodUUID, nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, testGoodUUID, w.Body.String())

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

}
