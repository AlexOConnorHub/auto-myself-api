package helpers

import (
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

type DatabaseMetadata struct {
	gorm.Model
	ID uuid.UUID `json:"ID" gorm:"type:uuid;primaryKey;not null"`
}

func PerformRequest(r *gin.Engine, method, path string, headers map[string]string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, nil)
	w := httptest.NewRecorder()

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	r.ServeHTTP(w, req)
	return w
}
