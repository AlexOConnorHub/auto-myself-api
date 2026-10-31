package reports_test

import (
	"auto-myself-api/internal/testutil"
	"net/http"
	"strings"
	"testing"

	"github.com/gofrs/uuid/v5"
)

type reportResponse struct {
	ID            uuid.UUID `json:"ID"`
	ReportedTable string    `json:"reported_topic"`
	ReportedID    uuid.UUID `json:"reported_id"`
	Reason        string    `json:"reason"`
	CreatedBy     uuid.UUID `json:"created_by"`
	PublicNotes   string    `json:"public_notes"`
	Resolved      bool      `json:"resolved"`
	ResolvedAt    *string   `json:"resolved_at"`
	Version       int64     `json:"version"`
}

func createReport(t *testing.T, env *testutil.Env, userID uuid.UUID) reportResponse {
	t.Helper()

	body := map[string]any{
		"reported_topic": "users",
		"reported_id":    testutil.User2ID,
		"reason":         "Test reason",
	}

	w := testutil.RequestAsUser(t, env.Router, http.MethodPost, "/v1/report", userID, testutil.JSONBody(t, body))
	testutil.RequireStatus(t, w, http.StatusCreated)

	var response reportResponse
	testutil.DecodeJSON(t, w, &response)
	if response.Version != 1 {
		t.Fatalf("expected a new report at version 1, got %d", response.Version)
	}
	return response
}

func TestReportCreate(t *testing.T) {
	env := testutil.SetupTest(t)

	body := map[string]any{
		"reported_topic": "users",
		"reported_id":    testutil.User2ID,
		"reason":         "Test reason",
		"version":        99,
	}

	w := testutil.RequestAsUser(t, env.Router, http.MethodPost, "/v1/report", testutil.User5ID, testutil.JSONBody(t, body))
	testutil.RequireStatus(t, w, http.StatusCreated)

	if strings.Contains(w.Body.String(), "internal_notes") {
		t.Errorf("create response leaked internal notes: %s", w.Body.String())
	}

	var response reportResponse
	testutil.DecodeJSON(t, w, &response)
	if response.ID.IsNil() {
		t.Error("expected a report ID")
	}
	if response.ReportedTable != "users" {
		t.Errorf("expected reported_topic \"users\", got %q", response.ReportedTable)
	}
	if response.ReportedID != testutil.User2ID {
		t.Errorf("expected reported_id %s, got %s", testutil.User2ID, response.ReportedID)
	}
	if response.CreatedBy != testutil.User5ID {
		t.Errorf("expected created_by %s, got %s", testutil.User5ID, response.CreatedBy)
	}
	if response.Version != 1 {
		t.Errorf("create should ignore a supplied version and start at 1, got %d", response.Version)
	}
}

func TestReportCreateBadBody(t *testing.T) {
	env := testutil.SetupTest(t)

	w := testutil.RequestAsUser(t, env.Router, http.MethodPost, "/v1/report", testutil.User5ID, strings.NewReader("{invalid"))
	testutil.RequireStatus(t, w, http.StatusUnprocessableEntity)
}

func TestReportList(t *testing.T) {
	env := testutil.SetupTest(t)

	user5Report := createReport(t, env, testutil.User5ID)
	user4Report := createReport(t, env, testutil.User4ID)
	moderatorReport := createReport(t, env, testutil.User2ID)
	adminReport := createReport(t, env, testutil.User1ID)

	list := func(userID uuid.UUID) []uuid.UUID {
		t.Helper()
		w := testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/report", userID, nil)
		testutil.RequireStatus(t, w, http.StatusOK)
		var ids []uuid.UUID
		testutil.DecodeJSON(t, w, &ids)
		return ids
	}

	// Every role, including moderators and admins, sees only their own submissions.
	tests := []struct {
		name string
		user uuid.UUID
		want uuid.UUID
	}{
		{"regular user", testutil.User5ID, user5Report.ID},
		{"another regular user", testutil.User4ID, user4Report.ID},
		{"moderator", testutil.User2ID, moderatorReport.ID},
		{"admin", testutil.User1ID, adminReport.ID},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ids := list(test.user)
			if len(ids) != 1 || ids[0] != test.want {
				t.Errorf("expected only report %s, got %v", test.want, ids)
			}
		})
	}
}

