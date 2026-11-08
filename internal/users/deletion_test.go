package users_test

import (
	"auto-myself-api/internal/auth"
	"auto-myself-api/internal/files"
	"auto-myself-api/internal/helpers"
	"auto-myself-api/internal/testutil"
	"auto-myself-api/internal/users"
	"bytes"
	"context"
	"database/sql"
	"net/http"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
)

type devLoginResponse struct {
	Authentication string `json:"authentication"`
	Refresh        string `json:"refresh"`
}

type mobileLoginResponse struct {
	JWT     string `json:"jwt"`
	Refresh string `json:"refresh"`
}

type userRow struct {
	Username  string
	Email     sql.NullString
	Role      string
	Version   int64
	DeletedAt sql.NullTime
}

func readUserRow(t *testing.T, env *testutil.Env, id uuid.UUID) userRow {
	t.Helper()

	var row userRow
	if err := env.DB.GetGorm().Table("users").
		Select("username, email, role, version, deleted_at").
		Where("id = ?", id).
		Scan(&row).Error; err != nil {
		t.Fatalf("reading user %s: %v", id, err)
	}
	return row
}

func countRows(t *testing.T, env *testutil.Env, table, query string, args ...any) int64 {
	t.Helper()

	var count int64
	if err := env.DB.GetGorm().Table(table).Where(query, args...).Count(&count).Error; err != nil {
		t.Fatalf("counting %s: %v", table, err)
	}
	return count
}

func devLogin(t *testing.T, env *testutil.Env, id uuid.UUID) devLoginResponse {
	t.Helper()

	w := testutil.RequestWithBearer(t, env.Router, http.MethodPost, "/auth/development", "",
		testutil.JSONBody(t, map[string]string{"user_id": id.String()}))
	testutil.RequireStatus(t, w, http.StatusOK)

	var response devLoginResponse
	testutil.DecodeJSON(t, w, &response)
	if response.Authentication == "" || response.Refresh == "" {
		t.Fatalf("development login for %s returned an empty token", id)
	}
	return response
}

func mobileLogin(t *testing.T, env *testutil.Env, token string) mobileLoginResponse {
	t.Helper()

	w := testutil.RequestWithBearer(t, env.Router, http.MethodPost, "/auth/mobile/google", "",
		testutil.JSONBody(t, map[string]string{"id_token": token, "nonce": "test-nonce"}))
	testutil.RequireStatus(t, w, http.StatusOK)

	var response mobileLoginResponse
	testutil.DecodeJSON(t, w, &response)
	if response.JWT == "" || response.Refresh == "" {
		t.Fatalf("mobile login returned an empty token")
	}
	return response
}

func currentUserID(t *testing.T, env *testutil.Env, jwt string) uuid.UUID {
	t.Helper()

	w := testutil.RequestWithBearer(t, env.Router, http.MethodGet, "/v1/user", "Bearer "+jwt, nil)
	testutil.RequireStatus(t, w, http.StatusOK)

	var response userResponse
	testutil.DecodeJSON(t, w, &response)
	if response.ID == uuid.Nil {
		t.Fatal("current user response has no ID")
	}
	return response.ID
}

func TestDeleteAccountRejectsOwnedVehicles(t *testing.T) {
	env := testutil.SetupTest(t)

	before := readUserRow(t, env, testutil.User1ID)
	login := devLogin(t, env, testutil.User1ID)
	refreshBefore := countRows(t, env, "refresh_tokens", "user_id = ?", testutil.User1ID)
	if refreshBefore == 0 {
		t.Fatal("expected the development login to create a refresh token")
	}

	w := testutil.RequestWithBearer(t, env.Router, http.MethodDelete, "/v1/user", login.Authentication, nil)
	testutil.RequireStatus(t, w, http.StatusConflict)

	var body struct {
		Error string `json:"error"`
	}
	testutil.DecodeJSON(t, w, &body)
	const want = "Delete all vehicles you own before deleting your account"
	if body.Error != want {
		t.Errorf("expected error %q, got %q", want, body.Error)
	}

	after := readUserRow(t, env, testutil.User1ID)
	if after.DeletedAt.Valid || after.Username != before.Username || after.Role != before.Role || after.Version != before.Version {
		t.Errorf("the rejected delete must not mutate the user: before=%+v after=%+v", before, after)
	}
	if got := countRows(t, env, "refresh_tokens", "user_id = ?", testutil.User1ID); got != refreshBefore {
		t.Errorf("refresh tokens changed after a rejected delete: got %d, want %d", got, refreshBefore)
	}
}

