package export_test

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"auto-myself-api/internal/auth"
	"auto-myself-api/internal/export"
	"auto-myself-api/internal/maintenance"
	"auto-myself-api/internal/reports"
	"auto-myself-api/internal/testutil"
	"auto-myself-api/internal/vehicles"

	"github.com/gofrs/uuid/v5"
)

func TestExportOwnedAndSharedVehicles(t *testing.T) {
	env := testutil.SetupTest(t)
	entries := exportArchive(t, env, testutil.User1ID)

	user := decodeEntry[export.UserExport](t, entries, "user.json")
	if user.ID != testutil.User1ID || user.Username != "User 1" || user.Role != "admin" {
		t.Errorf("unexpected user.json: %+v", user)
	}
	if len(user.Identities) != 0 {
		t.Errorf("seeded users have no identities, got %+v", user.Identities)
	}

	if _, exists := entries["vehicles/"]; !exists {
		t.Error("expected a vehicles directory entry")
	}
	for _, vehicleID := range []uuid.UUID{testutil.Vehicle1ID, testutil.Vehicle2ID} {
		if _, exists := entries["vehicles/"+vehicleID.String()+"/vehicle.json"]; !exists {
			t.Errorf("expected vehicle %s in the export", vehicleID)
		}
	}
	for _, vehicleID := range []uuid.UUID{testutil.Vehicle3ID, testutil.Vehicle4ID, testutil.Vehicle5ID} {
		requireNoEntriesWithPrefix(t, entries, "vehicles/"+vehicleID.String()+"/")
	}

	records := decodeEntry[[]export.MaintenanceRecordExport](t, entries,
		"vehicles/"+testutil.Vehicle2ID.String()+"/maintenance_records.json")
	if len(records) != 8 {
		t.Fatalf("expected the full Vehicle 2 history (8 records), got %d", len(records))
	}

	authors := make(map[uuid.UUID]int)
	for i, record := range records {
		authors[record.CreatedBy]++
		if record.VehicleID != testutil.Vehicle2ID {
			t.Errorf("record %s belongs to %s, not Vehicle 2", record.ID, record.VehicleID)
		}
		if i > 0 && record.Timestamp.Before(records[i-1].Timestamp) {
			t.Errorf("records are not ordered by timestamp: %s before %s", record.Timestamp, records[i-1].Timestamp)
		}
	}
	for _, authorID := range []uuid.UUID{testutil.User1ID, testutil.User2ID, testutil.User7ID, testutil.User8ID} {
		if authors[authorID] != 2 {
			t.Errorf("expected 2 Vehicle 2 records authored by %s, got %d", authorID, authors[authorID])
		}
	}

	vehicleOneRecords := decodeEntry[[]export.MaintenanceRecordExport](t, entries,
		"vehicles/"+testutil.Vehicle1ID.String()+"/maintenance_records.json")
	if len(vehicleOneRecords) != 2 {
		t.Errorf("expected 2 Vehicle 1 records, got %d", len(vehicleOneRecords))
	}

	vehicle := decodeEntry[export.VehicleExport](t, entries,
		"vehicles/"+testutil.Vehicle2ID.String()+"/vehicle.json")
	if vehicle.CreatedBy != testutil.User1ID || vehicle.Nickname != "Vehicle 2" {
		t.Errorf("unexpected vehicle.json: %+v", vehicle)
	}
}

func TestExportIncludesReadme(t *testing.T) {
	env := testutil.SetupTest(t)

	entries := exportArchive(t, env, testutil.User1ID)
	readme, exists := entries["README.md"]
	if !exists {
		t.Fatalf("expected README.md at the archive root; entries: %s", strings.Join(entryNames(entries), ", "))
	}
	for _, want := range []string{"24 hours", "signed_url", "maintenance_records.json"} {
		if !strings.Contains(string(readme), want) {
			t.Errorf("README.md does not mention %q", want)
		}
	}
}

