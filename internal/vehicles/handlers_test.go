package vehicles_test

import (
	"auto-myself-api/internal/testutil"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/gofrs/uuid/v5"
)

type vehicleResponse struct {
	ID       uuid.UUID `json:"id"`
	Nickname string    `json:"nickname"`
	Make     string    `json:"make"`
	MakeID   int       `json:"make_id"`
	Model    string    `json:"model"`
	ModelID  int       `json:"model_id"`
	Year     int       `json:"year"`
	Vin      string    `json:"vin"`
	Lpn      string    `json:"lpn"`
	Version  int64     `json:"version"`
	OwnedBy  uuid.UUID `json:"owned_by"`
}

func vehicleBody(fixture testutil.VehicleFixture) map[string]any {
	return map[string]any{
		"nickname": fixture.Nickname,
		"make":     fixture.Make,
		"make_id":  fixture.MakeID,
		"model":    fixture.Model,
		"model_id": fixture.ModelID,
		"year":     fixture.Year,
		"vin":      fixture.Vin,
		"lpn":      fixture.Lpn,
	}
}

func assertVehicle(t *testing.T, got, want vehicleResponse) {
	t.Helper()
	if got.Nickname != want.Nickname {
		t.Errorf("expected nickname %q, got %q", want.Nickname, got.Nickname)
	}
	if got.Make != want.Make || got.MakeID != want.MakeID {
		t.Errorf("expected make %q/%d, got %q/%d", want.Make, want.MakeID, got.Make, got.MakeID)
	}
	if got.Model != want.Model || got.ModelID != want.ModelID {
		t.Errorf("expected model %q/%d, got %q/%d", want.Model, want.ModelID, got.Model, got.ModelID)
	}
	if got.Year != want.Year {
		t.Errorf("expected year %d, got %d", want.Year, got.Year)
	}
	if got.Vin != want.Vin {
		t.Errorf("expected vin %q, got %q", want.Vin, got.Vin)
	}
	if got.Lpn != want.Lpn {
		t.Errorf("expected lpn %q, got %q", want.Lpn, got.Lpn)
	}
	if got.Version != want.Version {
		t.Errorf("expected version %d, got %d", want.Version, got.Version)
	}
	if got.OwnedBy != want.OwnedBy {
		t.Errorf("expected owned_by %s, got %s", want.OwnedBy, got.OwnedBy)
	}
}

func expectedVehicle(fixture testutil.VehicleFixture) vehicleResponse {
	return vehicleResponse{
		ID:       fixture.ID,
		Nickname: fixture.Nickname,
		Make:     fixture.Make,
		MakeID:   fixture.MakeID,
		Model:    fixture.Model,
		ModelID:  fixture.ModelID,
		Year:     fixture.Year,
		Vin:      fixture.Vin,
		Lpn:      fixture.Lpn,
		Version:  1,
		OwnedBy:  fixture.OwnerID,
	}
}

func TestVehiclesVerifyAllExist(t *testing.T) {
	env := testutil.SetupTest(t)

	for _, vehicle := range testutil.Vehicles {
		w := testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/vehicle/"+vehicle.ID.String(), vehicle.OwnerID, nil)
		testutil.RequireStatus(t, w, http.StatusOK)

		var response vehicleResponse
		testutil.DecodeJSON(t, w, &response)
		assertVehicle(t, response, expectedVehicle(vehicle))
	}
}

func TestVehicleList(t *testing.T) {
	env := testutil.SetupTest(t)

	for userIndex, accessRow := range testutil.VehicleAccessMatrix {
		user := testutil.Users[userIndex]

		expected := 0
		for _, access := range accessRow {
			if access >= testutil.READ_ONLY {
				expected++
			}
		}

		w := testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/vehicle", user.ID, nil)
		testutil.RequireStatus(t, w, http.StatusOK)

		var ids []uuid.UUID
		testutil.DecodeJSON(t, w, &ids)
		if len(ids) != expected {
			t.Errorf("%s: expected %d vehicles, got %d", user.Username, expected, len(ids))
		}
	}
}

