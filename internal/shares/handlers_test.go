package shares_test

import (
	"auto-myself-api/internal/helpers"
	"auto-myself-api/internal/shares"
	"auto-myself-api/internal/testutil"
	"auto-myself-api/internal/users"
	"auto-myself-api/internal/vehicles"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
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
		CreatedBy: testutil.User1ID,
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
	if access.CreatedBy != share.CreatedBy {
		t.Errorf("expected created_by %s (the inviter), got %s", share.CreatedBy, access.CreatedBy)
	}
	if access.CreatedBy == share.UserID {
		t.Error("accepted access must not record the recipient as created_by")
	}
}

type accessResponse struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	WriteAccess bool      `json:"write_access"`
	Version     int64     `json:"version"`
	CreatedAt   string    `json:"created_at"`
}

func TestAccessList(t *testing.T) {
	env := testutil.SetupTest(t)

	for userIndex, accessRow := range testutil.VehicleAccessMatrix {
		user := testutil.Users[userIndex]
		for vehicleIndex, permission := range accessRow {
			vehicle := testutil.Vehicles[vehicleIndex]
			path := "/v1/vehicle/" + vehicle.ID.String() + "/access"

			w := testutil.RequestAsUser(t, env.Router, http.MethodGet, path, user.ID, nil)
			raw := w.Body.String()

			if vehicle.OwnerID == user.ID {
				testutil.RequireStatus(t, w, http.StatusOK)
				entries := decodeAccessEntries(t, raw)
				expected := expectedAccessForVehicle(vehicle.ID)
				if len(entries) != len(expected) {
					t.Errorf("%s listing %s: expected %d rows, got %d", user.Username, vehicle.ID, len(expected), len(entries))
					continue
				}
				for i := range expected {
					if entries[i].ID != expected[i].ID || entries[i].UserID != expected[i].UserID || entries[i].WriteAccess != expected[i].WriteAccess {
						t.Errorf("%s listing %s row %d: expected %+v, got %+v", user.Username, vehicle.ID, i, expected[i], entries[i])
					}
				}
				assertAccessDTO(t, raw, entries)
				if len(expected) == 0 && strings.TrimSpace(raw) != "[]" {
					t.Errorf("%s listing empty %s: expected [], got %s", user.Username, vehicle.ID, raw)
				}
				continue
			}

			switch permission {
			case testutil.WRITE, testutil.READ_ONLY:
				testutil.RequireStatus(t, w, http.StatusOK)
				entries := decodeAccessEntries(t, raw)
				if len(entries) != 1 {
					t.Errorf("%s listing %s: expected exactly one own row, got %d", user.Username, vehicle.ID, len(entries))
					continue
				}
				wantWrite := permission == testutil.WRITE
				if entries[0].UserID != user.ID || entries[0].WriteAccess != wantWrite {
					t.Errorf("%s listing %s: expected own row (write=%v), got %+v", user.Username, vehicle.ID, wantWrite, entries[0])
				}
				assertAccessDTO(t, raw, entries)
			default:
				testutil.RequireStatus(t, w, http.StatusNotFound)
				var body map[string]string
				testutil.DecodeJSON(t, w, &body)
				if body["error"] != "Vehicle not found" {
					t.Errorf("%s listing %s: unexpected error body %v", user.Username, vehicle.ID, body)
				}
			}
		}
	}
}

func TestAccessRevokePermissions(t *testing.T) {
	env := testutil.SetupTest(t)
	tx := env.DB.GetGorm()

	for _, access := range testutil.VehicleAccesses {
		ownerID := ownerOf(t, access.VehicleID)
		for _, user := range testutil.Users {
			t.Run(fmt.Sprintf("%s/%s", user.Username, access.ID), func(t *testing.T) {
				if err := tx.SavePoint("revoke_access").Error; err != nil {
					t.Fatalf("creating savepoint: %v", err)
				}

				w := testutil.RequestAsUser(t, env.Router, http.MethodDelete, "/v1/access/"+access.ID.String(), user.ID, nil)

				var active int64
				if err := tx.Table("vehicle_user_access").
					Where("id = ? AND deleted_at IS NULL", access.ID).
					Count(&active).Error; err != nil {
					t.Fatalf("counting active access rows: %v", err)
				}

				if user.ID == access.UserID || user.ID == ownerID {
					if w.Code != http.StatusNoContent {
						t.Errorf("expected 204 revoking own/owned access, got %d (%s)", w.Code, w.Body.String())
					}
					if active != 0 {
						t.Errorf("expected the row to be revoked, found %d active", active)
					}
					var total int64
					if err := tx.Table("vehicle_user_access").Where("id = ?", access.ID).Count(&total).Error; err != nil {
						t.Fatalf("counting access rows: %v", err)
					}
					if total != 1 {
						t.Errorf("expected a soft-deleted row to remain, found %d total", total)
					}
				} else {
					if w.Code != http.StatusNotFound {
						t.Errorf("expected 404 revoking another user's access, got %d (%s)", w.Code, w.Body.String())
					}
					if active != 1 {
						t.Errorf("expected the denied revoke to leave the row active, found %d active", active)
					}
				}

				if err := tx.RollbackTo("revoke_access").Error; err != nil {
					t.Fatalf("rolling back to savepoint: %v", err)
				}
			})
		}
	}
}

