package shares_test

import (
	"auto-myself-api/internal/helpers"
	"auto-myself-api/internal/shares"
	"auto-myself-api/internal/testutil"
	"auto-myself-api/internal/users"
	"auto-myself-api/internal/vehicles"
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/gofrs/uuid/v5"
)

type shareResponse struct {
	UserID      uuid.UUID `json:"user_id"`
	VehicleID   uuid.UUID `json:"vehicle_id"`
	WriteAccess bool      `json:"write_access"`
}

func TestShareVerifyAllExist(t *testing.T) {
	env := testutil.SetupTest(t)

	for _, share := range testutil.Shares {
		for _, userID := range []uuid.UUID{share.UserID, share.CreatedBy} {
			w := testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/share/"+share.ID.String(), userID, nil)
			testutil.RequireStatus(t, w, http.StatusOK)

			var response shareResponse
			testutil.DecodeJSON(t, w, &response)
			if response.UserID != share.UserID || response.VehicleID != share.VehicleID || response.WriteAccess != share.WriteAccess {
				t.Errorf("unexpected share response: %+v", response)
			}
		}
	}
}

func TestShareCreate(t *testing.T) {
	env := testutil.SetupTest(t)

	body := map[string]any{
		"user_id":      testutil.User5ID,
		"vehicle_id":   testutil.Vehicle2ID,
		"write_access": true,
	}

	w := testutil.RequestAsUser(t, env.Router, http.MethodPost, "/v1/share", testutil.User1ID, testutil.JSONBody(t, body))
	testutil.RequireStatus(t, w, http.StatusCreated)

	location := w.Header().Get("Location")
	if location == "" {
		t.Fatal("expected Location header")
	}

	w = testutil.RequestAsUser(t, env.Router, http.MethodGet, location, testutil.User1ID, nil)
	testutil.RequireStatus(t, w, http.StatusOK)

	var response shareResponse
	testutil.DecodeJSON(t, w, &response)
	if response.UserID != testutil.User5ID || response.VehicleID != testutil.Vehicle2ID || !response.WriteAccess {
		t.Errorf("unexpected created share: %+v", response)
	}
}

func TestShareCreateDuplicate(t *testing.T) {
	env := testutil.SetupTest(t)

	// User 4 already has a pending read share for Vehicle 4, created by User 6.
	body := map[string]any{
		"user_id":      testutil.User4ID,
		"vehicle_id":   testutil.Vehicle4ID,
		"write_access": false,
	}

	w := testutil.RequestAsUser(t, env.Router, http.MethodPost, "/v1/share", testutil.User6ID, testutil.JSONBody(t, body))
	testutil.RequireStatus(t, w, http.StatusConflict)
}

func TestShareCreateWithSelf(t *testing.T) {
	env := testutil.SetupTest(t)

	body := map[string]any{
		"user_id":    testutil.User1ID,
		"vehicle_id": testutil.Vehicle1ID,
	}

	w := testutil.RequestAsUser(t, env.Router, http.MethodPost, "/v1/share", testutil.User1ID, testutil.JSONBody(t, body))
	testutil.RequireStatus(t, w, http.StatusUnprocessableEntity)
}

func TestShareCreatePermissions(t *testing.T) {
	env := testutil.SetupTest(t)

	tests := []struct {
		name    string
		user    uuid.UUID
		vehicle uuid.UUID
		want    int
	}{
		{"readable non-owner", testutil.User2ID, testutil.Vehicle2ID, http.StatusForbidden},
		{"no access", testutil.User5ID, testutil.Vehicle1ID, http.StatusNotFound},
		{"pending recipient", testutil.User4ID, testutil.Vehicle4ID, http.StatusNotFound},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body := map[string]any{"user_id": testutil.User3ID, "vehicle_id": test.vehicle}
			w := testutil.RequestAsUser(t, env.Router, http.MethodPost, "/v1/share", test.user, testutil.JSONBody(t, body))
			testutil.RequireStatus(t, w, test.want)
		})
	}
}

func TestShareCreateUnknownRecipient(t *testing.T) {
	env := testutil.SetupTest(t)

	unknown := "019785fe-4eb4-766e-9c45-fffffffffffe"
	body := map[string]any{"user_id": unknown, "vehicle_id": testutil.Vehicle1ID}

	w := testutil.RequestAsUser(t, env.Router, http.MethodPost, "/v1/share", testutil.User1ID, testutil.JSONBody(t, body))
	testutil.RequireStatus(t, w, http.StatusNotFound)
}