func TestExportSharedVehicleForCollaborators(t *testing.T) {
	env := testutil.SetupTest(t)

	for _, userID := range []uuid.UUID{testutil.User2ID, testutil.User3ID} {
		t.Run(userID.String(), func(t *testing.T) {
			entries := exportArchive(t, env, userID)

			if _, exists := entries["vehicles/"+testutil.Vehicle2ID.String()+"/vehicle.json"]; !exists {
				t.Errorf("expected the shared Vehicle 2 for user %s", userID)
			}
			requireNoEntriesWithPrefix(t, entries, "vehicles/"+testutil.Vehicle1ID.String()+"/")

			records := decodeEntry[[]export.MaintenanceRecordExport](t, entries,
				"vehicles/"+testutil.Vehicle2ID.String()+"/maintenance_records.json")
			if len(records) != 8 {
				t.Fatalf("expected the full shared history, got %d records", len(records))
			}

			foundOwnerRecord := false
			for _, record := range records {
				if record.CreatedBy == testutil.User1ID {
					foundOwnerRecord = true
					break
				}
			}
			if !foundOwnerRecord {
				t.Error("a collaborator's export must include records authored by the vehicle owner")
			}
		})
	}
}

func TestExportWithoutAccessibleVehicles(t *testing.T) {
	env := testutil.SetupTest(t)
	entries := exportArchive(t, env, testutil.User5ID)

	if _, exists := entries["vehicles/"]; !exists {
		t.Error("expected a vehicles directory entry even when the user has no vehicles")
	}
	for name := range entries {
		if strings.HasPrefix(name, "vehicles/") && name != "vehicles/" {
			t.Errorf("unexpected vehicle entry %s", name)
		}
	}
	// User 5 has a pending invite to Vehicle 4; a pending share is not access.
	requireNoEntriesWithPrefix(t, entries, "vehicles/"+testutil.Vehicle4ID.String()+"/")

	if _, exists := entries["reports/"]; exists {
		t.Error("a user with no reports must not get a reports directory")
	}

	user := decodeEntry[export.UserExport](t, entries, "user.json")
	if user.Username != "User 5" {
		t.Errorf("expected User 5, got %q", user.Username)
	}
}

func TestExportUserIdentityEmail(t *testing.T) {
	env := testutil.SetupTest(t)

	identity := auth.Identity{IdentityBase: auth.IdentityBase{
		Provider: "google",
		Subject:  "google-subject-user-1",
		UserID:   testutil.User1ID,
		Email:    "user1@example.com",
	}}
	if err := env.DB.GetGorm().Create(&identity).Error; err != nil {
		t.Fatalf("creating identity: %v", err)
	}

	entries := exportArchive(t, env, testutil.User1ID)
	user := decodeEntry[export.UserExport](t, entries, "user.json")
	if len(user.Identities) != 1 {
		t.Fatalf("expected one identity, got %+v", user.Identities)
	}
	if user.Identities[0].Provider != "google" || user.Identities[0].Email != "user1@example.com" {
		t.Errorf("unexpected identity: %+v", user.Identities[0])
	}
	if bytes.Contains(entries["user.json"], []byte("google-subject-user-1")) {
		t.Error("the identity subject must not be exported")
	}
}

func TestExportFileMetadataAndSignedURL(t *testing.T) {
	env := testutil.SetupTest(t)

	record := testutil.Maintenances[0]
	upload := testutil.RequestAsUser(t, env.Router, http.MethodPost,
		"/v1/file/maintenance/"+record.ID.String(), testutil.User1ID, bytes.NewReader([]byte("export attachment")))
	testutil.RequireStatus(t, upload, http.StatusCreated)

	var link struct {
		FileID uuid.UUID `json:"file_id"`
	}
	testutil.DecodeJSON(t, upload, &link)
	if link.FileID.IsNil() {
		t.Fatal("upload response did not include a file id")
	}

	entries := exportArchive(t, env, testutil.User1ID)
	name := "vehicles/" + testutil.Vehicle1ID.String() + "/files/" + link.FileID.String() + ".json"
	file := decodeEntry[export.FileExport](t, entries, name)
	if file.ID != link.FileID || file.MaintenanceRecordID != record.ID {
		t.Errorf("unexpected file identity: %+v", file)
	}
	if file.CreatedBy != testutil.User1ID {
		t.Errorf("expected the uploader as created_by, got %s", file.CreatedBy)
	}
	if file.Sha256Hash == "" || file.FileSize == 0 {
		t.Errorf("expected hash and size metadata, got %+v", file)
	}
	if !strings.HasPrefix(file.SignedURL, "https://storage.test/") {
		t.Errorf("expected a signed URL, got %q", file.SignedURL)
	}
	if bytes.Contains(entries[name], []byte("storage_key")) {
		t.Error("the storage key must not be exported")
	}
	if ttl := env.Storage.LastSignedURLTTL(); ttl != export.SignedURLTTL {
		t.Errorf("expected a %s signed URL, got %s", export.SignedURLTTL, ttl)
	}
}