func TestAccessRevokeByOwnerRemovesAccess(t *testing.T) {
	env := testutil.SetupTest(t)
	access := accessFixtureFor(t, testutil.Vehicle2ID, testutil.User3ID)

	w := testutil.RequestAsUser(t, env.Router, http.MethodDelete, "/v1/access/"+access.ID.String(), testutil.User1ID, nil)
	testutil.RequireStatus(t, w, http.StatusNoContent)

	var row struct {
		DeletedAt *time.Time `gorm:"column:deleted_at"`
	}
	if err := env.DB.GetGorm().Table("vehicle_user_access").
		Select("deleted_at").Where("id = ?", access.ID).Scan(&row).Error; err != nil {
		t.Fatalf("loading access row: %v", err)
	}
	if row.DeletedAt == nil {
		t.Error("expected the access row to be soft-deleted, not hard-deleted")
	}

	record := testutil.Maintenances[2] // one of Vehicle 2's records
	checks := []struct {
		name   string
		method string
		path   string
		body   map[string]any
	}{
		{"vehicle read", http.MethodGet, "/v1/vehicle/" + testutil.Vehicle2ID.String(), nil},
		{"vehicle write", http.MethodPatch, "/v1/vehicle/" + testutil.Vehicle2ID.String(), map[string]any{"nickname": "Nope", "version": 1}},
		{"vehicle delete", http.MethodDelete, "/v1/vehicle/" + testutil.Vehicle2ID.String(), nil},
		{"maintenance list", http.MethodGet, "/v1/vehicle/" + testutil.Vehicle2ID.String() + "/maintenance", nil},
		{"maintenance read", http.MethodGet, "/v1/maintenance/" + record.ID.String(), nil},
		{"maintenance create", http.MethodPost, "/v1/maintenance", map[string]any{"vehicle_id": testutil.Vehicle2ID}},
		{"file list", http.MethodGet, "/v1/file/maintenance/" + record.ID.String(), nil},
		{"access list", http.MethodGet, "/v1/vehicle/" + testutil.Vehicle2ID.String() + "/access", nil},
	}
	for _, check := range checks {
		var body io.Reader
		if check.body != nil {
			body = testutil.JSONBody(t, check.body)
		}
		w := testutil.RequestAsUser(t, env.Router, check.method, check.path, testutil.User3ID, body)
		if w.Code != http.StatusNotFound {
			t.Errorf("%s after revocation: expected 404, got %d (%s)", check.name, w.Code, w.Body.String())
		}
	}

	w = testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/vehicle", testutil.User3ID, nil)
	testutil.RequireStatus(t, w, http.StatusOK)
	var vehicleIDs []uuid.UUID
	testutil.DecodeJSON(t, w, &vehicleIDs)
	for _, id := range vehicleIDs {
		if id == testutil.Vehicle2ID {
			t.Error("revoked user still lists the vehicle")
		}
	}

	w = testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/vehicle/"+testutil.Vehicle2ID.String()+"/access", testutil.User1ID, nil)
	testutil.RequireStatus(t, w, http.StatusOK)
	var entries []accessResponse
	testutil.DecodeJSON(t, w, &entries)
	if len(entries) != 3 {
		t.Errorf("expected 3 remaining collaborators, got %d", len(entries))
	}
	for _, entry := range entries {
		if entry.UserID == testutil.User3ID {
			t.Error("revoked collaborator still appears in the owner's listing")
		}
	}

	// Cross-user profile visibility derived from the revoked row disappears in both directions.
	w = testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/user/"+testutil.User1ID.String(), testutil.User3ID, nil)
	testutil.RequireStatus(t, w, http.StatusNotFound)
	w = testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/user/"+testutil.User3ID.String(), testutil.User1ID, nil)
	testutil.RequireStatus(t, w, http.StatusNotFound)
}

