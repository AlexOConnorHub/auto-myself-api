package maintenance_test

import (
	"auto-myself-api/internal/testutil"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
)

type maintenanceResponse struct {
	ID           uuid.UUID `json:"ID"`
	CreatedBy    uuid.UUID `json:"created_by"`
	VehicleID    uuid.UUID `json:"vehicle_id"`
	Timestamp    time.Time `json:"timestamp"`
	Odometer     int       `json:"odometer"`
	Notes        string    `json:"notes"`
	Type         string    `json:"type"`
	Interval     int       `json:"interval"`
	IntervalType string    `json:"interval_type"`
	Cost         string    `json:"cost"`
	Version      int64     `json:"version"`
}

func expectedMaintenance(fixture testutil.MaintenanceFixture) maintenanceResponse {
	timestamp, err := time.Parse("2006-01-02", fixture.Timestamp)
	if err != nil {
		panic("invalid maintenance fixture timestamp " + fixture.Timestamp + ": " + err.Error())
	}

	return maintenanceResponse{
		ID:           fixture.ID,
		CreatedBy:    fixture.CreatedBy,
		VehicleID:    fixture.VehicleID,
		Timestamp:    timestamp,
		Odometer:     fixture.Odometer,
		Notes:        fixture.Notes,
		Type:         fixture.Type,
		Interval:     fixture.Interval,
		IntervalType: fixture.IntervalType,
		Cost:         fixture.Cost,
		Version:      1,
	}
}

func assertMaintenance(t *testing.T, got, want maintenanceResponse) {
	t.Helper()
	if got.ID != want.ID {
		t.Errorf("expected ID %s, got %s", want.ID, got.ID)
	}
	if got.VehicleID != want.VehicleID {
		t.Errorf("expected vehicle_id %s, got %s", want.VehicleID, got.VehicleID)
	}
	if want.CreatedBy != uuid.Nil && got.CreatedBy != want.CreatedBy {
		t.Errorf("expected created_by %s, got %s", want.CreatedBy, got.CreatedBy)
	}
	if got.Timestamp.Format("2006-01-02") != want.Timestamp.Format("2006-01-02") {
		t.Errorf("expected timestamp %s, got %s", want.Timestamp.Format("2006-01-02"), got.Timestamp.Format("2006-01-02"))
	}
	if got.Odometer != want.Odometer {
		t.Errorf("expected odometer %d, got %d", want.Odometer, got.Odometer)
	}
	if got.Notes != want.Notes {
		t.Errorf("expected notes %q, got %q", want.Notes, got.Notes)
	}
	if got.Type != want.Type {
		t.Errorf("expected type %q, got %q", want.Type, got.Type)
	}
	if got.Interval != want.Interval {
		t.Errorf("expected interval %d, got %d", want.Interval, got.Interval)
	}
	if got.IntervalType != want.IntervalType {
		t.Errorf("expected interval_type %q, got %q", want.IntervalType, got.IntervalType)
	}
	if got.Cost != want.Cost {
		t.Errorf("expected cost %q, got %q", want.Cost, got.Cost)
	}
	if got.Version != want.Version {
		t.Errorf("expected version %d, got %d", want.Version, got.Version)
	}
}

func vehicleOwner(vehicleID uuid.UUID) uuid.UUID {
	for _, vehicle := range testutil.Vehicles {
		if vehicle.ID == vehicleID {
			return vehicle.OwnerID
		}
	}
	panic("unknown vehicle " + vehicleID.String())
}

func vehicleIndex(vehicleID uuid.UUID) int {
	for i, vehicle := range testutil.Vehicles {
		if vehicle.ID == vehicleID {
			return i
		}
	}
	panic("unknown vehicle " + vehicleID.String())
}

func TestMaintenancesVerifyAllExist(t *testing.T) {
	env := testutil.SetupTest(t)

	for _, fixture := range testutil.Maintenances {
		w := testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/maintenance/"+fixture.ID.String(), vehicleOwner(fixture.VehicleID), nil)
		testutil.RequireStatus(t, w, http.StatusOK)

		var response maintenanceResponse
		testutil.DecodeJSON(t, w, &response)
		assertMaintenance(t, response, expectedMaintenance(fixture))
	}
}