func TestVehiclePost(t *testing.T) {
	env := testutil.SetupTest(t)

	body := map[string]any{
		"nickname": "New Vehicle",
		"year":     2022,
		"make":     "TEST-Make",
		"model":    "TEST-Model",
		"vin":      "VIN-TEST-1234567890",
		"lpn":      "LPN-TEST-1234",
		"version":  99,
		"owned_by": testutil.User2ID,
	}

	w := testutil.RequestAsUser(t, env.Router, http.MethodPost, "/v1/vehicle", testutil.User1ID, testutil.JSONBody(t, body))
	testutil.RequireStatus(t, w, http.StatusCreated)

	location := w.Header().Get("Location")
	if location == "" {
		t.Fatal("expected Location header")
	}

	w = testutil.RequestAsUser(t, env.Router, http.MethodGet, location, testutil.User1ID, nil)
	testutil.RequireStatus(t, w, http.StatusOK)

	var response vehicleResponse
	testutil.DecodeJSON(t, w, &response)
	if response.Nickname != "New Vehicle" || response.Year != 2022 || response.Make != "TEST-Make" {
		t.Errorf("unexpected created vehicle: %+v", response)
	}
	if response.Version != 1 {
		t.Errorf("create should ignore a supplied version and start at 1, got %d", response.Version)
	}
	if response.OwnedBy != testutil.User1ID {
		t.Errorf("create should set owned_by to the caller, got %s", response.OwnedBy)
	}
}

func TestVehiclePostWithUUID(t *testing.T) {
	env := testutil.SetupTest(t)

	supplied := "019d437c-5f48-753b-8240-5263972f15f0"
	body := map[string]any{"nickname": "Supplied ID Vehicle", "year": 2022}

	w := testutil.RequestAsUser(t, env.Router, http.MethodPost, "/v1/vehicle/"+supplied, testutil.User1ID, testutil.JSONBody(t, body))
	testutil.RequireStatus(t, w, http.StatusCreated)

	location := w.Header().Get("Location")
	if !strings.HasSuffix(location, "/"+supplied) {
		t.Errorf("expected Location to end with %s, got %s", supplied, location)
	}

	w = testutil.RequestAsUser(t, env.Router, http.MethodGet, location, testutil.User1ID, nil)
	testutil.RequireStatus(t, w, http.StatusOK)
}

func TestVehiclePostDuplicateUUID(t *testing.T) {
	env := testutil.SetupTest(t)

	supplied := "019d437c-5f48-753b-8240-5263972f15f1"
	body := map[string]any{"nickname": "Duplicate"}

	w := testutil.RequestAsUser(t, env.Router, http.MethodPost, "/v1/vehicle/"+supplied, testutil.User1ID, testutil.JSONBody(t, body))
	testutil.RequireStatus(t, w, http.StatusCreated)

	w = testutil.RequestAsUser(t, env.Router, http.MethodPost, "/v1/vehicle/"+supplied, testutil.User1ID, testutil.JSONBody(t, body))
	testutil.RequireStatus(t, w, http.StatusConflict)
}

func TestVehiclePatch(t *testing.T) {
	env := testutil.SetupTest(t)

	vehicle := testutil.Vehicles[0]
	newNickname := "MODIFIED " + vehicle.Nickname

	w := testutil.RequestAsUser(t, env.Router, http.MethodPatch, "/v1/vehicle/"+vehicle.ID.String(), vehicle.OwnerID,
		testutil.JSONBody(t, map[string]any{"nickname": newNickname, "version": 1}))
	testutil.RequireStatus(t, w, http.StatusOK)

	var response vehicleResponse
	testutil.DecodeJSON(t, w, &response)
	if response.Nickname != newNickname {
		t.Errorf("expected nickname %q, got %q", newNickname, response.Nickname)
	}
	if response.Year != vehicle.Year || response.Vin != vehicle.Vin {
		t.Errorf("PATCH should not clear untouched fields: %+v", response)
	}
	if response.Version != 2 {
		t.Errorf("expected version 2 after the first update, got %d", response.Version)
	}
}

