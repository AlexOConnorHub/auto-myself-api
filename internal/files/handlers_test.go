package files_test

import (
	"auto-myself-api/internal/files"
	"auto-myself-api/internal/testutil"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
)

type fileEntry struct {
	ID         uuid.UUID `json:"id"`
	CreatedAt  string    `json:"created_at"`
	UpdatedAt  string    `json:"updated_at"`
	Sha256Hash string    `json:"sha256_hash"`
	FileSize   int64     `json:"file_size"`
	SignedURL  string    `json:"signed_url"`
}

type fileLink struct {
	ID                  uuid.UUID `json:"id"`
	FileID              uuid.UUID `json:"file_id"`
	MaintenanceRecordID uuid.UUID `json:"maintenance_record_id"`
}

func uploadFile(t *testing.T, env *testutil.Env, userID, maintenanceID uuid.UUID, data []byte) *fileLink {
	t.Helper()

	w := testutil.RequestAsUser(t, env.Router, http.MethodPost,
		"/v1/file/maintenance/"+maintenanceID.String(), userID, bytes.NewReader(data))
	testutil.RequireStatus(t, w, http.StatusCreated)

	var link fileLink
	testutil.DecodeJSON(t, w, &link)
	return &link
}

func TestFileUploadAndList(t *testing.T) {
	env := testutil.SetupTest(t)

	fixture := testutil.Maintenances[0]
	owner := testutil.User1ID
	data := []byte(`{"isFile": true}`)

	link := uploadFile(t, env, owner, fixture.ID, data)
	if link.MaintenanceRecordID != fixture.ID {
		t.Errorf("expected maintenance_record_id %s, got %s", fixture.ID, link.MaintenanceRecordID)
	}

	storedKey := files.StorageKeyPrefix + link.FileID.String()
	stored, exists := env.Storage.Get(storedKey)
	if !exists {
		t.Fatalf("expected uploaded object at %s", storedKey)
	}
	if !bytes.Equal(stored, data) {
		t.Errorf("stored bytes do not match upload")
	}

	w := testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/file/maintenance/"+fixture.ID.String(), owner, nil)
	testutil.RequireStatus(t, w, http.StatusOK)

	if strings.Contains(w.Body.String(), "storage_key") {
		t.Errorf("file list must not expose storage_key: %s", w.Body.String())
	}

	var entries []fileEntry
	testutil.DecodeJSON(t, w, &entries)
	if len(entries) != 1 {
		t.Fatalf("expected 1 file, got %d", len(entries))
	}

	entry := entries[0]
	if entry.ID != link.FileID {
		t.Errorf("expected file id %s, got %s", link.FileID, entry.ID)
	}

	hash := sha256.Sum256(data)
	if entry.Sha256Hash != hex.EncodeToString(hash[:]) {
		t.Errorf("unexpected sha256 hash %q", entry.Sha256Hash)
	}
	if entry.FileSize != int64(len(data)) {
		t.Errorf("expected file_size %d, got %d", len(data), entry.FileSize)
	}
	if entry.SignedURL == "" {
		t.Error("expected a signed URL")
	}
	if _, err := time.Parse(time.RFC3339, entry.CreatedAt); err != nil {
		t.Errorf("created_at is not RFC3339: %q", entry.CreatedAt)
	}
	if _, err := time.Parse(time.RFC3339, entry.UpdatedAt); err != nil {
		t.Errorf("updated_at is not RFC3339: %q", entry.UpdatedAt)
	}
}

func TestFilePermissions(t *testing.T) {
	env := testutil.SetupTest(t)

	// Maintenance record 2 belongs to Vehicle 2: User 3 is read-only, User 5 has no access.
	readOnlyRecord := testutil.Maintenances[2]
	if readOnlyRecord.VehicleID != testutil.Vehicle2ID {
		t.Fatalf("fixture assumption broken: record is on %s", readOnlyRecord.VehicleID)
	}

	// A writable record for setup.
	uploadFile(t, env, testutil.User1ID, readOnlyRecord.ID, []byte("owned"))

	tests := []struct {
		name       string
		user       uuid.UUID
		wantList   int
		wantUpload int
	}{
		{"read-only recipient", testutil.User3ID, http.StatusOK, http.StatusForbidden},
		{"no access", testutil.User5ID, http.StatusNotFound, http.StatusNotFound},
	}

	for _, test := range tests {
		t.Run("list "+test.name, func(t *testing.T) {
			w := testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/file/maintenance/"+readOnlyRecord.ID.String(), test.user, nil)
			testutil.RequireStatus(t, w, test.wantList)
		})
		t.Run("upload "+test.name, func(t *testing.T) {
			w := testutil.RequestAsUser(t, env.Router, http.MethodPost,
				"/v1/file/maintenance/"+readOnlyRecord.ID.String(), test.user, bytes.NewReader([]byte("x")))
			testutil.RequireStatus(t, w, test.wantUpload)
		})
	}

	// Pending recipients cannot see record files at all.
	pendingRecord := testutil.Maintenances[12] // Vehicle 4, User 4 has a pending share.
	if pendingRecord.VehicleID != testutil.Vehicle4ID {
		t.Fatalf("fixture assumption broken: record is on %s", pendingRecord.VehicleID)
	}
	w := testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/file/maintenance/"+pendingRecord.ID.String(), testutil.User4ID, nil)
	testutil.RequireStatus(t, w, http.StatusNotFound)

	w = testutil.RequestAsUser(t, env.Router, http.MethodPost, "/v1/file/maintenance/"+pendingRecord.ID.String(), testutil.User4ID, bytes.NewReader([]byte("x")))
	testutil.RequireStatus(t, w, http.StatusNotFound)
}