func TestAccessSelfLeave(t *testing.T) {
	t.Run("read only", func(t *testing.T) {
		env := testutil.SetupTest(t)
		access := accessFixtureFor(t, testutil.Vehicle2ID, testutil.User3ID)

		w := testutil.RequestAsUser(t, env.Router, http.MethodDelete, "/v1/access/"+access.ID.String(), testutil.User3ID, nil)
		testutil.RequireStatus(t, w, http.StatusNoContent)

		w = testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/vehicle/"+testutil.Vehicle2ID.String(), testutil.User3ID, nil)
		testutil.RequireStatus(t, w, http.StatusNotFound)
	})

	t.Run("write", func(t *testing.T) {
		env := testutil.SetupTest(t)
		access := accessFixtureFor(t, testutil.Vehicle2ID, testutil.User2ID)

		w := testutil.RequestAsUser(t, env.Router, http.MethodDelete, "/v1/access/"+access.ID.String(), testutil.User2ID, nil)
		testutil.RequireStatus(t, w, http.StatusNoContent)

		w = testutil.RequestAsUser(t, env.Router, http.MethodPatch, "/v1/vehicle/"+testutil.Vehicle2ID.String(), testutil.User2ID,
			testutil.JSONBody(t, map[string]any{"nickname": "Nope", "version": 1}))
		testutil.RequireStatus(t, w, http.StatusNotFound)
	})
}

func TestAccessRevokeThenReinvite(t *testing.T) {
	env := testutil.SetupTest(t)
	old := accessFixtureFor(t, testutil.Vehicle2ID, testutil.User3ID)

	w := testutil.RequestAsUser(t, env.Router, http.MethodDelete, "/v1/access/"+old.ID.String(), testutil.User1ID, nil)
	testutil.RequireStatus(t, w, http.StatusNoContent)

	w = testutil.RequestAsUser(t, env.Router, http.MethodPost, "/v1/share", testutil.User1ID,
		testutil.JSONBody(t, map[string]any{
			"user_id":      testutil.User3ID,
			"vehicle_id":   testutil.Vehicle2ID,
			"write_access": true,
		}))
	testutil.RequireStatus(t, w, http.StatusCreated)
	location := w.Header().Get("Location")
	if location == "" {
		t.Fatal("expected Location header")
	}

	w = testutil.RequestAsUser(t, env.Router, http.MethodPatch, location, testutil.User3ID, nil)
	testutil.RequireStatus(t, w, http.StatusNoContent)

	w = testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/vehicle/"+testutil.Vehicle2ID.String()+"/access", testutil.User1ID, nil)
	testutil.RequireStatus(t, w, http.StatusOK)
	var entries []accessResponse
	testutil.DecodeJSON(t, w, &entries)
	var found *accessResponse
	for i := range entries {
		if entries[i].UserID == testutil.User3ID {
			found = &entries[i]
		}
	}
	if found == nil {
		t.Fatal("expected the re-granted collaborator in the owner's listing")
	}
	if found.ID == old.ID {
		t.Error("expected a fresh access row after re-invite, got the revoked row ID")
	}
	if !found.WriteAccess {
		t.Error("expected write access from the re-invite")
	}

	var active int64
	if err := env.DB.GetGorm().Table("vehicle_user_access").
		Where("vehicle_id = ? AND user_id = ? AND deleted_at IS NULL", testutil.Vehicle2ID, testutil.User3ID).
		Count(&active).Error; err != nil {
		t.Fatalf("counting active access rows: %v", err)
	}
	if active != 1 {
		t.Errorf("expected exactly one active access row, got %d", active)
	}

	var oldRow struct {
		DeletedAt *time.Time `gorm:"column:deleted_at"`
	}
	if err := env.DB.GetGorm().Table("vehicle_user_access").
		Select("deleted_at").Where("id = ?", old.ID).Scan(&oldRow).Error; err != nil {
		t.Fatalf("loading the revoked row: %v", err)
	}
	if oldRow.DeletedAt == nil {
		t.Error("expected the revoked row to remain soft-deleted")
	}

	w = testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/vehicle/"+testutil.Vehicle2ID.String(), testutil.User3ID, nil)
	testutil.RequireStatus(t, w, http.StatusOK)
}