func TestVehiclePatchVersionConflict(t *testing.T) {
	env := testutil.SetupTest(t)

	vehicle := testutil.Vehicles[0]
	path := "/v1/vehicle/" + vehicle.ID.String()

	w := testutil.RequestAsUser(t, env.Router, http.MethodPatch, path, vehicle.OwnerID,
		testutil.JSONBody(t, map[string]any{"nickname": "first", "version": 1}))
	testutil.RequireStatus(t, w, http.StatusOK)

	var first vehicleResponse
	testutil.DecodeJSON(t, w, &first)
	if first.Version != 2 {
		t.Fatalf("expected version 2 after the first update, got %d", first.Version)
	}

	w = testutil.RequestAsUser(t, env.Router, http.MethodPatch, path, vehicle.OwnerID,
		testutil.JSONBody(t, map[string]any{"nickname": "second", "version": 1}))
	testutil.RequireStatus(t, w, http.StatusConflict)

	var conflict struct {
		Error   string `json:"error"`
		Version int64  `json:"version"`
	}
	testutil.DecodeJSON(t, w, &conflict)
	if conflict.Error != "Version conflict" || conflict.Version != 2 {
		t.Errorf("expected a version conflict at version 2, got %+v", conflict)
	}

	w = testutil.RequestAsUser(t, env.Router, http.MethodGet, path, vehicle.OwnerID, nil)
	testutil.RequireStatus(t, w, http.StatusOK)

	var stored vehicleResponse
	testutil.DecodeJSON(t, w, &stored)
	if stored.Nickname != "first" || stored.Version != 2 {
		t.Errorf("the stale update must not be applied: %+v", stored)
	}
}

func TestVehiclePatchRequiresVersion(t *testing.T) {
	env := testutil.SetupTest(t)

	vehicle := testutil.Vehicles[0]
	path := "/v1/vehicle/" + vehicle.ID.String()

	bodies := map[string]map[string]any{
		"missing":  {"nickname": "missing version"},
		"zero":     {"nickname": "zero version", "version": 0},
		"negative": {"nickname": "negative version", "version": -1},
	}

	for name, body := range bodies {
		t.Run(name, func(t *testing.T) {
			w := testutil.RequestAsUser(t, env.Router, http.MethodPatch, path, vehicle.OwnerID, testutil.JSONBody(t, body))
			testutil.RequireStatus(t, w, http.StatusUnprocessableEntity)
		})
	}
}

func TestVehiclePatchIgnoresProtectedFields(t *testing.T) {
	env := testutil.SetupTest(t)

	vehicle := testutil.Vehicles[0]

	w := testutil.RequestAsUser(t, env.Router, http.MethodPatch, "/v1/vehicle/"+vehicle.ID.String(), vehicle.OwnerID,
		testutil.JSONBody(t, map[string]any{
			"nickname":   "renamed",
			"version":    1,
			"id":         testutil.Vehicle2ID,
			"ID":         testutil.Vehicle2ID,
			"created_by": testutil.User2ID,
			"owned_by":   testutil.User2ID,
			"owner_id":   testutil.User2ID,
		}))
	testutil.RequireStatus(t, w, http.StatusOK)

	var response vehicleResponse
	testutil.DecodeJSON(t, w, &response)
	if response.ID != vehicle.ID {
		t.Errorf("PATCH must not change the ID: got %s, want %s", response.ID, vehicle.ID)
	}
	if response.OwnedBy != vehicle.OwnerID {
		t.Errorf("PATCH must not change owned_by: got %s, want %s", response.OwnedBy, vehicle.OwnerID)
	}

	var stored struct {
		CreatedBy uuid.UUID
		OwnedBy   uuid.UUID
	}
	if err := env.DB.GetGorm().Table("vehicles").Select("created_by, owned_by").
		Where("id = ?", vehicle.ID).Take(&stored).Error; err != nil {
		t.Fatalf("reading vehicle audit fields: %v", err)
	}
	if stored.CreatedBy != vehicle.OwnerID {
		t.Errorf("PATCH must not change created_by: got %s, want %s", stored.CreatedBy, vehicle.OwnerID)
	}
	if stored.OwnedBy != vehicle.OwnerID {
		t.Errorf("PATCH must not change owned_by: got %s, want %s", stored.OwnedBy, vehicle.OwnerID)
	}
}

