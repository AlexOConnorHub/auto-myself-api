package server_test

import (
	"auto-myself-api/internal/testutil"
	"net/http"
	"testing"

	"github.com/gofrs/uuid/v5"
)

// TestHealth proves the harness boots the real router against the test database and that
// authenticated requests reach a handler.
func TestHealth(t *testing.T) {
	env := testutil.SetupTest(t)

	w := testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/vehicle", testutil.User1ID, nil)
	testutil.RequireStatus(t, w, http.StatusOK)

	var ids []uuid.UUID
	testutil.DecodeJSON(t, w, &ids)
	if len(ids) == 0 {
		t.Error("expected User 1 to see at least one vehicle")
	}
}