func TestAccessRevokeNotFoundAndContract(t *testing.T) {
	env := testutil.SetupTest(t)

	missing := testutil.UUID("019d5352-0a33-78e5-9148-59bac3670aff")
	w := testutil.RequestAsUser(t, env.Router, http.MethodDelete, "/v1/access/"+missing.String(), testutil.User1ID, nil)
	testutil.RequireStatus(t, w, http.StatusNotFound)
	var body map[string]string
	testutil.DecodeJSON(t, w, &body)
	if body["error"] != "Access not found" {
		t.Errorf("unexpected missing-access body: %v", body)
	}

	access := accessFixtureFor(t, testutil.Vehicle4ID, testutil.User8ID)
	w = testutil.RequestAsUser(t, env.Router, http.MethodDelete, "/v1/access/"+access.ID.String(), testutil.User6ID, nil)
	testutil.RequireStatus(t, w, http.StatusNoContent)
	w = testutil.RequestAsUser(t, env.Router, http.MethodDelete, "/v1/access/"+access.ID.String(), testutil.User6ID, nil)
	testutil.RequireStatus(t, w, http.StatusNotFound)

	w = testutil.RequestAsUser(t, env.Router, http.MethodDelete, "/v1/access/not-a-uuid", testutil.User1ID, nil)
	testutil.RequireStatus(t, w, http.StatusBadRequest)
	testutil.DecodeJSON(t, w, &body)
	if body["error"] != "Malformed UUID" {
		t.Errorf("unexpected malformed-UUID body: %v", body)
	}

	v4 := testutil.UUID("550e8400-e29b-41d4-a716-446655440000")
	w = testutil.RequestAsUser(t, env.Router, http.MethodDelete, "/v1/access/"+v4.String(), testutil.User1ID, nil)
	testutil.RequireStatus(t, w, http.StatusBadRequest)

	w = testutil.RequestWithBearer(t, env.Router, http.MethodDelete, "/v1/access/"+missing.String(), "", nil)
	testutil.RequireStatus(t, w, http.StatusUnauthorized)
}

// TestAcceptRevokedPendingFails proves the conditional hard delete: a pending invite that was
// soft-deleted between the read and the transaction cannot be consumed.
func TestAcceptRevokedPendingFails(t *testing.T) {
	env := testutil.SetupTest(t)
	repository := shares.NewShareRepository(env.DB)

	pending, err := repository.FindByID(context.Background(), testutil.Share1ID)
	if err != nil {
		t.Fatalf("loading pending share: %v", err)
	}
	if err := repository.DeleteByID(context.Background(), testutil.Share1ID); err != nil {
		t.Fatalf("soft-deleting pending share: %v", err)
	}

	access, err := repository.Accept(context.Background(), pending)
	if !errors.Is(err, helpers.ErrRowCountOff) {
		t.Fatalf("expected ErrRowCountOff, got %v", err)
	}
	if access != nil {
		t.Errorf("expected no access row, got %+v", access)
	}

	var active int64
	if err := env.DB.GetGorm().Table("vehicle_user_access").
		Where("vehicle_id = ? AND user_id = ? AND deleted_at IS NULL", pending.VehicleID, pending.UserID).
		Count(&active).Error; err != nil {
		t.Fatalf("counting access rows: %v", err)
	}
	if active != 0 {
		t.Errorf("expected no granted access, found %d active rows", active)
	}

	var pendingTotal int64
	if err := env.DB.GetGorm().Table("vehicle_user_access_pending").
		Where("id = ?", pending.ID).Count(&pendingTotal).Error; err != nil {
		t.Fatalf("counting pending rows: %v", err)
	}
	if pendingTotal != 1 {
		t.Errorf("expected the soft-deleted pending row to remain, found %d", pendingTotal)
	}
}

func TestAcceptDuplicateActiveAccess(t *testing.T) {
	env := testutil.SetupTest(t)

	duplicate := &shares.VehicleUserAccess{
		VehicleUserAccessBase: shares.VehicleUserAccessBase{
			UserID:      testutil.User5ID,
			VehicleID:   testutil.Vehicle4ID,
			WriteAccess: true,
		},
		CreatedBy: testutil.User6ID,
	}
	if err := env.DB.GetGorm().Create(duplicate).Error; err != nil {
		t.Fatalf("creating the pre-existing access row: %v", err)
	}

	w := testutil.RequestAsUser(t, env.Router, http.MethodPatch, "/v1/share/"+testutil.Share1ID.String(), testutil.User5ID, nil)
	testutil.RequireStatus(t, w, http.StatusConflict)

	var pendingTotal int64
	if err := env.DB.GetGorm().Table("vehicle_user_access_pending").
		Where("id = ?", testutil.Share1ID).Count(&pendingTotal).Error; err != nil {
		t.Fatalf("counting pending rows: %v", err)
	}
	if pendingTotal != 1 {
		t.Errorf("expected the pending row to survive a failed accept, found %d", pendingTotal)
	}
}