func TestVehicleDelete(t *testing.T) {
	env := testutil.SetupTest(t)

	// Vehicle 2 has maintenance records and access rows to exercise the cascade.
	vehicle := testutil.Vehicles[1]

	w := testutil.RequestAsUser(t, env.Router, http.MethodDelete, "/v1/vehicle/"+vehicle.ID.String(), vehicle.OwnerID, nil)
	testutil.RequireStatus(t, w, http.StatusNoContent)

	w = testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/vehicle/"+vehicle.ID.String(), vehicle.OwnerID, nil)
	testutil.RequireStatus(t, w, http.StatusNotFound)

	db := env.DB.GetGorm()
	checks := []struct {
		table string
		where string
		args  []any
	}{
		{"vehicles", "id = ? AND deleted_at IS NULL", []any{vehicle.ID}},
		{"maintenance_records", "vehicle_id = ? AND deleted_at IS NULL", []any{vehicle.ID}},
		{"maintenance_record_files", "deleted_at IS NULL AND maintenance_record_id IN (SELECT id FROM maintenance_records WHERE vehicle_id = ?)", []any{vehicle.ID}},
		{"vehicle_user_access", "vehicle_id = ? AND deleted_at IS NULL", []any{vehicle.ID}},
		{"vehicle_user_access_pending", "vehicle_id = ? AND deleted_at IS NULL", []any{vehicle.ID}},
	}
	for _, check := range checks {
		var count int64
		if err := db.Table(check.table).Where(check.where, check.args...).Count(&count).Error; err != nil {
			t.Fatalf("counting %s: %v", check.table, err)
		}
		if count != 0 {
			t.Errorf("expected no active %s rows for deleted vehicle, found %d", check.table, count)
		}
	}
}

func TestVehicleReadPermissions(t *testing.T) {
	env := testutil.SetupTest(t)

	for userIndex, accessRow := range testutil.VehicleAccessMatrix {
		user := testutil.Users[userIndex]
		for vehicleIndex, access := range accessRow {
			vehicle := testutil.Vehicles[vehicleIndex]

			w := testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/vehicle/"+vehicle.ID.String(), user.ID, nil)

			if access == testutil.NO_ACCESS {
				if w.Code != http.StatusNotFound {
					t.Errorf("%s reading %s: expected 404, got %d", user.Username, vehicle.Nickname, w.Code)
				}
				continue
			}

			// READ_BY_INVITE and above may view the vehicle itself.
			if w.Code != http.StatusOK {
				t.Errorf("%s reading %s: expected 200, got %d", user.Username, vehicle.Nickname, w.Code)
				continue
			}

			var response vehicleResponse
			testutil.DecodeJSON(t, w, &response)
			assertVehicle(t, response, expectedVehicle(vehicle))
		}
	}
}

func TestVehicleWritePermissions(t *testing.T) {
	env := testutil.SetupTest(t)

	for userIndex, accessRow := range testutil.VehicleAccessMatrix {
		user := testutil.Users[userIndex]
		for vehicleIndex, access := range accessRow {
			vehicle := testutil.Vehicles[vehicleIndex]
			newNickname := "MODIFIED " + vehicle.Nickname
			version := testutil.CurrentVersion(t, env.DB.GetGorm(), "vehicles", vehicle.ID)

			w := testutil.RequestAsUser(t, env.Router, http.MethodPatch, "/v1/vehicle/"+vehicle.ID.String(), user.ID,
				testutil.JSONBody(t, map[string]any{"nickname": newNickname, "version": version}))

			switch {
			case access == testutil.WRITE:
				if w.Code != http.StatusOK {
					t.Errorf("%s patching %s: expected 200, got %d", user.Username, vehicle.Nickname, w.Code)
					continue
				}
				var response vehicleResponse
				testutil.DecodeJSON(t, w, &response)
				if response.Nickname != newNickname {
					t.Errorf("%s patching %s: expected nickname %q, got %q", user.Username, vehicle.Nickname, newNickname, response.Nickname)
				}
			case access == testutil.READ_ONLY:
				if w.Code != http.StatusForbidden {
					t.Errorf("%s patching %s: expected 403, got %d", user.Username, vehicle.Nickname, w.Code)
				}
			default:
				if w.Code != http.StatusNotFound {
					t.Errorf("%s patching %s: expected 404, got %d", user.Username, vehicle.Nickname, w.Code)
				}
			}
		}
	}
}