func TestReportListByDate(t *testing.T) {
	env := testutil.SetupTest(t)

	report := createReport(t, env, testutil.User5ID)

	// Regular users may not use the by-date endpoint.
	w := testutil.RequestAsUser(t, env.Router, http.MethodPost, "/v1/report/by-date", testutil.User5ID, testutil.JSONBody(t, map[string]any{}))
	testutil.RequireStatus(t, w, http.StatusForbidden)

	// Default range covers the report we just created.
	w = testutil.RequestAsUser(t, env.Router, http.MethodPost, "/v1/report/by-date", testutil.User2ID, testutil.JSONBody(t, map[string]any{}))
	testutil.RequireStatus(t, w, http.StatusOK)

	var ids []uuid.UUID
	testutil.DecodeJSON(t, w, &ids)
	found := false
	for _, id := range ids {
		if id == report.ID {
			found = true
		}
	}
	if !found {
		t.Errorf("expected default date range to include %s, got %v", report.ID, ids)
	}

	// A past range excludes it.
	rangeBody := map[string]any{
		"start_date": "2020-01-01T00:00:00Z",
		"end_date":   "2020-12-31T23:59:59Z",
	}
	w = testutil.RequestAsUser(t, env.Router, http.MethodPost, "/v1/report/by-date", testutil.User1ID, testutil.JSONBody(t, rangeBody))
	testutil.RequireStatus(t, w, http.StatusOK)

	testutil.DecodeJSON(t, w, &ids)
	for _, id := range ids {
		if id == report.ID {
			t.Errorf("expected past range to exclude %s", report.ID)
		}
	}

	// Malformed dates are rejected.
	w = testutil.RequestAsUser(t, env.Router, http.MethodPost, "/v1/report/by-date", testutil.User2ID,
		testutil.JSONBody(t, map[string]any{"start_date": "not-a-date"}))
	testutil.RequireStatus(t, w, http.StatusUnprocessableEntity)
}

func TestReportFindByIDPermissions(t *testing.T) {
	env := testutil.SetupTest(t)

	report := createReport(t, env, testutil.User5ID)

	tests := []struct {
		name string
		user uuid.UUID
		want int
	}{
		{"owner", testutil.User5ID, http.StatusOK},
		{"moderator", testutil.User2ID, http.StatusOK},
		{"admin", testutil.User1ID, http.StatusOK},
		{"other user", testutil.User4ID, http.StatusNotFound},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w := testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/report/"+report.ID.String(), test.user, nil)
			testutil.RequireStatus(t, w, test.want)
		})
	}
}

func TestReportFindByIDHidesInternalNotes(t *testing.T) {
	env := testutil.SetupTest(t)

	report := createReport(t, env, testutil.User5ID)

	w := testutil.RequestAsUser(t, env.Router, http.MethodPatch, "/v1/report/"+report.ID.String(), testutil.User2ID,
		testutil.JSONBody(t, map[string]any{"resolved": true, "public_notes": "handled", "internal_notes": "secret note", "version": 1}))
	testutil.RequireStatus(t, w, http.StatusOK)

	for _, userID := range []uuid.UUID{testutil.User5ID, testutil.User2ID, testutil.User1ID} {
		w := testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/report/"+report.ID.String(), userID, nil)
		testutil.RequireStatus(t, w, http.StatusOK)

		if strings.Contains(w.Body.String(), "internal_notes") || strings.Contains(w.Body.String(), "secret note") {
			t.Errorf("report response leaked internal notes: %s", w.Body.String())
		}
	}
}

