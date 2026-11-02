package server_test

import (
	"auto-myself-api/internal/testutil"
	"net/http"
	"testing"
)

func TestProbeEndpoints(t *testing.T) {
	env := testutil.SetupTest(t)

	for _, path := range []string{"/healthz", "/readyz"} {
		t.Run(path, func(t *testing.T) {
			recorder := testutil.RequestWithBearer(t, env.Router, http.MethodGet, path, "", nil)
			testutil.RequireStatus(t, recorder, http.StatusOK)

			var body struct {
				Status string `json:"status"`
			}
			testutil.DecodeJSON(t, recorder, &body)
			if body.Status != "ok" {
				t.Errorf("expected status ok from %s, got %q", path, body.Status)
			}
		})
	}
}
