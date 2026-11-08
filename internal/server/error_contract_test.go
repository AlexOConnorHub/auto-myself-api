package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"auto-myself-api/internal/server"
	"auto-myself-api/internal/testutil"

	"github.com/gin-gonic/gin"
)

type errorBody struct {
	Error   string `json:"error"`
	Version int64  `json:"version"`
}

func requireErrorBody(t *testing.T, recorder *httptest.ResponseRecorder, status int, message string) errorBody {
	t.Helper()

	testutil.RequireStatus(t, recorder, status)
	if contentType := recorder.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
		t.Errorf("expected JSON content type, got %q", contentType)
	}

	var body errorBody
	testutil.DecodeJSON(t, recorder, &body)
	if body.Error != message {
		t.Errorf("error = %q, want %q; body: %s", body.Error, message, recorder.Body.String())
	}
	return body
}

func TestErrorContractBodies(t *testing.T) {
	env := testutil.SetupTest(t)

	t.Run("400 malformed uuid", func(t *testing.T) {
		w := testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/vehicle/not-a-uuid", testutil.User1ID, nil)
		requireErrorBody(t, w, http.StatusBadRequest, "Malformed UUID")
	})

	t.Run("401 unauthenticated", func(t *testing.T) {
		w := testutil.RequestWithBearer(t, env.Router, http.MethodGet, "/v1/vehicle", "", nil)
		requireErrorBody(t, w, http.StatusUnauthorized, "Invalid request")
	})

	t.Run("403 readable but forbidden", func(t *testing.T) {
		w := testutil.RequestAsUser(t, env.Router, http.MethodPatch, "/v1/vehicle/"+testutil.Vehicle2ID.String(), testutil.User3ID,
			testutil.JSONBody(t, map[string]any{"nickname": "nope", "version": 1}))
		requireErrorBody(t, w, http.StatusForbidden, "Access denied")
	})

	t.Run("404 absent", func(t *testing.T) {
		missing := testutil.UUID("019d437c-5f48-753b-8240-5263972f15f0")
		w := testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/vehicle/"+missing.String(), testutil.User1ID, nil)
		requireErrorBody(t, w, http.StatusNotFound, "Vehicle not found")
	})

	t.Run("409 version conflict", func(t *testing.T) {
		w := testutil.RequestAsUser(t, env.Router, http.MethodPatch, "/v1/vehicle/"+testutil.Vehicle1ID.String(), testutil.User1ID,
			testutil.JSONBody(t, map[string]any{"nickname": "stale", "version": 2}))
		body := requireErrorBody(t, w, http.StatusConflict, "Version conflict")
		if body.Version != 1 {
			t.Errorf("version = %d, want the current version 1", body.Version)
		}
	})

	t.Run("422 malformed body", func(t *testing.T) {
		w := testutil.RequestAsUser(t, env.Router, http.MethodPost, "/v1/vehicle", testutil.User1ID,
			strings.NewReader(`{"year":"not-a-number"}`))
		requireErrorBody(t, w, http.StatusUnprocessableEntity, "Invalid request")
	})

	t.Run("500 unexpected failure", func(t *testing.T) {
		env.Storage.FailUpload = true
		t.Cleanup(func() { env.Storage.FailUpload = false })

		w := testutil.RequestAsUser(t, env.Router, http.MethodPost,
			"/v1/file/maintenance/"+testutil.Maintenances[0].ID.String(), testutil.User1ID, strings.NewReader("data"))
		requireErrorBody(t, w, http.StatusInternalServerError, "Internal server error")
	})
}

func TestPanicRecoveryReturnsGenericJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := server.NewEngine()
	router.GET("/panic", func(*gin.Context) { panic("boom") })

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/panic", nil))

	testutil.RequireStatus(t, w, http.StatusInternalServerError)
	if contentType := w.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
		t.Errorf("expected JSON content type, got %q", contentType)
	}
	if body := strings.TrimSpace(w.Body.String()); body != `{"error":"Internal server error"}` {
		t.Errorf("panic body = %s, want the generic JSON 500", body)
	}
	if strings.Contains(w.Body.String(), "boom") {
		t.Error("panic detail leaked into the response")
	}
}
