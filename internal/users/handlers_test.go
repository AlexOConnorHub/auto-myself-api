package users_test

import (
	"auto-myself-api/internal/testutil"
	"net/http"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
)

type userResponse struct {
	ID       uuid.UUID `json:"ID"`
	Username string    `json:"username"`
	Version  int64     `json:"version"`
}

func TestUsersVerifyAllExist(t *testing.T) {
	env := testutil.SetupTest(t)

	for _, user := range testutil.Users {
		w := testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/user", user.ID, nil)
		testutil.RequireStatus(t, w, http.StatusOK)

		var current userResponse
		testutil.DecodeJSON(t, w, &current)
		if current.Username != user.Username {
			t.Errorf("current user: expected username %q, got %q", user.Username, current.Username)
		}
		if current.ID != user.ID {
			t.Errorf("current user: expected ID %s, got %s", user.ID, current.ID)
		}

		w = testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/user/"+user.ID.String(), user.ID, nil)
		testutil.RequireStatus(t, w, http.StatusOK)

		var byID userResponse
		testutil.DecodeJSON(t, w, &byID)
		if byID.Username != user.Username {
			t.Errorf("by-ID user: expected username %q, got %q", user.Username, byID.Username)
		}
		if byID.ID != user.ID {
			t.Errorf("by-ID user: expected ID %s, got %s", user.ID, byID.ID)
		}
	}
}

func TestUserPatch(t *testing.T) {
	env := testutil.SetupTest(t)

	user := testutil.Users[0]
	newName := "MODIFIED " + user.Username

	w := testutil.RequestAsUser(t, env.Router, http.MethodPatch, "/v1/user", user.ID,
		testutil.JSONBody(t, map[string]any{"username": newName, "version": 1}))
	testutil.RequireStatus(t, w, http.StatusOK)

	var response userResponse
	testutil.DecodeJSON(t, w, &response)
	if response.Username != newName {
		t.Errorf("expected updated username %q, got %q", newName, response.Username)
	}
	if response.Version != 2 {
		t.Errorf("expected version 2 after the first update, got %d", response.Version)
	}

	w = testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/user", user.ID, nil)
	testutil.RequireStatus(t, w, http.StatusOK)
	testutil.DecodeJSON(t, w, &response)
	if response.Username != newName {
		t.Errorf("expected persisted username %q, got %q", newName, response.Username)
	}
	if response.Version != 2 {
		t.Errorf("expected persisted version 2, got %d", response.Version)
	}
}

func TestUserPatchVersionConflict(t *testing.T) {
	env := testutil.SetupTest(t)

	user := testutil.Users[0]

	w := testutil.RequestAsUser(t, env.Router, http.MethodPatch, "/v1/user", user.ID,
		testutil.JSONBody(t, map[string]any{"username": "first", "version": 1}))
	testutil.RequireStatus(t, w, http.StatusOK)

	var first userResponse
	testutil.DecodeJSON(t, w, &first)
	if first.Version != 2 {
		t.Fatalf("expected version 2 after the first update, got %d", first.Version)
	}

	w = testutil.RequestAsUser(t, env.Router, http.MethodPatch, "/v1/user", user.ID,
		testutil.JSONBody(t, map[string]any{"username": "second", "version": 1}))
	testutil.RequireStatus(t, w, http.StatusConflict)

	var conflict struct {
		Error   string `json:"error"`
		Version int64  `json:"version"`
	}
	testutil.DecodeJSON(t, w, &conflict)
	if conflict.Error != "Version conflict" || conflict.Version != 2 {
		t.Errorf("expected a version conflict at version 2, got %+v", conflict)
	}

	w = testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/user", user.ID, nil)
	testutil.RequireStatus(t, w, http.StatusOK)

	var stored userResponse
	testutil.DecodeJSON(t, w, &stored)
	if stored.Username != "first" || stored.Version != 2 {
		t.Errorf("the stale update must not be applied: %+v", stored)
	}
}

func TestUserPatchRequiresVersion(t *testing.T) {
	env := testutil.SetupTest(t)

	bodies := map[string]map[string]any{
		"missing":  {"username": "missing version"},
		"zero":     {"username": "zero version", "version": 0},
		"negative": {"username": "negative version", "version": -1},
	}

	for name, body := range bodies {
		t.Run(name, func(t *testing.T) {
			w := testutil.RequestAsUser(t, env.Router, http.MethodPatch, "/v1/user", testutil.User5ID, testutil.JSONBody(t, body))
			testutil.RequireStatus(t, w, http.StatusUnprocessableEntity)
		})
	}
}