func TestMaintenanceList(t *testing.T) {
	env := testutil.SetupTest(t)

	counts := make(map[uuid.UUID]int)
	for _, fixture := range testutil.Maintenances {
		counts[fixture.VehicleID]++
	}

	for userIndex, accessRow := range testutil.VehicleAccessMatrix {
		user := testutil.Users[userIndex]
		for vi, vehicle := range testutil.Vehicles {
			w := testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/vehicle/"+vehicle.ID.String()+"/maintenance", user.ID, nil)

			if accessRow[vi] < testutil.READ_ONLY {
				if w.Code != http.StatusNotFound {
					t.Errorf("%s listing maintenance for %s: expected 404, got %d", user.Username, vehicle.Nickname, w.Code)
				}
				continue
			}

			testutil.RequireStatus(t, w, http.StatusOK)
			var ids []uuid.UUID
			testutil.DecodeJSON(t, w, &ids)
			if len(ids) != counts[vehicle.ID] {
				t.Errorf("%s listing maintenance for %s: expected %d, got %d", user.Username, vehicle.Nickname, counts[vehicle.ID], len(ids))
			}
		}
	}
}

func TestMaintenancePost(t *testing.T) {
	env := testutil.SetupTest(t)

	body := map[string]any{
		"vehicle_id":    testutil.Vehicle1ID,
		"timestamp":     time.Now().Format(time.RFC3339),
		"odometer":      10000,
		"notes":         "TEST Initial maintenance",
		"type":          "TEST Oil Change",
		"interval":      5000,
		"interval_type": "miles",
		"cost":          "100.00",
		"version":       99,
	}

	w := testutil.RequestAsUser(t, env.Router, http.MethodPost, "/v1/maintenance", testutil.User1ID, testutil.JSONBody(t, body))
	testutil.RequireStatus(t, w, http.StatusCreated)

	location := w.Header().Get("Location")
	if location == "" {
		t.Fatal("expected Location header")
	}

	w = testutil.RequestAsUser(t, env.Router, http.MethodGet, location, testutil.User1ID, nil)
	testutil.RequireStatus(t, w, http.StatusOK)

	var response maintenanceResponse
	testutil.DecodeJSON(t, w, &response)
	if response.VehicleID != testutil.Vehicle1ID || response.Notes != "TEST Initial maintenance" || response.Odometer != 10000 {
		t.Errorf("unexpected created maintenance record: %+v", response)
	}
	if response.CreatedBy != testutil.User1ID {
		t.Errorf("expected created_by %s, got %s", testutil.User1ID, response.CreatedBy)
	}
	if response.Version != 1 {
		t.Errorf("create should ignore a supplied version and start at 1, got %d", response.Version)
	}
}

func TestMaintenancePostWithUUID(t *testing.T) {
	env := testutil.SetupTest(t)

	supplied := "019d4374-c861-795b-8db5-5cbecdd1378e"
	body := map[string]any{
		"vehicle_id": testutil.Vehicle1ID,
		"timestamp":  time.Now().Format(time.RFC3339),
		"odometer":   10000,
		"notes":      "TEST maintenance with supplied ID",
	}

	w := testutil.RequestAsUser(t, env.Router, http.MethodPost, "/v1/maintenance/"+supplied, testutil.User1ID, testutil.JSONBody(t, body))
	testutil.RequireStatus(t, w, http.StatusCreated)

	location := w.Header().Get("Location")
	if !strings.HasSuffix(location, "/"+supplied) {
		t.Errorf("expected Location to end with %s, got %s", supplied, location)
	}

	w = testutil.RequestAsUser(t, env.Router, http.MethodGet, location, testutil.User1ID, nil)
	testutil.RequireStatus(t, w, http.StatusOK)
}

