package vehicles_test

import (
	"auto-myself-api/internal/testutil"
	"net/http"
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
		testutil.JSONBody(t, map[string]any{"nickname": newNickname}))
	testutil.RequireStatus(t, w, http.StatusOK)

	var response vehicleResponse
	testutil.DecodeJSON(t, w, &response)
	if response.Nickname != newNickname {
		t.Errorf("expected nickname %q, got %q", newNickname, response.Nickname)
	}
	if response.Year != vehicle.Year || response.Vin != vehicle.Vin {
		t.Errorf("PATCH should not clear untouched fields: %+v", response)
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

			w := testutil.RequestAsUser(t, env.Router, http.MethodPatch, "/v1/vehicle/"+vehicle.ID.String(), user.ID,
				testutil.JSONBody(t, map[string]any{"nickname": newNickname}))

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