func TestReportUpdatePermissions(t *testing.T) {
	env := testutil.SetupTest(t)

	report := createReport(t, env, testutil.User5ID)
	path := "/v1/report/" + report.ID.String()

	// A regular user who owns the report may read it, but only moderators may resolve it.
	w := testutil.RequestAsUser(t, env.Router, http.MethodPatch, path, testutil.User5ID,
		testutil.JSONBody(t, map[string]any{"resolved": true, "version": 1}))
	testutil.RequireStatus(t, w, http.StatusForbidden)

	// A regular user who cannot read it gets 404.
	w = testutil.RequestAsUser(t, env.Router, http.MethodPatch, path, testutil.User4ID,
		testutil.JSONBody(t, map[string]any{"resolved": true, "version": 1}))
	testutil.RequireStatus(t, w, http.StatusNotFound)

	// The moderator resolves it.
	w = testutil.RequestAsUser(t, env.Router, http.MethodPatch, path, testutil.User2ID,
		testutil.JSONBody(t, map[string]any{"resolved": true, "public_notes": "done", "internal_notes": "secret", "version": 1}))
	testutil.RequireStatus(t, w, http.StatusOK)

	var resolved reportResponse
	testutil.DecodeJSON(t, w, &resolved)
	if !resolved.Resolved {
		t.Error("expected the report to be resolved")
	}
	if resolved.ResolvedAt == nil {
		t.Error("expected resolved_at to be set")
	}
	if resolved.PublicNotes != "done" {
		t.Errorf("expected public_notes %q, got %q", "done", resolved.PublicNotes)
	}

	// Once resolved, no further updates are allowed — not even by the resolver or an admin.
	w = testutil.RequestAsUser(t, env.Router, http.MethodPatch, path, testutil.User2ID,
		testutil.JSONBody(t, map[string]any{"internal_notes": "updated by resolver", "version": 1}))
	testutil.RequireStatus(t, w, http.StatusForbidden)

	w = testutil.RequestAsUser(t, env.Router, http.MethodPatch, path, testutil.User1ID,
		testutil.JSONBody(t, map[string]any{"internal_notes": "updated by admin", "version": 1}))
	testutil.RequireStatus(t, w, http.StatusForbidden)

	var internalNotes string
	if err := env.DB.GetGorm().Table("reports").Select("internal_notes").Where("id = ?", report.ID).Row().Scan(&internalNotes); err != nil {
		t.Fatalf("reading internal notes: %v", err)
	}
	if internalNotes != "secret" {
		t.Errorf("expected notes from the resolution to be unchanged, got %q", internalNotes)
	}
}

func TestReportPatchVersionConflict(t *testing.T) {
	env := testutil.SetupTest(t)

	report := createReport(t, env, testutil.User5ID)
	path := "/v1/report/" + report.ID.String()

	w := testutil.RequestAsUser(t, env.Router, http.MethodPatch, path, testutil.User2ID,
		testutil.JSONBody(t, map[string]any{"public_notes": "first", "version": 1}))
	testutil.RequireStatus(t, w, http.StatusOK)

	var first reportResponse
	testutil.DecodeJSON(t, w, &first)
	if first.Version != 2 {
		t.Fatalf("expected version 2 after the first update, got %d", first.Version)
	}

	w = testutil.RequestAsUser(t, env.Router, http.MethodPatch, path, testutil.User2ID,
		testutil.JSONBody(t, map[string]any{"public_notes": "second", "version": 1}))
	testutil.RequireStatus(t, w, http.StatusConflict)

	var conflict struct {
		Error   string `json:"error"`
		Version int64  `json:"version"`
	}
	testutil.DecodeJSON(t, w, &conflict)
	if conflict.Error != "Version conflict" || conflict.Version != 2 {
		t.Errorf("expected a version conflict at version 2, got %+v", conflict)
	}

	w = testutil.RequestAsUser(t, env.Router, http.MethodGet, path, testutil.User5ID, nil)
	testutil.RequireStatus(t, w, http.StatusOK)

	var stored reportResponse
	testutil.DecodeJSON(t, w, &stored)
	if stored.PublicNotes != "first" || stored.Version != 2 {
		t.Errorf("the stale update must not be applied: %+v", stored)
	}
}