func TestExportReports(t *testing.T) {
	env := testutil.SetupTest(t)

	create := testutil.RequestAsUser(t, env.Router, http.MethodPost, "/v1/report", testutil.User1ID,
		testutil.JSONBody(t, map[string]any{
			"reported_topic": "vehicle",
			"reported_id":    testutil.Vehicle1ID.String(),
			"reason":         "export test",
		}))
	testutil.RequireStatus(t, create, http.StatusCreated)

	var created struct {
		ID uuid.UUID `json:"ID"`
	}
	testutil.DecodeJSON(t, create, &created)
	if created.ID.IsNil() {
		t.Fatal("report response did not include an ID")
	}

	if err := env.DB.GetGorm().Model(&reports.Report{}).Where("id = ?", created.ID).Updates(map[string]any{
		"internal_notes": "moderator only",
		"resolved_by":    testutil.User2ID,
		"resolved":       true,
	}).Error; err != nil {
		t.Fatalf("setting moderator fields: %v", err)
	}

	entries := exportArchive(t, env, testutil.User1ID)
	name := "reports/" + created.ID.String() + ".json"
	report := decodeEntry[export.ReportExport](t, entries, name)
	if report.ID != created.ID || report.CreatedBy != testutil.User1ID {
		t.Errorf("unexpected report identity: %+v", report)
	}
	if report.ReportedTopic != "vehicle" || report.ReportedID != testutil.Vehicle1ID || report.Reason != "export test" {
		t.Errorf("unexpected report content: %+v", report)
	}
	if !report.Resolved {
		t.Error("expected the resolved flag in the export")
	}
	if bytes.Contains(entries[name], []byte("internal_notes")) || bytes.Contains(entries[name], []byte("moderator only")) {
		t.Error("internal notes must not be exported")
	}
	if bytes.Contains(entries[name], []byte("resolved_by")) {
		t.Error("the resolver must not be exported")
	}
}

func TestExportExcludesSensitiveData(t *testing.T) {
	env := testutil.SetupTest(t)

	identity := auth.Identity{IdentityBase: auth.IdentityBase{
		Provider: "google",
		Subject:  "subject-leak-check",
		UserID:   testutil.User1ID,
		Email:    "user1@example.com",
	}}
	if err := env.DB.GetGorm().Create(&identity).Error; err != nil {
		t.Fatalf("creating identity: %v", err)
	}

	record := testutil.Maintenances[0]
	upload := testutil.RequestAsUser(t, env.Router, http.MethodPost,
		"/v1/file/maintenance/"+record.ID.String(), testutil.User1ID, bytes.NewReader([]byte("secret bytes")))
	testutil.RequireStatus(t, upload, http.StatusCreated)

	create := testutil.RequestAsUser(t, env.Router, http.MethodPost, "/v1/report", testutil.User1ID,
		testutil.JSONBody(t, map[string]any{
			"reported_topic": "vehicle",
			"reported_id":    testutil.Vehicle1ID.String(),
			"reason":         "sensitive-data check",
		}))
	testutil.RequireStatus(t, create, http.StatusCreated)
	var created struct {
		ID uuid.UUID `json:"ID"`
	}
	testutil.DecodeJSON(t, create, &created)
	if err := env.DB.GetGorm().Model(&reports.Report{}).Where("id = ?", created.ID).Updates(map[string]any{
		"internal_notes": "moderator only",
		"resolved_by":    testutil.User2ID,
	}).Error; err != nil {
		t.Fatalf("setting moderator fields: %v", err)
	}

	entries := exportArchive(t, env, testutil.User1ID)

	var raw bytes.Buffer
	for name, data := range entries {
		raw.WriteString(name)
		raw.Write(data)
		if name != "README.md" && name != "user.json" && !strings.HasPrefix(name, "vehicles/") && !strings.HasPrefix(name, "reports/") {
			t.Errorf("unexpected archive entry %s", name)
		}
	}

	for _, forbidden := range []string{
		"storage_key",
		"internal_notes",
		"resolved_by",
		"subject-leak-check",
		`"subject"`,
		"refresh_token",
		`"token"`,
		`"version"`,
		`"deleted_at"`,
		"User 2",
		"User 7",
	} {
		if bytes.Contains(raw.Bytes(), []byte(forbidden)) {
			t.Errorf("archive contains forbidden value %q", forbidden)
		}
	}
}

