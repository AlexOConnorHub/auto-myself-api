package helpers

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

type DatabaseMetadata struct {
	gorm.Model
	ID uuid.UUID `json:"ID" gorm:"type:uuid;primaryKey;not null"`
}

var jwt_collection = make(map[string]string)

func TestRequestAsUser(r *gin.Engine, method, path string, user_id string, body io.Reader) *httptest.ResponseRecorder {
	bearer, exists := jwt_collection[user_id]
	if !exists {
		req, _ := http.NewRequest("POST", "/auth/development", strings.NewReader(`{"user_id":"`+user_id+`"}`))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			panic("Failed to get JWT for user " + user_id)
		}
		var response struct {
			Bearer string `json:"authentication"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			panic("Failed to parse JWT response for user " + user_id + ": " + err.Error())
		}
		bearer = response.Bearer
		jwt_collection[user_id] = bearer
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

func GetRelativeRootPath(tb testing.TB) string {
	if tb != nil {
		tb.Helper()
	}
	importPath := runGoList(tb, "list", "-f", "{{.ImportPath}}")
	modulePath := runGoList(tb, "list", "-m", "-f", "{{.Path}}")
	pkgPath := runGoList(tb, "list", "-f", "{{.Dir}}")

	relativePath, err := filepath.Rel(importPath, modulePath)
	if err != nil {
		panic("failed to get relative path: " + err.Error())
	}
	return filepath.Join(pkgPath, relativePath)
}

func runGoList(tb testing.TB, arg ...string) string {
	if tb != nil {
		tb.Helper()
	}
	cmd := exec.Command("go", arg...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		panic("runGoList: " + err.Error() + "\nOutput: " + string(output))
	}
	return strings.TrimSpace(string(output))
}