func TestMaintenanceCreatePermissions(t *testing.T) {
	env := testutil.SetupTest(t)

	body := func(vehicleID uuid.UUID) map[string]any {
		return map[string]any{"vehicle_id": vehicleID, "notes": "permission test"}
	}

	tests := []struct {
		name      string
		user      uuid.UUID
		vehicleID uuid.UUID
		want      int
	}{
		{"owner", testutil.User1ID, testutil.Vehicle1ID, http.StatusCreated},
		{"write recipient", testutil.User2ID, testutil.Vehicle2ID, http.StatusCreated},
		{"read-only recipient", testutil.User3ID, testutil.Vehicle2ID, http.StatusForbidden},
		{"pending recipient", testutil.User4ID, testutil.Vehicle4ID, http.StatusNotFound},
		{"no access", testutil.User5ID, testutil.Vehicle1ID, http.StatusNotFound},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w := testutil.RequestAsUser(t, env.Router, http.MethodPost, "/v1/maintenance", test.user, testutil.JSONBody(t, body(test.vehicleID)))
			testutil.RequireStatus(t, w, test.want)
		})
	}
}

func TestMaintenancePostRejectsMissingVehicle(t *testing.T) {
	env := testutil.SetupTest(t)

	w := testutil.RequestAsUser(t, env.Router, http.MethodPost, "/v1/maintenance", testutil.User1ID,
		testutil.JSONBody(t, map[string]any{"notes": "no vehicle"}))
	testutil.RequireStatus(t, w, http.StatusUnprocessableEntity)
}

func TestMaintenancePatch(t *testing.T) {
	env := testutil.SetupTest(t)

	fixture := testutil.Maintenances[0]
	newNotes := fixture.Notes + " MODIFIED"

	w := testutil.RequestAsUser(t, env.Router, http.MethodPatch, "/v1/maintenance/"+fixture.ID.String(), vehicleOwner(fixture.VehicleID),
		testutil.JSONBody(t, map[string]any{"notes": newNotes, "version": 1}))
	testutil.RequireStatus(t, w, http.StatusOK)

	var response maintenanceResponse
	testutil.DecodeJSON(t, w, &response)
	if response.Notes != newNotes {
		t.Errorf("expected notes %q, got %q", newNotes, response.Notes)
	}
	if response.VehicleID != fixture.VehicleID {
		t.Errorf("expected vehicle_id %s, got %s", fixture.VehicleID, response.VehicleID)
	}
	if response.Odometer != fixture.Odometer || response.Cost != fixture.Cost {
		t.Errorf("PATCH should not clear untouched fields: %+v", response)
	}
	if response.Version != 2 {
		t.Errorf("expected version 2 after the first update, got %d", response.Version)
	}
}

func TestMaintenancePatchCannotMoveVehicle(t *testing.T) {
	env := testutil.SetupTest(t)

	fixture := testutil.Maintenances[0]
	otherVehicle := testutil.Vehicle2ID

	w := testutil.RequestAsUser(t, env.Router, http.MethodPatch, "/v1/maintenance/"+fixture.ID.String(), vehicleOwner(fixture.VehicleID),
		testutil.JSONBody(t, map[string]any{"vehicle_id": otherVehicle, "notes": "attempted move", "version": 1}))
	testutil.RequireStatus(t, w, http.StatusOK)

	var response maintenanceResponse
	testutil.DecodeJSON(t, w, &response)
	if response.VehicleID != fixture.VehicleID {
		t.Errorf("maintenance record moved to %s; expected to stay on %s", response.VehicleID, fixture.VehicleID)
	}
}