func TestExportExcludesSoftDeletedRows(t *testing.T) {
	env := testutil.SetupTest(t)
	now := time.Now()

	if err := env.DB.GetGorm().Model(&vehicles.Vehicle{}).
		Where("id = ?", testutil.Vehicle1ID).Update("deleted_at", now).Error; err != nil {
		t.Fatalf("soft-deleting vehicle: %v", err)
	}

	deletedRecord := testutil.Maintenances[2]
	if err := env.DB.GetGorm().Model(&maintenance.MaintenanceRecord{}).
		Where("id = ?", deletedRecord.ID).Update("deleted_at", now).Error; err != nil {
		t.Fatalf("soft-deleting record: %v", err)
	}

	entries := exportArchive(t, env, testutil.User1ID)
	requireNoEntriesWithPrefix(t, entries, "vehicles/"+testutil.Vehicle1ID.String()+"/")

	records := decodeEntry[[]export.MaintenanceRecordExport](t, entries,
		"vehicles/"+testutil.Vehicle2ID.String()+"/maintenance_records.json")
	if len(records) != 7 {
		t.Fatalf("expected 7 Vehicle 2 records after the soft delete, got %d", len(records))
	}
	for _, record := range records {
		if record.ID == deletedRecord.ID {
			t.Error("a soft-deleted maintenance record must not be exported")
		}
	}
}

func TestExportRequiresAuthentication(t *testing.T) {
	env := testutil.SetupTest(t)

	w := testutil.RequestWithBearer(t, env.Router, http.MethodGet, "/v1/user/export", "", nil)
	testutil.RequireStatus(t, w, http.StatusUnauthorized)
}

func exportArchive(t *testing.T, env *testutil.Env, userID uuid.UUID) map[string][]byte {
	t.Helper()

	w := testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/user/export", userID, nil)
	testutil.RequireStatus(t, w, http.StatusOK)
	if contentType := w.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/zip") {
		t.Fatalf("expected an application/zip response, got %q", contentType)
	}
	if disposition := w.Header().Get("Content-Disposition"); !strings.HasPrefix(disposition, "attachment;") {
		t.Fatalf("expected an attachment disposition, got %q", disposition)
	}

	reader, err := zip.NewReader(bytes.NewReader(w.Body.Bytes()), int64(w.Body.Len()))
	if err != nil {
		t.Fatalf("reading export zip: %v", err)
	}

	entries := make(map[string][]byte, len(reader.File))
	for _, file := range reader.File {
		body, err := file.Open()
		if err != nil {
			t.Fatalf("opening %s: %v", file.Name, err)
		}
		data, err := io.ReadAll(body)
		if closeErr := body.Close(); closeErr != nil {
			t.Errorf("closing %s: %v", file.Name, closeErr)
		}
		if err != nil {
			t.Fatalf("reading %s: %v", file.Name, err)
		}
		entries[file.Name] = data
	}
	return entries
}

func decodeEntry[T any](t *testing.T, entries map[string][]byte, name string) T {
	t.Helper()

	data, exists := entries[name]
	if !exists {
		t.Fatalf("missing %s; archive entries: %s", name, strings.Join(entryNames(entries), ", "))
	}

	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatalf("decoding %s: %v (body: %s)", name, err, data)
	}
	return value
}

func requireNoEntriesWithPrefix(t *testing.T, entries map[string][]byte, prefix string) {
	t.Helper()

	for name := range entries {
		if strings.HasPrefix(name, prefix) {
			t.Errorf("unexpected archive entry %s", name)
		}
	}
}

func entryNames(entries map[string][]byte) []string {
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	return names
}