func TestFileDelete(t *testing.T) {
	env := testutil.SetupTest(t)

	fixture := testutil.Maintenances[0]
	owner := testutil.User1ID
	link := uploadFile(t, env, owner, fixture.ID, []byte("to be deleted"))

	w := testutil.RequestAsUser(t, env.Router, http.MethodDelete, "/v1/file/"+link.FileID.String(), owner, nil)
	testutil.RequireStatus(t, w, http.StatusNoContent)

	if env.Storage.Len() != 0 {
		t.Errorf("expected storage object to be deleted, %d remain", env.Storage.Len())
	}

	w = testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/file/maintenance/"+fixture.ID.String(), owner, nil)
	testutil.RequireStatus(t, w, http.StatusOK)

	var entries []fileEntry
	testutil.DecodeJSON(t, w, &entries)
	if len(entries) != 0 {
		t.Errorf("expected no files after delete, got %d", len(entries))
	}

	w = testutil.RequestAsUser(t, env.Router, http.MethodDelete, "/v1/file/"+link.FileID.String(), owner, nil)
	testutil.RequireStatus(t, w, http.StatusNotFound)
}

func TestFileDeletePermissions(t *testing.T) {
	env := testutil.SetupTest(t)

	fixture := testutil.Maintenances[2] // Vehicle 2
	link := uploadFile(t, env, testutil.User1ID, fixture.ID, []byte("protected"))

	w := testutil.RequestAsUser(t, env.Router, http.MethodDelete, "/v1/file/"+link.FileID.String(), testutil.User3ID, nil)
	testutil.RequireStatus(t, w, http.StatusForbidden)

	w = testutil.RequestAsUser(t, env.Router, http.MethodDelete, "/v1/file/"+link.FileID.String(), testutil.User5ID, nil)
	testutil.RequireStatus(t, w, http.StatusNotFound)
}

func TestFileUnsupportedRecordType(t *testing.T) {
	env := testutil.SetupTest(t)

	id := testutil.Maintenances[0].ID.String()

	w := testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/file/vehicle/"+id, testutil.User1ID, nil)
	testutil.RequireStatus(t, w, http.StatusBadRequest)

	w = testutil.RequestAsUser(t, env.Router, http.MethodPost, "/v1/file/vehicle/"+id, testutil.User1ID, bytes.NewReader([]byte("x")))
	testutil.RequireStatus(t, w, http.StatusBadRequest)
}

func TestFileUploadTooLarge(t *testing.T) {
	env := testutil.SetupTest(t)

	data := make([]byte, files.MaxUploadBytes+1)
	w := testutil.RequestAsUser(t, env.Router, http.MethodPost,
		"/v1/file/maintenance/"+testutil.Maintenances[0].ID.String(), testutil.User1ID, bytes.NewReader(data))
	testutil.RequireStatus(t, w, http.StatusRequestEntityTooLarge)

	if env.Storage.Len() != 0 {
		t.Errorf("expected no stored objects after a rejected upload")
	}
}

func TestFileUploadStorageFailure(t *testing.T) {
	env := testutil.SetupTest(t)
	env.Storage.FailUpload = true

	w := testutil.RequestAsUser(t, env.Router, http.MethodPost,
		"/v1/file/maintenance/"+testutil.Maintenances[0].ID.String(), testutil.User1ID, bytes.NewReader([]byte("data")))
	testutil.RequireStatus(t, w, http.StatusInternalServerError)

	env.Storage.FailUpload = false
	w = testutil.RequestAsUser(t, env.Router, http.MethodGet,
		"/v1/file/maintenance/"+testutil.Maintenances[0].ID.String(), testutil.User1ID, nil)
	testutil.RequireStatus(t, w, http.StatusOK)

	var entries []fileEntry
	testutil.DecodeJSON(t, w, &entries)
	if len(entries) != 0 {
		t.Errorf("expected no file rows after failed upload, got %d", len(entries))
	}
}

func TestFileDeleteLeavesOrphanWhenStorageFails(t *testing.T) {
	env := testutil.SetupTest(t)

	fixture := testutil.Maintenances[0]
	link := uploadFile(t, env, testutil.User1ID, fixture.ID, []byte("orphan"))

	env.Storage.FailDelete = true
	w := testutil.RequestAsUser(t, env.Router, http.MethodDelete, "/v1/file/"+link.FileID.String(), testutil.User1ID, nil)
	testutil.RequireStatus(t, w, http.StatusNoContent)

	// The DB rows are gone even though the object could not be removed.
	w = testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/file/maintenance/"+fixture.ID.String(), testutil.User1ID, nil)
	testutil.RequireStatus(t, w, http.StatusOK)

	var entries []fileEntry
	testutil.DecodeJSON(t, w, &entries)
	if len(entries) != 0 {
		t.Errorf("expected no files after delete, got %d", len(entries))
	}
}
