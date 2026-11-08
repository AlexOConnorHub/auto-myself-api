package middleware

import (
	"auto-myself-api/internal/helpers"
	"auto-myself-api/internal/users"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
)

func TestRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RateLimitMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	for i := 0; i < 50; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "127.0.0.1:1234"
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.JSONEq(t, `{"error":"Rate limit exceeded"}`, w.Body.String())

	time.Sleep(2 * time.Second)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUserRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID, err := uuid.FromString("019d50bf-776b-7860-a5b2-354fe9fd7a7d")
	if err != nil {
		t.Fatalf("parsing test UUID: %v", err)
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(helpers.UserCtxKey, &users.User{UserWeb: users.UserWeb{ID: userID}})
	})
	router.Use(UserRateLimitMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	first := httptest.NewRecorder()
	router.ServeHTTP(first, httptest.NewRequest("GET", "/test", nil))
	assert.Equal(t, http.StatusOK, first.Code)

	limited := false
	for i := 0; i < 400; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest("GET", "/test", nil))
		if w.Code == http.StatusTooManyRequests {
			limited = true
			assert.NotEmpty(t, w.Header().Get("Retry-After"))
			assert.JSONEq(t, `{"error":"Rate limit exceeded"}`, w.Body.String())
			break
		}
		assert.Equal(t, http.StatusOK, w.Code)
	}
	assert.True(t, limited, "expected the per-user limiter to reject sustained traffic")
}