func TestDeleteAccountOwnershipFollowsOwnedBy(t *testing.T) {
	env := testutil.SetupTest(t)

	if err := env.DB.GetGorm().Table("vehicles").
		Where("id = ?", testutil.Vehicle4ID).Update("owned_by", testutil.User5ID).Error; err != nil {
		t.Fatalf("transferring vehicle ownership: %v", err)
	}

	login := devLogin(t, env, testutil.User5ID)
	w := testutil.RequestWithBearer(t, env.Router, http.MethodDelete, "/v1/user", login.Authentication, nil)
	testutil.RequireStatus(t, w, http.StatusConflict)

	login = devLogin(t, env, testutil.User6ID)
	w = testutil.RequestWithBearer(t, env.Router, http.MethodDelete, "/v1/user", login.Authentication, nil)
	testutil.RequireStatus(t, w, http.StatusNoContent)
}

func TestDeleteAccountAnonymizesAndHardDeletesAuthArtifacts(t *testing.T) {
	env := testutil.SetupTest(t)

	env.GoogleAuth.Register("google-delete-token", auth.VerifiedIdentity{
		Subject: "google-delete-subject",
		Email:   "delete@example.com",
		Name:    "Delete Me",
	})

	first := mobileLogin(t, env, "google-delete-token")
	firstID := currentUserID(t, env, first.JWT)

	// Simulate profile data that deletion must scrub; users.email is not mapped on the Go model.
	if err := env.DB.GetGorm().Table("users").Where("id = ?", firstID).
		Updates(map[string]any{"email": "legacy@example.com", "role": "moderator"}).Error; err != nil {
		t.Fatalf("seeding legacy profile fields: %v", err)
	}

	w := testutil.RequestWithBearer(t, env.Router, http.MethodDelete, "/v1/user", "Bearer "+first.JWT, nil)
	testutil.RequireStatus(t, w, http.StatusNoContent)

	row := readUserRow(t, env, firstID)
	if !row.DeletedAt.Valid {
		t.Error("expected deleted_at to be set")
	}
	if row.Username != users.DeletedUsername {
		t.Errorf("expected username %q, got %q", users.DeletedUsername, row.Username)
	}
	if row.Email.Valid {
		t.Errorf("expected email to be cleared, got %q", row.Email.String)
	}
	if row.Role != users.RoleUser {
		t.Errorf("expected role %q, got %q", users.RoleUser, row.Role)
	}
	if row.Version != 2 {
		t.Errorf("expected version 2 after deletion, got %d", row.Version)
	}

	if got := countRows(t, env, "identities", "user_id = ?", firstID); got != 0 {
		t.Errorf("expected identities to be hard-deleted, found %d", got)
	}
	if got := countRows(t, env, "refresh_tokens", "user_id = ?", firstID); got != 0 {
		t.Errorf("expected refresh tokens to be hard-deleted, found %d", got)
	}

	w = testutil.RequestWithBearer(t, env.Router, http.MethodGet, "/v1/user", "Bearer "+first.JWT, nil)
	testutil.RequireStatus(t, w, http.StatusUnauthorized)

	w = testutil.RequestWithBearer(t, env.Router, http.MethodPost, "/auth/refresh", "",
		testutil.JSONBody(t, map[string]string{"refresh_token": first.Refresh}))
	testutil.RequireStatus(t, w, http.StatusUnauthorized)

	w = testutil.RequestWithBearer(t, env.Router, http.MethodPost, "/auth/development", "",
		testutil.JSONBody(t, map[string]string{"user_id": firstID.String()}))
	testutil.RequireStatus(t, w, http.StatusNotFound)

	// The identity was hard-deleted, so the same provider subject registers a new account.
	second := mobileLogin(t, env, "google-delete-token")
	secondID := currentUserID(t, env, second.JWT)
	if secondID == firstID {
		t.Fatal("re-registration must create a new user ID")
	}
}