func TestAccessUniqueness(t *testing.T) {
	env := testutil.SetupTest(t)
	tx := env.DB.GetGorm()

	t.Run("duplicate active access rejected", func(t *testing.T) {
		if err := tx.SavePoint("dup_access").Error; err != nil {
			t.Fatalf("creating savepoint: %v", err)
		}
		duplicate := &shares.VehicleUserAccess{
			VehicleUserAccessBase: shares.VehicleUserAccessBase{
				UserID:      testutil.User2ID,
				VehicleID:   testutil.Vehicle2ID,
				WriteAccess: true,
			},
			CreatedBy: testutil.User1ID,
		}
		err := tx.Create(duplicate).Error
		if err := tx.RollbackTo("dup_access").Error; err != nil {
			t.Fatalf("rolling back to savepoint: %v", err)
		}
		if !errors.Is(err, gorm.ErrDuplicatedKey) {
			t.Fatalf("expected ErrDuplicatedKey, got %v", err)
		}
	})

	t.Run("soft-deleted access duplicate allowed", func(t *testing.T) {
		duplicate := &shares.VehicleUserAccess{
			VehicleUserAccessBase: shares.VehicleUserAccessBase{
				UserID:      testutil.User2ID,
				VehicleID:   testutil.Vehicle2ID,
				WriteAccess: true,
			},
			CreatedBy: testutil.User1ID,
			DeletedAt: gorm.DeletedAt{Time: time.Now(), Valid: true},
		}
		if err := tx.Create(duplicate).Error; err != nil {
			t.Fatalf("expected a soft-deleted duplicate to be allowed, got %v", err)
		}
	})

	t.Run("duplicate active pending rejected", func(t *testing.T) {
		if err := tx.SavePoint("dup_pending").Error; err != nil {
			t.Fatalf("creating savepoint: %v", err)
		}
		duplicate := &shares.VehicleUserAccessPending{
			VehicleUserAccessBase: shares.VehicleUserAccessBase{
				UserID:      testutil.User4ID,
				VehicleID:   testutil.Vehicle4ID,
				WriteAccess: false,
			},
			CreatedBy: testutil.User6ID,
		}
		err := tx.Create(duplicate).Error
		if err := tx.RollbackTo("dup_pending").Error; err != nil {
			t.Fatalf("rolling back to savepoint: %v", err)
		}
		if !errors.Is(err, gorm.ErrDuplicatedKey) {
			t.Fatalf("expected ErrDuplicatedKey, got %v", err)
		}
	})

	t.Run("soft-deleted pending duplicate allowed", func(t *testing.T) {
		duplicate := &shares.VehicleUserAccessPending{
			VehicleUserAccessBase: shares.VehicleUserAccessBase{
				UserID:      testutil.User4ID,
				VehicleID:   testutil.Vehicle4ID,
				WriteAccess: false,
			},
			CreatedBy: testutil.User6ID,
			DeletedAt: gorm.DeletedAt{Time: time.Now(), Valid: true},
		}
		if err := tx.Create(duplicate).Error; err != nil {
			t.Fatalf("expected a soft-deleted duplicate to be allowed, got %v", err)
		}
	})

	t.Run("indexes exist", func(t *testing.T) {
		var count int64
		if err := tx.Raw(`SELECT count(*) FROM pg_indexes WHERE indexname IN (?, ?)`,
			"vehicle_user_access_active_unique", "vehicle_user_access_pending_active_unique").
			Scan(&count).Error; err != nil {
			t.Fatalf("querying pg_indexes: %v", err)
		}
		if count != 2 {
			t.Errorf("expected both partial unique indexes, found %d", count)
		}
	})
}

func TestShareCreateForAcceptedUser(t *testing.T) {
	env := testutil.SetupTest(t)

	body := map[string]any{
		"user_id":      testutil.User2ID,
		"vehicle_id":   testutil.Vehicle2ID,
		"write_access": true,
	}
	w := testutil.RequestAsUser(t, env.Router, http.MethodPost, "/v1/share", testutil.User1ID, testutil.JSONBody(t, body))
	testutil.RequireStatus(t, w, http.StatusConflict)
}

