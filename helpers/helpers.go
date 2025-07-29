package helpers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

type DatabaseMetadata struct {
	gorm.Model
	ID uuid.UUID `json:"ID" gorm:"type:uuid;primaryKey;not null"`
}

func PerformRequest(r *gin.Engine, method, path string, headers map[string]string, body io.Reader) *httptest.ResponseRecorder {
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