func TestDeleteAccountRetainsAccessAndTombstone(t *testing.T) {
	env := testutil.SetupTest(t)

	login := devLogin(t, env, testutil.User7ID)

	activeAccessBefore := countRows(t, env, "vehicle_user_access", "user_id = ? AND deleted_at IS NULL", testutil.User7ID)
	if activeAccessBefore != 2 {
		t.Fatalf("fixture assumption broken: User 7 has %d active access rows, want 2", activeAccessBefore)
	}
	activeRecordsBefore := countRows(t, env, "maintenance_records", "created_by = ? AND deleted_at IS NULL", testutil.User7ID)
	if activeRecordsBefore == 0 {
		t.Fatal("fixture assumption broken: User 7 has no active maintenance records")
	}

	w := testutil.RequestWithBearer(t, env.Router, http.MethodDelete, "/v1/user", login.Authentication, nil)
	testutil.RequireStatus(t, w, http.StatusNoContent)

	row := readUserRow(t, env, testutil.User7ID)
	if !row.DeletedAt.Valid || row.Username != users.DeletedUsername {
		t.Errorf("expected an anonymized tombstone, got %+v", row)
	}

	if got := countRows(t, env, "vehicle_user_access", "user_id = ? AND deleted_at IS NULL", testutil.User7ID); got != activeAccessBefore {
		t.Errorf("access rows must be retained: got %d, want %d", got, activeAccessBefore)
	}
	if got := countRows(t, env, "maintenance_records", "created_by = ? AND deleted_at IS NULL", testutil.User7ID); got != activeRecordsBefore {
		t.Errorf("authored records must be retained: got %d, want %d", got, activeRecordsBefore)
	}

	w = testutil.RequestWithBearer(t, env.Router, http.MethodGet, "/v1/user", login.Authentication, nil)
	testutil.RequireStatus(t, w, http.StatusUnauthorized)

	// A deleted account cannot authenticate, so even a repeated DELETE is rejected before the handler.
	w = testutil.RequestWithBearer(t, env.Router, http.MethodDelete, "/v1/user", login.Authentication, nil)
	testutil.RequireStatus(t, w, http.StatusUnauthorized)

	w = testutil.RequestWithBearer(t, env.Router, http.MethodPost, "/auth/refresh", "",
		testutil.JSONBody(t, map[string]string{"refresh_token": login.Refresh}))
	testutil.RequireStatus(t, w, http.StatusUnauthorized)
	if got := countRows(t, env, "refresh_tokens", "user_id = ?", testutil.User7ID); got != 0 {
		t.Errorf("expected refresh tokens to be hard-deleted, found %d", got)
	}

	// User 1 owns Vehicle 2, where User 7 had write access, so the tombstone stays readable.
	w = testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/user/"+testutil.User7ID.String(), testutil.User1ID, nil)
	testutil.RequireStatus(t, w, http.StatusOK)

	var tombstone userResponse
	testutil.DecodeJSON(t, w, &tombstone)
	if tombstone.ID != testutil.User7ID || tombstone.Username != users.DeletedUsername {
		t.Errorf("unexpected tombstone response: %+v", tombstone)
	}

	// User 4 has no shared vehicle with User 7 and still gets 404.
	w = testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/user/"+testutil.User7ID.String(), testutil.User4ID, nil)
	testutil.RequireStatus(t, w, http.StatusNotFound)

	// The retained access row is still listed for the vehicle owner.
	w = testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/vehicle/"+testutil.Vehicle2ID.String()+"/access", testutil.User1ID, nil)
	testutil.RequireStatus(t, w, http.StatusOK)

	var accessRows []struct {
		UserID uuid.UUID `json:"user_id"`
	}
	testutil.DecodeJSON(t, w, &accessRows)
	found := false
	for _, accessRow := range accessRows {
		if accessRow.UserID == testutil.User7ID {
			found = true
		}
	}
	if !found {
		t.Error("expected the retained access row to still be listed for the vehicle owner")
	}

	// The owner can still read a record the deleted user authored.
	record := testutil.Maintenances[6]
	if record.CreatedBy != testutil.User7ID {
		t.Fatalf("fixture assumption broken: record %s is not User 7's", record.ID)
	}
	w = testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/maintenance/"+record.ID.String(), testutil.User1ID, nil)
	testutil.RequireStatus(t, w, http.StatusOK)

	// A duplicate delete that reaches the service layer is idempotent.
	repo := users.NewUserRepository(env.DB)
	service := users.NewUserService(repo)
	serviceCtx := context.WithValue(context.Background(), helpers.UserCtxKey, &users.User{UserWeb: users.UserWeb{ID: testutil.User7ID}})
	if err := service.DeleteAccount(serviceCtx); err != nil {
		t.Fatalf("a duplicate delete at the service layer must be idempotent: %v", err)
	}
}