func TestShareAccept(t *testing.T) {
	env := testutil.SetupTest(t)
	tx := env.DB.GetGorm()

	for userIndex, accessRow := range testutil.ShareAccessMatrix {
		user := testutil.Users[userIndex]
		for shareIndex, permission := range accessRow {
			share := testutil.Shares[shareIndex]

			if err := tx.SavePoint("share_accept").Error; err != nil {
				t.Fatalf("creating savepoint: %v", err)
			}

			w := testutil.RequestAsUser(t, env.Router, http.MethodPatch, "/v1/share/"+share.ID.String(), user.ID, nil)

			if err := tx.RollbackTo("share_accept").Error; err != nil {
				t.Fatalf("rolling back to savepoint: %v", err)
			}

			switch {
			case permission == testutil.WRITE:
				if w.Code != http.StatusNoContent {
					t.Errorf("%s accepting share %s: expected 204, got %d", user.Username, share.ID, w.Code)
					continue
				}
				wantLocation := "/v1/vehicle/" + share.VehicleID.String()
				if got := w.Header().Get("Location"); got != wantLocation {
					t.Errorf("%s accepting share %s: expected Location %s, got %s", user.Username, share.ID, wantLocation, got)
				}
			case user.ID == share.CreatedBy:
				if w.Code != http.StatusForbidden {
					t.Errorf("creator %s accepting share %s: expected 403, got %d", user.Username, share.ID, w.Code)
				}
			default:
				if w.Code != http.StatusNotFound {
					t.Errorf("%s accepting share %s: expected 404, got %d", user.Username, share.ID, w.Code)
				}
			}
		}
	}
}

func TestShareAcceptGrantsAccess(t *testing.T) {
	env := testutil.SetupTest(t)

	share := testutil.Shares[0] // User 6 -> User 5, write access to Vehicle 4.

	w := testutil.RequestAsUser(t, env.Router, http.MethodPatch, "/v1/share/"+share.ID.String(), share.UserID, nil)
	testutil.RequireStatus(t, w, http.StatusNoContent)

	// The pending share is gone, and its row is physically removed (not soft-deleted).
	w = testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/share/"+share.ID.String(), share.UserID, nil)
	testutil.RequireStatus(t, w, http.StatusNotFound)

	var pendingRows int64
	if err := env.DB.GetGorm().Table("vehicle_user_access_pending").
		Where("id = ?", share.ID).
		Count(&pendingRows).Error; err != nil {
		t.Fatalf("counting pending rows: %v", err)
	}
	if pendingRows != 0 {
		t.Errorf("expected the consumed pending share to be hard-deleted, found %d row(s)", pendingRows)
	}

	// ...and write access to the vehicle is live.
	w = testutil.RequestAsUser(t, env.Router, http.MethodPatch, "/v1/vehicle/"+share.VehicleID.String(), share.UserID,
		testutil.JSONBody(t, map[string]any{"nickname": "Accepted Vehicle", "version": 1}))
	testutil.RequireStatus(t, w, http.StatusOK)

	// Accepting a second time finds nothing.
	w = testutil.RequestAsUser(t, env.Router, http.MethodPatch, "/v1/share/"+share.ID.String(), share.UserID, nil)
	testutil.RequireStatus(t, w, http.StatusNotFound)
}

func TestShareReadAccess(t *testing.T) {
	env := testutil.SetupTest(t)

	for userIndex, accessRow := range testutil.ShareAccessMatrix {
		user := testutil.Users[userIndex]
		for shareIndex, permission := range accessRow {
			share := testutil.Shares[shareIndex]

			w := testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/share/"+share.ID.String(), user.ID, nil)

			if permission == testutil.NO_ACCESS {
				if w.Code != http.StatusNotFound {
					t.Errorf("%s reading share %s: expected 404, got %d", user.Username, share.ID, w.Code)
				}
				continue
			}
			if w.Code != http.StatusOK {
				t.Errorf("%s reading share %s: expected 200, got %d", user.Username, share.ID, w.Code)
			}
		}
	}
}

