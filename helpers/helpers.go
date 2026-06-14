package helpers

import (
	"auto-myself-api/app"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/gin-contrib/slog"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	"gorm.io/gorm"
)

type DatabaseMetadata struct {
	gorm.Model
	ID uuid.UUID `json:"ID" gorm:"type:uuid;primaryKey;not null"`
}

var jwt_collection = make(map[string]string)
var ctx = context.Background()

func TestRequestAsUser(r *gin.Engine, method, path string, userId string, body io.Reader) *httptest.ResponseRecorder {
	bearer, exists := jwt_collection[userId]
	if !exists {
		req, _ := http.NewRequest("POST", "/auth/development", strings.NewReader(`{"user_id":"`+userId+`"}`))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			panic("Failed to get JWT for user " + userId)
		}
		var response struct {
			Bearer string `json:"authentication"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			panic("Failed to parse JWT response for user " + userId + ": " + err.Error())
		}
		bearer = response.Bearer
		jwt_collection[userId] = bearer
	}

	headers := map[string]string{
		"Authorization": bearer,
		"Content-Type":  "application/json",
	}

	req, _ := http.NewRequest(method, path, body)
	w := httptest.NewRecorder()

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	r.ServeHTTP(w, req)
	return w
}

func MakeApp() *app.App {
	db := ConnectDB()
	gorm := ConnectGorm(db)

	return &app.App{
		Gorm: gorm,
		DB:   db,
	}
}

func MakeGin() *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(slog.SetLogger())
	r.TrustedPlatform = gin.PlatformGoogleAppEngine
	return r
}