func TestVehicleDeletePermissions(t *testing.T) {
	env := testutil.SetupTest(t)
	tx := env.DB.GetGorm()

	for userIndex, accessRow := range testutil.VehicleAccessMatrix {
		user := testutil.Users[userIndex]
		for vehicleIndex, access := range accessRow {
			vehicle := testutil.Vehicles[vehicleIndex]

			if err := tx.SavePoint("delete_permissions").Error; err != nil {
				t.Fatalf("creating savepoint: %v", err)
			}

			w := testutil.RequestAsUser(t, env.Router, http.MethodDelete, "/v1/vehicle/"+vehicle.ID.String(), user.ID, nil)

			if err := tx.RollbackTo("delete_permissions").Error; err != nil {
				t.Fatalf("rolling back to savepoint: %v", err)
			}

			switch {
			case vehicle.OwnerID == user.ID:
				if w.Code != http.StatusNoContent {
					t.Errorf("owner %s deleting %s: expected 204, got %d", user.Username, vehicle.Nickname, w.Code)
				}
			case access >= testutil.READ_ONLY:
				if w.Code != http.StatusForbidden {
					t.Errorf("readable non-owner %s deleting %s: expected 403, got %d", user.Username, vehicle.Nickname, w.Code)
				}
			default:
				if w.Code != http.StatusNotFound {
					t.Errorf("%s deleting %s: expected 404, got %d", user.Username, vehicle.Nickname, w.Code)
				}
			}
		}
	}
}

func TestVehiclePermissionsFollowOwnership(t *testing.T) {
	env := testutil.SetupTest(t)

	vehicle := testutil.Vehicles[0]
	if err := env.DB.GetGorm().Table("vehicles").
		Where("id = ?", vehicle.ID).Update("owned_by", testutil.User3ID).Error; err != nil {
		t.Fatalf("transferring vehicle ownership: %v", err)
	}

	w := testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/vehicle", testutil.User3ID, nil)
	testutil.RequireStatus(t, w, http.StatusOK)
	var ids []uuid.UUID
	testutil.DecodeJSON(t, w, &ids)
	if !slices.Contains(ids, vehicle.ID) {
		t.Errorf("expected the new owner's list to contain %s, got %v", vehicle.ID, ids)
	}

	w = testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/vehicle/"+vehicle.ID.String(), testutil.User3ID, nil)
	testutil.RequireStatus(t, w, http.StatusOK)
	var response vehicleResponse
	testutil.DecodeJSON(t, w, &response)
	if response.OwnedBy != testutil.User3ID {
		t.Errorf("expected owned_by %s, got %s", testutil.User3ID, response.OwnedBy)
	}

	version := testutil.CurrentVersion(t, env.DB.GetGorm(), "vehicles", vehicle.ID)
	w = testutil.RequestAsUser(t, env.Router, http.MethodPatch, "/v1/vehicle/"+vehicle.ID.String(), testutil.User3ID,
		testutil.JSONBody(t, map[string]any{"nickname": "Transferred", "version": version}))
	testutil.RequireStatus(t, w, http.StatusOK)

	w = testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/vehicle/"+vehicle.ID.String(), testutil.User1ID, nil)
	testutil.RequireStatus(t, w, http.StatusNotFound)
	w = testutil.RequestAsUser(t, env.Router, http.MethodDelete, "/v1/vehicle/"+vehicle.ID.String(), testutil.User1ID, nil)
	testutil.RequireStatus(t, w, http.StatusNotFound)

	w = testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/vehicle", testutil.User1ID, nil)
	testutil.RequireStatus(t, w, http.StatusOK)
	ids = nil
	testutil.DecodeJSON(t, w, &ids)
	if slices.Contains(ids, vehicle.ID) {
		t.Errorf("expected the creator's list to exclude %s, got %v", vehicle.ID, ids)
	}

	w = testutil.RequestAsUser(t, env.Router, http.MethodDelete, "/v1/vehicle/"+vehicle.ID.String(), testutil.User3ID, nil)
	testutil.RequireStatus(t, w, http.StatusNoContent)
}