func TestMaintenancePatchVersionConflict(t *testing.T) {
	env := testutil.SetupTest(t)

	fixture := testutil.Maintenances[0]
	owner := vehicleOwner(fixture.VehicleID)
	path := "/v1/maintenance/" + fixture.ID.String()

	w := testutil.RequestAsUser(t, env.Router, http.MethodPatch, path, owner,
		testutil.JSONBody(t, map[string]any{"notes": "first", "version": 1}))
	testutil.RequireStatus(t, w, http.StatusOK)

	var first maintenanceResponse
	testutil.DecodeJSON(t, w, &first)
	if first.Version != 2 {
		t.Fatalf("expected version 2 after the first update, got %d", first.Version)
	}

	w = testutil.RequestAsUser(t, env.Router, http.MethodPatch, path, owner,
		testutil.JSONBody(t, map[string]any{"notes": "second", "version": 1}))
	testutil.RequireStatus(t, w, http.StatusConflict)

	var conflict struct {
		Error   string `json:"error"`
		Version int64  `json:"version"`
	}
	testutil.DecodeJSON(t, w, &conflict)
	if conflict.Error != "Version conflict" || conflict.Version != 2 {
		t.Errorf("expected a version conflict at version 2, got %+v", conflict)
	}

	w = testutil.RequestAsUser(t, env.Router, http.MethodGet, path, owner, nil)
	testutil.RequireStatus(t, w, http.StatusOK)

	var stored maintenanceResponse
	testutil.DecodeJSON(t, w, &stored)
	if stored.Notes != "first" || stored.Version != 2 {
		t.Errorf("the stale update must not be applied: %+v", stored)
	}
}

func TestMaintenancePatchRequiresVersion(t *testing.T) {
	env := testutil.SetupTest(t)

	fixture := testutil.Maintenances[0]
	path := "/v1/maintenance/" + fixture.ID.String()

	bodies := map[string]map[string]any{
		"missing":  {"notes": "missing version"},
		"zero":     {"notes": "zero version", "version": 0},
		"negative": {"notes": "negative version", "version": -1},
	}

	for name, body := range bodies {
		t.Run(name, func(t *testing.T) {
			w := testutil.RequestAsUser(t, env.Router, http.MethodPatch, path, vehicleOwner(fixture.VehicleID), testutil.JSONBody(t, body))
			testutil.RequireStatus(t, w, http.StatusUnprocessableEntity)
		})
	}
}

func TestMaintenancePatchIgnoresProtectedFields(t *testing.T) {
	env := testutil.SetupTest(t)

	fixture := testutil.Maintenances[0]

	w := testutil.RequestAsUser(t, env.Router, http.MethodPatch, "/v1/maintenance/"+fixture.ID.String(), vehicleOwner(fixture.VehicleID),
		testutil.JSONBody(t, map[string]any{
			"notes":      "renamed",
			"version":    1,
			"ID":         testutil.Maintenances[1].ID,
			"id":         testutil.Maintenances[1].ID,
			"created_by": testutil.User2ID,
		}))
	testutil.RequireStatus(t, w, http.StatusOK)

	var response maintenanceResponse
	testutil.DecodeJSON(t, w, &response)
	if response.ID != fixture.ID {
		t.Errorf("PATCH must not change the ID: got %s, want %s", response.ID, fixture.ID)
	}
	if response.CreatedBy != fixture.CreatedBy {
		t.Errorf("PATCH must not change created_by: got %s, want %s", response.CreatedBy, fixture.CreatedBy)
	}
}

func TestMaintenanceDelete(t *testing.T) {
	env := testutil.SetupTest(t)

	fixture := testutil.Maintenances[0]
	owner := vehicleOwner(fixture.VehicleID)

	w := testutil.RequestAsUser(t, env.Router, http.MethodDelete, "/v1/maintenance/"+fixture.ID.String(), owner, nil)
	testutil.RequireStatus(t, w, http.StatusNoContent)

	w = testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/maintenance/"+fixture.ID.String(), owner, nil)
	testutil.RequireStatus(t, w, http.StatusNotFound)
}

func TestMaintenanceReadPermissions(t *testing.T) {
	env := testutil.SetupTest(t)

	for _, fixture := range testutil.Maintenances {
		vi := vehicleIndex(fixture.VehicleID)
		for userIndex, accessRow := range testutil.VehicleAccessMatrix {
			user := testutil.Users[userIndex]

			w := testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/maintenance/"+fixture.ID.String(), user.ID, nil)

			if accessRow[vi] < testutil.READ_ONLY {
				if w.Code != http.StatusNotFound {
					t.Errorf("%s reading maintenance %s: expected 404, got %d", user.Username, fixture.ID, w.Code)
				}
				continue
			}

			if w.Code != http.StatusOK {
				t.Errorf("%s reading maintenance %s: expected 200, got %d", user.Username, fixture.ID, w.Code)
				continue
			}

			var response maintenanceResponse
			testutil.DecodeJSON(t, w, &response)
			assertMaintenance(t, response, expectedMaintenance(fixture))
		}
	}
}

