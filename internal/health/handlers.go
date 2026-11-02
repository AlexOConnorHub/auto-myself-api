package health

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-contrib/slog"
	"github.com/gin-gonic/gin"
)

const readinessTimeout = 2 * time.Second

type Pinger interface {
	Ping(ctx context.Context) error
}

type HealthHandler interface {
	Liveness(*gin.Context)
	Readiness(*gin.Context)
}

type Handler struct {
	pinger Pinger
}

func NewHandler(pinger Pinger) *Handler {
	return &Handler{
		pinger: pinger,
	}
}

func (h *Handler) Liveness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) Readiness(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), readinessTimeout)
	defer cancel()

	if err := h.pinger.Ping(ctx); err != nil {
		slog.Get(c).ErrorContext(ctx, "readiness check failed", "error", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