func TestAccessUpdateByOwner(t *testing.T) {
	env := testutil.SetupTest(t)

	t.Run("promote read only to write", func(t *testing.T) {
		access := accessFixtureFor(t, testutil.Vehicle2ID, testutil.User3ID)
		if access.WriteAccess {
			t.Fatal("fixture precondition: expected User 3 to start read only")
		}

		w := testutil.RequestAsUser(t, env.Router, http.MethodPatch, "/v1/access/"+access.ID.String(), testutil.User1ID,
			testutil.JSONBody(t, map[string]any{"write_access": true, "version": 1}))
		testutil.RequireStatus(t, w, http.StatusOK)

		var response accessResponse
		testutil.DecodeJSON(t, w, &response)
		if response.ID != access.ID || response.UserID != testutil.User3ID {
			t.Errorf("unexpected access response identity: %+v", response)
		}
		if !response.WriteAccess || response.Version != 2 {
			t.Errorf("expected write access at version 2, got %+v", response)
		}
		assertAccessDTO(t, w.Body.String(), []accessResponse{response})

		writeAccess, version := accessRowState(t, env, access.ID)
		if !writeAccess || version != 2 {
			t.Errorf("expected the stored row to gain write access at version 2, got write=%v version=%d", writeAccess, version)
		}
	})

	t.Run("demote write to read only", func(t *testing.T) {
		access := accessFixtureFor(t, testutil.Vehicle2ID, testutil.User2ID)
		if !access.WriteAccess {
			t.Fatal("fixture precondition: expected User 2 to start with write access")
		}

		w := testutil.RequestAsUser(t, env.Router, http.MethodPatch, "/v1/access/"+access.ID.String(), testutil.User1ID,
			testutil.JSONBody(t, map[string]any{"write_access": false, "version": 1}))
		testutil.RequireStatus(t, w, http.StatusOK)

		var response accessResponse
		testutil.DecodeJSON(t, w, &response)
		if response.WriteAccess || response.Version != 2 {
			t.Errorf("expected read only at version 2, got %+v", response)
		}

		// A struct update would skip the zero value and leave write_access true.
		writeAccess, version := accessRowState(t, env, access.ID)
		if writeAccess || version != 2 {
			t.Errorf("expected the stored row to lose write access at version 2, got write=%v version=%d", writeAccess, version)
		}
	})
}

func TestAccessUpdatePermissions(t *testing.T) {
	env := testutil.SetupTest(t)
	tx := env.DB.GetGorm()

	for _, access := range testutil.VehicleAccesses {
		ownerID := ownerOf(t, access.VehicleID)
		for _, user := range testutil.Users {
			t.Run(fmt.Sprintf("%s/%s", user.Username, access.ID), func(t *testing.T) {
				if err := tx.SavePoint("update_access").Error; err != nil {
					t.Fatalf("creating savepoint: %v", err)
				}

				wantWrite := !access.WriteAccess
				w := testutil.RequestAsUser(t, env.Router, http.MethodPatch, "/v1/access/"+access.ID.String(), user.ID,
					testutil.JSONBody(t, map[string]any{
						"write_access": wantWrite,
						"version":      testutil.CurrentVersion(t, tx, "vehicle_user_access", access.ID),
					}))

				writeAccess, version := accessRowState(t, env, access.ID)

				switch {
				case user.ID == ownerID:
					if w.Code != http.StatusOK {
						t.Errorf("expected 200 for the vehicle owner, got %d (%s)", w.Code, w.Body.String())
					}
					if writeAccess != wantWrite || version != 2 {
						t.Errorf("expected the owner's update to persist write=%v at version 2, got write=%v version=%d", wantWrite, writeAccess, version)
					}
				case canReadVehicle(t, user.ID, access.VehicleID):
					if w.Code != http.StatusForbidden {
						t.Errorf("expected 403 for a collaborator, got %d (%s)", w.Code, w.Body.String())
					}
					if writeAccess != access.WriteAccess || version != 1 {
						t.Errorf("denied collaborator update changed the row: write=%v version=%d", writeAccess, version)
					}
				default:
					if w.Code != http.StatusNotFound {
						t.Errorf("expected 404 for a stranger or pending invitee, got %d (%s)", w.Code, w.Body.String())
					}
					if writeAccess != access.WriteAccess || version != 1 {
						t.Errorf("denied stranger update changed the row: write=%v version=%d", writeAccess, version)
					}
				}

				if err := tx.RollbackTo("update_access").Error; err != nil {
					t.Fatalf("rolling back to savepoint: %v", err)
				}
			})
		}
	}
}