func TestMaintenanceWritePermissions(t *testing.T) {
	env := testutil.SetupTest(t)

	for _, fixture := range testutil.Maintenances {
		vi := vehicleIndex(fixture.VehicleID)
		for userIndex, accessRow := range testutil.VehicleAccessMatrix {
			user := testutil.Users[userIndex]
			version := testutil.CurrentVersion(t, env.DB.GetGorm(), "maintenance_records", fixture.ID)

			w := testutil.RequestAsUser(t, env.Router, http.MethodPatch, "/v1/maintenance/"+fixture.ID.String(), user.ID,
				testutil.JSONBody(t, map[string]any{"notes": fixture.Notes + " MODIFIED", "version": version}))

			switch {
			case accessRow[vi] == testutil.WRITE:
				if w.Code != http.StatusOK {
					t.Errorf("%s patching maintenance %s: expected 200, got %d", user.Username, fixture.ID, w.Code)
				}
			case accessRow[vi] == testutil.READ_ONLY:
				if w.Code != http.StatusForbidden {
					t.Errorf("%s patching maintenance %s: expected 403, got %d", user.Username, fixture.ID, w.Code)
				}
			default:
				if w.Code != http.StatusNotFound {
					t.Errorf("%s patching maintenance %s: expected 404, got %d", user.Username, fixture.ID, w.Code)
				}
			}
		}
	}
}

func TestMaintenanceDeletePermissions(t *testing.T) {
	env := testutil.SetupTest(t)
	tx := env.DB.GetGorm()

	for _, fixture := range testutil.Maintenances {
		vi := vehicleIndex(fixture.VehicleID)
		for userIndex, accessRow := range testutil.VehicleAccessMatrix {
			user := testutil.Users[userIndex]

			if err := tx.SavePoint("maintenance_delete").Error; err != nil {
				t.Fatalf("creating savepoint: %v", err)
			}

			w := testutil.RequestAsUser(t, env.Router, http.MethodDelete, "/v1/maintenance/"+fixture.ID.String(), user.ID, nil)

			if err := tx.RollbackTo("maintenance_delete").Error; err != nil {
				t.Fatalf("rolling back to savepoint: %v", err)
			}

			switch {
			case accessRow[vi] == testutil.WRITE:
				if w.Code != http.StatusNoContent {
					t.Errorf("%s deleting maintenance %s: expected 204, got %d", user.Username, fixture.ID, w.Code)
				}
			case accessRow[vi] == testutil.READ_ONLY:
				if w.Code != http.StatusForbidden {
					t.Errorf("%s deleting maintenance %s: expected 403, got %d", user.Username, fixture.ID, w.Code)
				}
			default:
				if w.Code != http.StatusNotFound {
					t.Errorf("%s deleting maintenance %s: expected 404, got %d", user.Username, fixture.ID, w.Code)
				}
			}
		}
	}
}

func TestMaintenanceEdgeCases(t *testing.T) {
	env := testutil.SetupTest(t)

	w := testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/maintenance/invalid-uuid", testutil.User1ID, nil)
	testutil.RequireStatus(t, w, http.StatusBadRequest)

	w = testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/vehicle/0195fba0-de33-79e3-ab0c-7357d15cbb3d/maintenance", testutil.User1ID, nil)
	testutil.RequireStatus(t, w, http.StatusNotFound)

	w = testutil.RequestAsUser(t, env.Router, http.MethodPost, "/v1/maintenance", testutil.User1ID,
		strings.NewReader("{invalid-json}"))
	testutil.RequireStatus(t, w, http.StatusUnprocessableEntity)
}