func TestReportPatchRequiresVersion(t *testing.T) {
	env := testutil.SetupTest(t)

	report := createReport(t, env, testutil.User5ID)
	path := "/v1/report/" + report.ID.String()

	bodies := map[string]map[string]any{
		"missing":  {"public_notes": "missing version"},
		"zero":     {"public_notes": "zero version", "version": 0},
		"negative": {"public_notes": "negative version", "version": -1},
	}

	for name, body := range bodies {
		t.Run(name, func(t *testing.T) {
			w := testutil.RequestAsUser(t, env.Router, http.MethodPatch, path, testutil.User2ID, testutil.JSONBody(t, body))
			testutil.RequireStatus(t, w, http.StatusUnprocessableEntity)
		})
	}
}

func TestReportPatchIgnoresProtectedFields(t *testing.T) {
	env := testutil.SetupTest(t)

	report := createReport(t, env, testutil.User5ID)

	w := testutil.RequestAsUser(t, env.Router, http.MethodPatch, "/v1/report/"+report.ID.String(), testutil.User2ID,
		testutil.JSONBody(t, map[string]any{
			"public_notes": "renamed",
			"version":      1,
			"ID":           testutil.User1ID,
			"created_by":   testutil.User2ID,
			"resolved_by":  testutil.User1ID,
		}))
	testutil.RequireStatus(t, w, http.StatusOK)

	var response reportResponse
	testutil.DecodeJSON(t, w, &response)
	if response.ID != report.ID {
		t.Errorf("PATCH must not change the ID: got %s, want %s", response.ID, report.ID)
	}
	if response.CreatedBy != testutil.User5ID {
		t.Errorf("PATCH must not change created_by: got %s, want %s", response.CreatedBy, testutil.User5ID)
	}
}

func TestReportCreateRequiresTarget(t *testing.T) {
	env := testutil.SetupTest(t)

	tests := []struct {
		name string
		body map[string]any
	}{
		{"missing reported_topic", map[string]any{"reported_id": testutil.User2ID}},
		{"missing reported_id", map[string]any{"reported_topic": "users"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w := testutil.RequestAsUser(t, env.Router, http.MethodPost, "/v1/report", testutil.User5ID, testutil.JSONBody(t, test.body))
			testutil.RequireStatus(t, w, http.StatusUnprocessableEntity)
		})
	}
}

func TestReportListUnresolved(t *testing.T) {
	env := testutil.SetupTest(t)

	first := createReport(t, env, testutil.User5ID)
	second := createReport(t, env, testutil.User4ID)

	// Regular users may not use the moderator endpoint.
	w := testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/report/unresolved", testutil.User5ID, nil)
	testutil.RequireStatus(t, w, http.StatusForbidden)

	list := func(userID uuid.UUID) []uuid.UUID {
		t.Helper()
		w := testutil.RequestAsUser(t, env.Router, http.MethodGet, "/v1/report/unresolved", userID, nil)
		testutil.RequireStatus(t, w, http.StatusOK)
		var ids []uuid.UUID
		testutil.DecodeJSON(t, w, &ids)
		return ids
	}
	contains := func(ids []uuid.UUID, id uuid.UUID) bool {
		for _, candidate := range ids {
			if candidate == id {
				return true
			}
		}
		return false
	}

	ids := list(testutil.User2ID)
	if !contains(ids, first.ID) || !contains(ids, second.ID) {
		t.Fatalf("expected both unresolved reports for the moderator, got %v", ids)
	}

	// Resolving a report removes it from the unresolved list.
	w = testutil.RequestAsUser(t, env.Router, http.MethodPatch, "/v1/report/"+first.ID.String(), testutil.User2ID,
		testutil.JSONBody(t, map[string]any{"resolved": true, "public_notes": "done", "version": 1}))
	testutil.RequireStatus(t, w, http.StatusOK)

	for _, moderator := range []uuid.UUID{testutil.User2ID, testutil.User1ID} {
		ids = list(moderator)
		if contains(ids, first.ID) {
			t.Errorf("resolved report %s should not be listed as unresolved", first.ID)
		}
		if !contains(ids, second.ID) {
			t.Errorf("expected unresolved report %s in the list", second.ID)
		}
	}
}