func TestDeleteAccountAllowsSoftDeletedOwnedVehicles(t *testing.T) {
	env := testutil.SetupTest(t)

	w := testutil.RequestAsUser(t, env.Router, http.MethodDelete, "/v1/vehicle/"+testutil.Vehicle3ID.String(), testutil.User4ID, nil)
	testutil.RequireStatus(t, w, http.StatusNoContent)

	w = testutil.RequestAsUser(t, env.Router, http.MethodDelete, "/v1/user", testutil.User4ID, nil)
	testutil.RequireStatus(t, w, http.StatusNoContent)

	row := readUserRow(t, env, testutil.User4ID)
	if !row.DeletedAt.Valid || row.Username != users.DeletedUsername {
		t.Errorf("expected User 4 to be tombstoned, got %+v", row)
	}

	var vehicle struct {
		CreatedBy uuid.UUID
	}
	if err := env.DB.GetGorm().Table("vehicles").Select("created_by").
		Where("id = ?", testutil.Vehicle3ID).Scan(&vehicle).Error; err != nil {
		t.Fatalf("reading soft-deleted vehicle: %v", err)
	}
	if vehicle.CreatedBy != testutil.User4ID {
		t.Errorf("expected the soft-deleted vehicle to keep its creator, got %s", vehicle.CreatedBy)
	}
}

func TestDeleteAccountRetainsPendingShares(t *testing.T) {
	env := testutil.SetupTest(t)

	login := devLogin(t, env, testutil.User5ID)
	w := testutil.RequestWithBearer(t, env.Router, http.MethodDelete, "/v1/user", login.Authentication, nil)
	testutil.RequireStatus(t, w, http.StatusNoContent)

	if got := countRows(t, env, "vehicle_user_access_pending", "user_id = ? AND deleted_at IS NULL", testutil.User5ID); got != 1 {
		t.Errorf("expected the pending share to be retained, found %d active rows", got)
	}

	// Pending shares do not grant CanRead visibility, so the inviter still gets 404.
	w = testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/user/"+testutil.User5ID.String(), testutil.User6ID, nil)
	testutil.RequireStatus(t, w, http.StatusNotFound)
}

