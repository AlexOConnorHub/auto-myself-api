package server_test

import (
	"bytes"
	"net/http"
	"strings"
	"testing"

	"auto-myself-api/internal/server"
	"auto-myself-api/internal/testutil"
)

func TestJSONBodyLimitRejectsOversizedBody(t *testing.T) {
	env := testutil.SetupTest(t)

	body := `{"nickname":"` + strings.Repeat("a", server.JSONBodyLimitBytes) + `"}`
	w := testutil.RequestAsUser(t, env.Router, http.MethodPost, "/v1/vehicle", testutil.User1ID, strings.NewReader(body))
	testutil.RequireStatus(t, w, http.StatusRequestEntityTooLarge)

	var response struct {
		Error string `json:"error"`
	}
	testutil.DecodeJSON(t, w, &response)
	if response.Error != "Request body too large" {
		t.Errorf("error = %q, want %q", response.Error, "Request body too large")
	}
}

func TestJSONBodyLimitDoesNotApplyToUploads(t *testing.T) {
	env := testutil.SetupTest(t)

	// 2 MiB is above the JSON limit but below the upload route's 10 MiB raw-body cap.
	data := make([]byte, 2<<20)
	w := testutil.RequestAsUser(t, env.Router, http.MethodPost,
		"/v1/file/maintenance/"+testutil.Maintenances[0].ID.String(), testutil.User1ID, bytes.NewReader(data))
	testutil.RequireStatus(t, w, http.StatusCreated)

	if env.Storage.Len() != 1 {
		t.Errorf("expected 1 stored object, got %d", env.Storage.Len())
	}
}

func TestSecurityHeaders(t *testing.T) {
	env := testutil.SetupTest(t)

	w := testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/vehicle", testutil.User1ID, nil)
	testutil.RequireStatus(t, w, http.StatusOK)

	if got := w.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want %q", got, "nosniff")
	}
}