func TestUserPatchIgnoresProtectedFields(t *testing.T) {
	env := testutil.SetupTest(t)

	w := testutil.RequestAsUser(t, env.Router, http.MethodPatch, "/v1/user", testutil.User5ID,
		testutil.JSONBody(t, map[string]any{
			"username": "renamed",
			"version":  1,
			"ID":       testutil.User1ID,
			"role":     "admin",
		}))
	testutil.RequireStatus(t, w, http.StatusOK)

	var response userResponse
	testutil.DecodeJSON(t, w, &response)
	if response.ID != testutil.User5ID {
		t.Errorf("PATCH must not change the user ID: got %s, want %s", response.ID, testutil.User5ID)
	}

	var role string
	if err := env.DB.GetGorm().Table("users").Select("role").
		Where("id = ?", testutil.User5ID).Row().Scan(&role); err != nil {
		t.Fatalf("reading role: %v", err)
	}
	if role != "user" {
		t.Errorf("PATCH must not change the role: got %q, want %q", role, "user")
	}
}

func TestUserPatchRejectsBadBody(t *testing.T) {
	env := testutil.SetupTest(t)

	w := testutil.RequestAsUser(t, env.Router, http.MethodPatch, "/v1/user", testutil.User1ID,
		testutil.JSONBody(t, map[string]any{"username": 12}))
	testutil.RequireStatus(t, w, http.StatusUnprocessableEntity)
}

func TestUserReadPermissions(t *testing.T) {
	env := testutil.SetupTest(t)

	for authIndex, accessRow := range testutil.UserAccessMatrix {
		authUser := testutil.Users[authIndex]
		for readIndex, permission := range accessRow {
			readUser := testutil.Users[readIndex]

			w := testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/user/"+readUser.ID.String(), authUser.ID, nil)

			switch {
			case permission == testutil.NO_ACCESS:
				if w.Code != http.StatusNotFound {
					t.Errorf("%s reading %s: expected 404, got %d", authUser.Username, readUser.Username, w.Code)
				}
			default:
				if w.Code != http.StatusOK {
					t.Errorf("%s reading %s: expected 200, got %d", authUser.Username, readUser.Username, w.Code)
					continue
				}
				var response userResponse
				testutil.DecodeJSON(t, w, &response)
				if response.Username != readUser.Username {
					t.Errorf("%s reading %s: expected username %q, got %q", authUser.Username, readUser.Username, readUser.Username, response.Username)
				}
			}
		}
	}
}

func TestUserGetUnknownID(t *testing.T) {
	env := testutil.SetupTest(t)

	// A well-formed v7 UUID that is not seeded.
	unknown := "019785fe-4eb4-766e-9c45-ffffffffffff"
	w := testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/user/"+unknown, testutil.User1ID, nil)
	testutil.RequireStatus(t, w, http.StatusNotFound)
}

func TestUserGetMalformedID(t *testing.T) {
	env := testutil.SetupTest(t)

	w := testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/user/not-a-uuid", testutil.User1ID, nil)
	testutil.RequireStatus(t, w, http.StatusBadRequest)
}

func TestUserVisibilityEndsWhenShareIsDeleted(t *testing.T) {
	env := testutil.SetupTest(t)

	// User 2 can read User 7 because User 7 has write access to Vehicle 2, which User 2 can read.
	w := testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/user/"+testutil.User7ID.String(), testutil.User2ID, nil)
	testutil.RequireStatus(t, w, http.StatusOK)

	// Soft-delete User 7's share of Vehicle 2.
	if err := env.DB.GetGorm().Table("vehicle_user_access").
		Where("user_id = ? AND vehicle_id = ?", testutil.User7ID, testutil.Vehicle2ID).
		Update("deleted_at", time.Now()).Error; err != nil {
		t.Fatalf("soft-deleting access: %v", err)
	}

	// The vehicle they shared is no longer shared, so User 7 is hidden.
	w = testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/user/"+testutil.User7ID.String(), testutil.User2ID, nil)
	testutil.RequireStatus(t, w, http.StatusNotFound)
}

func TestUserVisibilityEndsWhenVehicleIsDeleted(t *testing.T) {
	env := testutil.SetupTest(t)

	// User 1 (owner) and User 2 share Vehicle 2.
	w := testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/user/"+testutil.User2ID.String(), testutil.User1ID, nil)
	testutil.RequireStatus(t, w, http.StatusOK)

	if err := env.DB.GetGorm().Table("vehicles").
		Where("id = ?", testutil.Vehicle2ID).
		Update("deleted_at", time.Now()).Error; err != nil {
		t.Fatalf("soft-deleting vehicle: %v", err)
	}

	w = testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/user/"+testutil.User2ID.String(), testutil.User1ID, nil)
	testutil.RequireStatus(t, w, http.StatusNotFound)
}
