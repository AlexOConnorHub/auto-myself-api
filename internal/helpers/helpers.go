package helpers

import (
	"context"
	"encoding/json"
	"errors"
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

var UserCtxKey = "user"
var ErrAccessDenied = errors.New("Access denied")
var ErrRowCountOff = errors.New("Unexpected number of rows")

func DatabaseFetchError(c *gin.Context, err error, args ...any) {
	if err != gorm.ErrRecordNotFound {
		slog.Get(c).Error("Error fetching by ID", append([]any{"error", err}, args...)...)
		c.Status(http.StatusInternalServerError)
	} else {
		slog.Get(c).Info("Record not found for ID", args...)
		c.Status(http.StatusNotFound)
	}
}

type DatabaseMetadata struct {
	gorm.Model
	ID uuid.UUID `json:"ID" gorm:"type:uuid;primaryKey;not null"`
}

var jwt_collection = make(map[string]string)

// var ctx = context.Background()

func TestGetUserJWT(r *gin.Engine, userId string) string {
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
	return bearer
}

func TestRequestAsUser(r *gin.Engine, method, path string, userId string, body io.Reader) *httptest.ResponseRecorder {
	bearer := TestGetUserJWT(r, userId)

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

var (
	NO_ACCESS      = 0
	READ_BY_INVITE = 1
	READ_ONLY      = 2
	WRITE          = 3
)

func SetUUIDV7IfNil(id *uuid.UUID) error {
	if !id.IsNil() {
		return nil
	}

	var err error
	*id, err = uuid.NewV7()
	return err
}

func FindByID[T any](ctx context.Context, db *gorm.DB, id uuid.UUID) (*T, error) {
	item, err := gorm.G[T](db).Where("id = ?", id).First(ctx)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func UpdateByID[T any](ctx context.Context, db *gorm.DB, id uuid.UUID, item *T) error {
	rows, err := gorm.G[T](db).Where("id = ?", id).Updates(ctx, *item)
	if err != nil {
		return err
	}
	if rows != 1 {
		return ErrRowCountOff
	}
	return nil
}

func DeleteByID[T any](ctx context.Context, db *gorm.DB, id uuid.UUID) error {
	rows, err := gorm.G[T](db).Where("id = ?", id).Delete(ctx)
	if err != nil {
		return err
	}
	if rows != 1 {
		return ErrRowCountOff
	}
	return nil
}

func Create[T any](ctx context.Context, db *gorm.DB, item *T) error {
	return gorm.G[T](db).Create(ctx, item)
}
