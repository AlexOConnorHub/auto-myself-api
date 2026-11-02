package health_test

import (
	"auto-myself-api/internal/health"
	"auto-myself-api/internal/middleware"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakePinger struct {
	err         error
	calls       int
	hasDeadline bool
}

func (p *fakePinger) Ping(ctx context.Context) error {
	p.calls++
	_, p.hasDeadline = ctx.Deadline()
	return p.err
}

func newProbeRouter(t *testing.T, pinger health.Pinger) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)

	// Readiness logs through slog.Get(c), which panics without the logger middleware.
	router := gin.New()
	router.Use(middleware.Logger())

	handler := health.NewHandler(pinger)
	router.GET("/healthz", handler.Liveness)
	router.GET("/readyz", handler.Readiness)
	return router
}

func serve(t *testing.T, router http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
	return recorder
}

func decodeStatus(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()

	var body struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decoding response body %q: %v", recorder.Body.String(), err)
	}
	return body.Status
}

func TestLivenessDoesNotPing(t *testing.T) {
	pinger := &fakePinger{err: errors.New("database down")}
	recorder := serve(t, newProbeRouter(t, pinger), "/healthz")

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", recorder.Code, recorder.Body.String())
	}
	if pinger.calls != 0 {
		t.Errorf("expected liveness not to ping, got %d calls", pinger.calls)
	}
	if got := decodeStatus(t, recorder); got != "ok" {
		t.Errorf("expected status ok, got %q", got)
	}
}

func TestReadinessHealthy(t *testing.T) {
	pinger := &fakePinger{}
	recorder := serve(t, newProbeRouter(t, pinger), "/readyz")

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", recorder.Code, recorder.Body.String())
	}
	if pinger.calls != 1 {
		t.Errorf("expected exactly one ping, got %d", pinger.calls)
	}
	if !pinger.hasDeadline {
		t.Error("expected readiness ping context to have a deadline")
	}
	if got := decodeStatus(t, recorder); got != "ok" {
		t.Errorf("expected status ok, got %q", got)
	}
}

func TestReadinessUnavailable(t *testing.T) {
	pinger := &fakePinger{err: errors.New("dial tcp 10.0.0.5:5432: connection refused")}
	recorder := serve(t, newProbeRouter(t, pinger), "/readyz")

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d; body: %s", recorder.Code, recorder.Body.String())
	}
	if pinger.calls != 1 {
		t.Errorf("expected exactly one ping, got %d", pinger.calls)
	}
	if got := decodeStatus(t, recorder); got != "unavailable" {
		t.Errorf("expected status unavailable, got %q", got)
	}
	if body := recorder.Body.String(); strings.Contains(body, "10.0.0.5") {
		t.Errorf("readiness response leaked the internal error: %s", body)
	}
}
