package testutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gofrs/uuid/v5"
)

var (
	tokenMu    sync.Mutex
	tokenCache = make(map[uuid.UUID]string)
)

// BearerFor signs in as userID through /auth/development and caches the bearer token.
func BearerFor(t *testing.T, router http.Handler, userID uuid.UUID) string {
	t.Helper()

	tokenMu.Lock()
	defer tokenMu.Unlock()

	if token, exists := tokenCache[userID]; exists {
		return token
	}

	payload, err := json.Marshal(map[string]string{"user_id": userID.String()})
	if err != nil {
		t.Fatalf("marshalling development login request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/auth/development", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("development login for user %s failed with %d: %s", userID, recorder.Code, recorder.Body.String())
	}

	var response struct {
		Authentication string `json:"authentication"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decoding development login response: %v", err)
	}
	if response.Authentication == "" {
		t.Fatalf("development login for user %s returned an empty bearer token", userID)
	}

	tokenCache[userID] = response.Authentication
	return response.Authentication
}

// RequestAsUser performs an authenticated request as userID.
func RequestAsUser(t *testing.T, router http.Handler, method, path string, userID uuid.UUID, body io.Reader) *httptest.ResponseRecorder {
	t.Helper()
	return RequestWithBearer(t, router, method, path, BearerFor(t, router, userID), body)
}

func RequestWithBearer(t *testing.T, router http.Handler, method, path, bearer string, body io.Reader) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, body)
	if bearer != "" {
		req.Header.Set("Authorization", bearer)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

// RequireStatus asserts the status code and includes the response body in failures.
func RequireStatus(t *testing.T, recorder *httptest.ResponseRecorder, want int) {
	t.Helper()
	if recorder.Code != want {
		t.Fatalf("expected status %d, got %d; body: %s", want, recorder.Code, recorder.Body.String())
	}
}

// DecodeJSON decodes the response body or fails the test.
func DecodeJSON[T any](t *testing.T, recorder *httptest.ResponseRecorder, value *T) {
	t.Helper()
	if err := json.Unmarshal(recorder.Body.Bytes(), value); err != nil {
		t.Fatalf("decoding response body %q: %v", recorder.Body.String(), err)
	}
}

// JSONBody marshals a fixture into a request body.
func JSONBody(t *testing.T, value any) io.Reader {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshalling request body: %v", err)
	}
	return bytes.NewReader(body)
}

func UUID(value string) uuid.UUID {
	id, err := uuid.FromString(value)
	if err != nil {
		panic(fmt.Sprintf("invalid UUID %q: %v", value, err))
	}
	return id
}