func TestDeleteAccountRetainsAuthoredReportsAndFiles(t *testing.T) {
	env := testutil.SetupTest(t)

	record := testutil.Maintenances[2] // Vehicle 2: User 7 has write access.
	upload := testutil.RequestAsUser(t, env.Router, http.MethodPost,
		"/v1/file/maintenance/"+record.ID.String(), testutil.User7ID, bytes.NewReader([]byte("retained file")))
	testutil.RequireStatus(t, upload, http.StatusCreated)

	var link struct {
		FileID uuid.UUID `json:"file_id"`
	}
	testutil.DecodeJSON(t, upload, &link)
	if link.FileID == uuid.Nil {
		t.Fatal("upload response has no file_id")
	}

	report := testutil.RequestAsUser(t, env.Router, http.MethodPost, "/v1/report", testutil.User7ID,
		testutil.JSONBody(t, map[string]any{
			"reported_topic": "vehicle",
			"reported_id":    testutil.Vehicle1ID,
			"reason":         "retention test",
		}))
	testutil.RequireStatus(t, report, http.StatusCreated)

	var created struct {
		ID uuid.UUID `json:"ID"`
	}
	testutil.DecodeJSON(t, report, &created)
	if created.ID == uuid.Nil {
		t.Fatal("report response has no ID")
	}

	w := testutil.RequestAsUser(t, env.Router, http.MethodDelete, "/v1/user", testutil.User7ID, nil)
	testutil.RequireStatus(t, w, http.StatusNoContent)

	var fileRow struct {
		CreatedBy uuid.UUID
	}
	if err := env.DB.GetGorm().Table("files").Select("created_by").
		Where("id = ? AND deleted_at IS NULL", link.FileID).Scan(&fileRow).Error; err != nil {
		t.Fatalf("reading retained file: %v", err)
	}
	if fileRow.CreatedBy != testutil.User7ID {
		t.Errorf("expected the file to remain authored by User 7, got %s", fileRow.CreatedBy)
	}
	if _, exists := env.Storage.Get(files.StorageKeyPrefix + link.FileID.String()); !exists {
		t.Error("expected the storage object to remain")
	}

	var reportRow struct {
		CreatedBy uuid.UUID
	}
	if err := env.DB.GetGorm().Table("reports").Select("created_by").
		Where("id = ? AND deleted_at IS NULL", created.ID).Scan(&reportRow).Error; err != nil {
		t.Fatalf("reading retained report: %v", err)
	}
	if reportRow.CreatedBy != testutil.User7ID {
		t.Errorf("expected the report to remain authored by User 7, got %s", reportRow.CreatedBy)
	}

	// The vehicle owner can still list the retained file.
	w = testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/file/maintenance/"+record.ID.String(), testutil.User1ID, nil)
	testutil.RequireStatus(t, w, http.StatusOK)

	var entries []struct {
		ID uuid.UUID `json:"id"`
	}
	testutil.DecodeJSON(t, w, &entries)
	found := false
	for _, entry := range entries {
		if entry.ID == link.FileID {
			found = true
		}
	}
	if !found {
		t.Error("expected the retained file to be listed for the vehicle owner")
	}
}

func TestGetUserTombstoneOverridesLegacyUsername(t *testing.T) {
	env := testutil.SetupTest(t)

	// Simulate a legacy soft-deleted row whose stored username was never anonymized.
	if err := env.DB.GetGorm().Table("users").Where("id = ?", testutil.User5ID).
		Updates(map[string]any{"username": "legacy name", "deleted_at": time.Now()}).Error; err != nil {
		t.Fatalf("soft-deleting legacy user: %v", err)
	}

	accessID, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("generating access id: %v", err)
	}
	if err := env.DB.GetGorm().Exec(
		"INSERT INTO vehicle_user_access (id, user_id, vehicle_id, write_access, created_by) VALUES (?, ?, ?, false, ?)",
		accessID, testutil.User5ID, testutil.Vehicle2ID, testutil.User1ID).Error; err != nil {
		t.Fatalf("inserting access row: %v", err)
	}

	w := testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/user/"+testutil.User5ID.String(), testutil.User1ID, nil)
	testutil.RequireStatus(t, w, http.StatusOK)

	var response userResponse
	testutil.DecodeJSON(t, w, &response)
	if response.Username != users.DeletedUsername {
		t.Errorf("expected the legacy username to be hidden, got %q", response.Username)
	}
}

func TestDeleteAccountUnauthenticated(t *testing.T) {
	env := testutil.SetupTest(t)

	w := testutil.RequestWithBearer(t, env.Router, http.MethodDelete, "/v1/user", "", nil)
	testutil.RequireStatus(t, w, http.StatusUnauthorized)
}