func TestShareDelete(t *testing.T) {
	env := testutil.SetupTest(t)
	tx := env.DB.GetGorm()

	for userIndex, accessRow := range testutil.ShareAccessMatrix {
		user := testutil.Users[userIndex]
		for shareIndex, permission := range accessRow {
			share := testutil.Shares[shareIndex]

			if err := tx.SavePoint("share_delete").Error; err != nil {
				t.Fatalf("creating savepoint: %v", err)
			}

			w := testutil.RequestAsUser(t, env.Router, http.MethodDelete, "/v1/share/"+share.ID.String(), user.ID, nil)

			if err := tx.RollbackTo("share_delete").Error; err != nil {
				t.Fatalf("rolling back to savepoint: %v", err)
			}

			if permission >= testutil.READ_ONLY {
				if w.Code != http.StatusNoContent {
					t.Errorf("%s deleting share %s: expected 204, got %d", user.Username, share.ID, w.Code)
				}
			} else if w.Code != http.StatusNotFound {
				t.Errorf("%s deleting share %s: expected 404, got %d", user.Username, share.ID, w.Code)
			}
		}
	}
}

func TestShareList(t *testing.T) {
	env := testutil.SetupTest(t)

	for userIndex, accessRow := range testutil.ShareAccessMatrix {
		user := testutil.Users[userIndex]

		expected := 0
		for _, permission := range accessRow {
			if permission > testutil.NO_ACCESS {
				expected++
			}
		}

		w := testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/share", user.ID, nil)
		testutil.RequireStatus(t, w, http.StatusOK)

		var ids []uuid.UUID
		testutil.DecodeJSON(t, w, &ids)
		if len(ids) != expected {
			t.Errorf("%s: expected %d shares, got %d", user.Username, expected, len(ids))
		}
	}
}

// TestShareAcceptRollback proves the accept transaction rolls back the granted access when the
// pending row cannot be deleted.
func TestShareAcceptRollback(t *testing.T) {
	env := testutil.SetupTest(t)
	repository := shares.NewShareRepository(env.DB)

	missing := &shares.VehicleUserAccessPending{
		ID: testutil.UUID("019d5352-0a33-78e5-9148-59bac3670aff"),
		VehicleUserAccessBase: shares.VehicleUserAccessBase{
			UserID:    testutil.User3ID,
			VehicleID: testutil.Vehicle1ID,
		},
	}

	access, err := repository.Accept(context.Background(), missing)
	if !errors.Is(err, helpers.ErrRowCountOff) {
		t.Fatalf("expected ErrRowCountOff, got %v", err)
	}
	if access != nil {
		t.Errorf("expected no access row on a rolled-back accept, got %+v", access)
	}

	var count int64
	if err := env.DB.GetGorm().Table("vehicle_user_access").
		Where("vehicle_id = ? AND user_id = ? AND deleted_at IS NULL", testutil.Vehicle1ID, testutil.User3ID).
		Count(&count).Error; err != nil {
		t.Fatalf("counting access rows: %v", err)
	}
	if count != 0 {
		t.Errorf("expected the granted access row to be rolled back, found %d", count)
	}
}

func TestShareCreateRequiresIDs(t *testing.T) {
	env := testutil.SetupTest(t)

	tests := []struct {
		name string
		body map[string]any
	}{
		{"missing vehicle_id", map[string]any{"user_id": testutil.User5ID}},
		{"missing user_id", map[string]any{"vehicle_id": testutil.Vehicle2ID}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w := testutil.RequestAsUser(t, env.Router, http.MethodPost, "/v1/share", testutil.User1ID, testutil.JSONBody(t, test.body))
			testutil.RequireStatus(t, w, http.StatusUnprocessableEntity)
		})
	}
}

func TestAcceptShareReturnsTheNewAccess(t *testing.T) {
	env := testutil.SetupTest(t)

	shareRepository := shares.NewShareRepository(env.DB)
	vehicleRepository := vehicles.NewVehicleRepository(env.DB)
	userRepository := users.NewUserRepository(env.DB)
	service := shares.NewShareService(shareRepository, vehicleRepository, userRepository)

	recipient, err := userRepository.FindByID(context.Background(), testutil.User5ID)
	if err != nil {
		t.Fatalf("loading recipient: %v", err)
	}
	ctx := context.WithValue(context.Background(), helpers.UserCtxKey, recipient)

	share := testutil.Shares[0]
	access, err := service.AcceptShare(ctx, share.ID)
	if err != nil {
		t.Fatalf("accepting share: %v", err)
	}
	if access == nil {
		t.Fatal("expected the new access row")
	}
	if access.ID.IsNil() {
		t.Error("expected the access row ID to be assigned")
	}
	if access.UserID != share.UserID || access.VehicleID != share.VehicleID || access.WriteAccess != share.WriteAccess {
		t.Errorf("unexpected access row: %+v", access)
	}
	if access.CreatedBy != share.UserID {
		t.Errorf("expected created_by %s, got %s", share.UserID, access.CreatedBy)
	}
}