func TestAccessUpdateVersionConflict(t *testing.T) {
	env := testutil.SetupTest(t)
	access := accessFixtureFor(t, testutil.Vehicle2ID, testutil.User3ID)
	path := "/v1/access/" + access.ID.String()

	w := testutil.RequestAsUser(t, env.Router, http.MethodPatch, path, testutil.User1ID,
		testutil.JSONBody(t, map[string]any{"write_access": true, "version": 1}))
	testutil.RequireStatus(t, w, http.StatusOK)

	var first accessResponse
	testutil.DecodeJSON(t, w, &first)
	if !first.WriteAccess || first.Version != 2 {
		t.Fatalf("expected version 2 with write access after the first update, got %+v", first)
	}

	w = testutil.RequestAsUser(t, env.Router, http.MethodPatch, path, testutil.User1ID,
		testutil.JSONBody(t, map[string]any{"write_access": false, "version": 1}))
	testutil.RequireStatus(t, w, http.StatusConflict)

	var conflict struct {
		Error   string `json:"error"`
		Version int64  `json:"version"`
	}
	testutil.DecodeJSON(t, w, &conflict)
	if conflict.Error != "Version conflict" || conflict.Version != 2 {
		t.Errorf("expected a version conflict at version 2, got %+v", conflict)
	}

	writeAccess, version := accessRowState(t, env, access.ID)
	if !writeAccess || version != 2 {
		t.Errorf("the stale update must not be applied: write=%v version=%d", writeAccess, version)
	}
}

func TestAccessUpdateValidation(t *testing.T) {
	env := testutil.SetupTest(t)
	access := accessFixtureFor(t, testutil.Vehicle2ID, testutil.User3ID)
	path := "/v1/access/" + access.ID.String()

	cases := []struct {
		name string
		body map[string]any
	}{
		{"missing version", map[string]any{"write_access": true}},
		{"zero version", map[string]any{"write_access": true, "version": 0}},
		{"negative version", map[string]any{"write_access": true, "version": -1}},
		{"missing write_access", map[string]any{"version": 1}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := testutil.RequestAsUser(t, env.Router, http.MethodPatch, path, testutil.User1ID, testutil.JSONBody(t, tc.body))
			testutil.RequireStatus(t, w, http.StatusUnprocessableEntity)
		})
	}

	// A false value must bind and be written: a plain bool update DTO or a struct update would drop it.
	writer := accessFixtureFor(t, testutil.Vehicle2ID, testutil.User2ID)
	w := testutil.RequestAsUser(t, env.Router, http.MethodPatch, "/v1/access/"+writer.ID.String(), testutil.User1ID,
		testutil.JSONBody(t, map[string]any{"write_access": false, "version": 1}))
	testutil.RequireStatus(t, w, http.StatusOK)

	writeAccess, version := accessRowState(t, env, writer.ID)
	if writeAccess || version != 2 {
		t.Errorf("expected write_access false at version 2, got write=%v version=%d", writeAccess, version)
	}
}

func TestAccessUpdateNotFoundAndContract(t *testing.T) {
	env := testutil.SetupTest(t)

	missing := testutil.UUID("019d5352-0a33-78e5-9148-59bac3670aff")
	w := testutil.RequestAsUser(t, env.Router, http.MethodPatch, "/v1/access/"+missing.String(), testutil.User1ID,
		testutil.JSONBody(t, map[string]any{"write_access": true, "version": 1}))
	testutil.RequireStatus(t, w, http.StatusNotFound)
	var body map[string]string
	testutil.DecodeJSON(t, w, &body)
	if body["error"] != "Access not found" {
		t.Errorf("unexpected missing-access body: %v", body)
	}

	// A revoked (soft-deleted) row cannot be updated.
	revoked := testutil.UUID("01978666-50ab-7399-9d85-b3e259e67262")
	w = testutil.RequestAsUser(t, env.Router, http.MethodPatch, "/v1/access/"+revoked.String(), testutil.User1ID,
		testutil.JSONBody(t, map[string]any{"write_access": true, "version": 1}))
	testutil.RequireStatus(t, w, http.StatusNotFound)

	w = testutil.RequestAsUser(t, env.Router, http.MethodPatch, "/v1/access/not-a-uuid", testutil.User1ID,
		testutil.JSONBody(t, map[string]any{"write_access": true, "version": 1}))
	testutil.RequireStatus(t, w, http.StatusBadRequest)
	testutil.DecodeJSON(t, w, &body)
	if body["error"] != "Malformed UUID" {
		t.Errorf("unexpected malformed-UUID body: %v", body)
	}

	v4 := testutil.UUID("550e8400-e29b-41d4-a716-446655440000")
	w = testutil.RequestAsUser(t, env.Router, http.MethodPatch, "/v1/access/"+v4.String(), testutil.User1ID,
		testutil.JSONBody(t, map[string]any{"write_access": true, "version": 1}))
	testutil.RequireStatus(t, w, http.StatusBadRequest)

	w = testutil.RequestWithBearer(t, env.Router, http.MethodPatch, "/v1/access/"+missing.String(), "",
		testutil.JSONBody(t, map[string]any{"write_access": true, "version": 1}))
	testutil.RequireStatus(t, w, http.StatusUnauthorized)
}

