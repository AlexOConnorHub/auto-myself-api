package server_test

import (
	"auto-myself-api/internal/server"
	"auto-myself-api/internal/testutil"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

func setupTimeoutEnv(t *testing.T, defaultTimeout, uploadTimeout time.Duration) *testutil.Env {
	t.Helper()

	warm := testutil.SetupTest(t)
	testutil.BearerFor(t, warm.Router, testutil.User1ID)

	return testutil.SetupTest(t, server.WithTimeouts(defaultTimeout, uploadTimeout))
}

// delayedBody sleeps once, then serves its data. It simulates a slow mobile uplink.
type delayedBody struct {
	delay time.Duration
	data  []byte
	once  sync.Once
}

func (b *delayedBody) Read(p []byte) (int, error) {
	b.once.Do(func() { time.Sleep(b.delay) })
	if len(b.data) == 0 {
		return 0, io.EOF
	}
	n := copy(p, b.data)
	b.data = b.data[n:]
	return n, nil
}

// blockingBody blocks in Read until Close. The safety timer makes a regression fail the test
// instead of hanging it.
type blockingBody struct {
	closed    chan struct{}
	closeOnce sync.Once
	safety    time.Duration
}

func newBlockingBody(safety time.Duration) *blockingBody {
	return &blockingBody{closed: make(chan struct{}), safety: safety}
}

func (b *blockingBody) Read([]byte) (int, error) {
	select {
	case <-b.closed:
		return 0, io.EOF
	case <-time.After(b.safety):
		return 0, io.EOF
	}
}

func (b *blockingBody) Close() error {
	b.closeOnce.Do(func() { close(b.closed) })
	return nil
}

func TestUploadUsesSeparateLongTimeout(t *testing.T) {
	env := setupTimeoutEnv(t, 1*time.Second, 4*time.Second)

	body := &delayedBody{delay: 1500 * time.Millisecond, data: []byte("slow upload")}
	w := testutil.RequestAsUser(t, env.Router, http.MethodPost,
		"/v1/file/maintenance/"+testutil.Maintenances[0].ID.String(), testutil.User1ID, body)
	testutil.RequireStatus(t, w, http.StatusCreated)

	if w.Header().Get("Location") == "" {
		t.Error("expected a Location header")
	}
	if env.Storage.Len() != 1 {
		t.Fatalf("expected 1 stored object, got %d", env.Storage.Len())
	}

	w = testutil.RequestAsUser(t, env.Router, http.MethodGet,
		"/v1/file/maintenance/"+testutil.Maintenances[0].ID.String(), testutil.User1ID, nil)
	testutil.RequireStatus(t, w, http.StatusOK)

	var entries []json.RawMessage
	testutil.DecodeJSON(t, w, &entries)
	if len(entries) != 1 {
		t.Fatalf("expected 1 file entry, got %d", len(entries))
	}
}

func TestUploadTimeoutReturnsJSONAndPersistsNothing(t *testing.T) {
	env := setupTimeoutEnv(t, 2*time.Second, 200*time.Millisecond)

	body := newBlockingBody(5 * time.Second)
	t.Cleanup(func() { _ = body.Close() })

	start := time.Now()
	w := testutil.RequestAsUser(t, env.Router, http.MethodPost,
		"/v1/file/maintenance/"+testutil.Maintenances[0].ID.String(), testutil.User1ID, body)
	elapsed := time.Since(start)

	testutil.RequireStatus(t, w, http.StatusRequestTimeout)
	if elapsed >= 1*time.Second {
		t.Errorf("expected the handler to return promptly after the timeout, took %s", elapsed)
	}
	if contentType := w.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
		t.Errorf("expected JSON content type, got %q", contentType)
	}
	if got, want := strings.TrimSpace(w.Body.String()), `{"error":"Request timeout"}`; got != want {
		t.Errorf("body = %s, want %s", got, want)
	}
	if env.Storage.Len() != 0 {
		t.Errorf("expected no stored objects, got %d", env.Storage.Len())
	}

	w = testutil.RequestAsUser(t, env.Router, http.MethodGet,
		"/v1/file/maintenance/"+testutil.Maintenances[0].ID.String(), testutil.User1ID, nil)
	testutil.RequireStatus(t, w, http.StatusOK)

	var entries []json.RawMessage
	testutil.DecodeJSON(t, w, &entries)
	if len(entries) != 0 {
		t.Errorf("expected no file rows after a timed-out upload, got %d", len(entries))
	}
}

func TestDefaultTimeoutAppliesToNonUploadRoutes(t *testing.T) {
	env := setupTimeoutEnv(t, 500*time.Millisecond, 4*time.Second)

	body := &delayedBody{delay: 1200 * time.Millisecond, data: []byte(`{"nickname":"slow"}`)}
	w := testutil.RequestAsUser(t, env.Router, http.MethodPost, "/v1/vehicle", testutil.User1ID, body)
	testutil.RequireStatus(t, w, http.StatusRequestTimeout)

	if contentType := w.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
		t.Errorf("expected JSON content type, got %q", contentType)
	}
}