func TestAccessUpdateAffectsAuthorization(t *testing.T) {
	env := testutil.SetupTest(t)

	writer := accessFixtureFor(t, testutil.Vehicle2ID, testutil.User2ID)
	w := testutil.RequestAsUser(t, env.Router, http.MethodPatch, "/v1/access/"+writer.ID.String(), testutil.User1ID,
		testutil.JSONBody(t, map[string]any{"write_access": false, "version": 1}))
	testutil.RequireStatus(t, w, http.StatusOK)

	w = testutil.RequestAsUser(t, env.Router, http.MethodPost, "/v1/maintenance", testutil.User2ID,
		testutil.JSONBody(t, map[string]any{"vehicle_id": testutil.Vehicle2ID}))
	testutil.RequireStatus(t, w, http.StatusForbidden)

	w = testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/vehicle/"+testutil.Vehicle2ID.String(), testutil.User2ID, nil)
	testutil.RequireStatus(t, w, http.StatusOK)

	reader := accessFixtureFor(t, testutil.Vehicle2ID, testutil.User3ID)
	w = testutil.RequestAsUser(t, env.Router, http.MethodPatch, "/v1/access/"+reader.ID.String(), testutil.User1ID,
		testutil.JSONBody(t, map[string]any{"write_access": true, "version": 1}))
	testutil.RequireStatus(t, w, http.StatusOK)

	w = testutil.RequestAsUser(t, env.Router, http.MethodPost, "/v1/maintenance", testutil.User3ID,
		testutil.JSONBody(t, map[string]any{"vehicle_id": testutil.Vehicle2ID}))
	testutil.RequireStatus(t, w, http.StatusCreated)
}

func decodeAccessEntries(t *testing.T, raw string) []accessResponse {
	t.Helper()

	var entries []accessResponse
	if err := json.Unmarshal([]byte(raw), &entries); err != nil {
		t.Fatalf("decoding access list %q: %v", raw, err)
	}
	return entries
}

func assertAccessDTO(t *testing.T, raw string, entries []accessResponse) {
	t.Helper()

	for _, forbidden := range []string{`"created_by"`, `"deleted_at"`, `"vehicle_id"`} {
		if strings.Contains(raw, forbidden) {
			t.Errorf("access response leaked %s: %s", forbidden, raw)
		}
	}
	for _, entry := range entries {
		if entry.ID.IsNil() {
			t.Error("expected an access row id")
		}
		if _, err := time.Parse(time.RFC3339, entry.CreatedAt); err != nil {
			t.Errorf("created_at %q is not RFC3339: %v", entry.CreatedAt, err)
		}
	}
}

func expectedAccessForVehicle(vehicleID uuid.UUID) []testutil.VehicleAccessFixture {
	expected := make([]testutil.VehicleAccessFixture, 0)
	for _, access := range testutil.VehicleAccesses {
		if access.VehicleID == vehicleID {
			expected = append(expected, access)
		}
	}
	// Seed rows share one created_at; the API orders by created_at ASC, id ASC.
	sort.Slice(expected, func(i, j int) bool {
		return expected[i].ID.String() < expected[j].ID.String()
	})
	return expected
}

func ownerOf(t *testing.T, vehicleID uuid.UUID) uuid.UUID {
	t.Helper()

	for _, vehicle := range testutil.Vehicles {
		if vehicle.ID == vehicleID {
			return vehicle.OwnerID
		}
	}
	t.Fatalf("vehicle %s not found in fixtures", vehicleID)
	return uuid.Nil
}

func accessFixtureFor(t *testing.T, vehicleID, userID uuid.UUID) testutil.VehicleAccessFixture {
	t.Helper()

	for _, access := range testutil.VehicleAccesses {
		if access.VehicleID == vehicleID && access.UserID == userID {
			return access
		}
	}
	t.Fatalf("access fixture for vehicle %s and user %s not found", vehicleID, userID)
	return testutil.VehicleAccessFixture{}
}

func accessRowState(t *testing.T, env *testutil.Env, id uuid.UUID) (bool, int64) {
	t.Helper()

	var row struct {
		WriteAccess bool  `gorm:"column:write_access"`
		Version     int64 `gorm:"column:version"`
	}
	if err := env.DB.GetGorm().Table("vehicle_user_access").
		Select("write_access", "version").Where("id = ?", id).Scan(&row).Error; err != nil {
		t.Fatalf("loading access row %s: %v", id, err)
	}
	return row.WriteAccess, row.Version
}

func canReadVehicle(t *testing.T, userID, vehicleID uuid.UUID) bool {
	t.Helper()

	for _, access := range testutil.VehicleAccesses {
		if access.VehicleID == vehicleID && access.UserID == userID {
			return true
		}
	}
	return false
}
