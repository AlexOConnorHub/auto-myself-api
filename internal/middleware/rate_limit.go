package middleware

import (
	"auto-myself-api/internal/helpers"
	"auto-myself-api/internal/users"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type Limit struct {
	limit int
	burst int
}

var (
	nonUserLimits = Limit{limit: 20, burst: 50}
	userLimits    = Limit{limit: 100, burst: 150}
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type visitorStore struct {
	mu        sync.Mutex
	visitors  map[string]*visitor
	lastSweep time.Time
}

func newVisitorStore() *visitorStore {
	return &visitorStore{
		visitors:  make(map[string]*visitor),
		lastSweep: time.Now(),
	}
}

func (s *visitorStore) get(key string, limits Limit) *rate.Limiter {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if now.Sub(s.lastSweep) > time.Minute {
		for visitorKey, v := range s.visitors {
			if now.Sub(v.lastSeen) > 3*time.Minute {
				delete(s.visitors, visitorKey)
			}
		}
		s.lastSweep = now
	}

	v, exists := s.visitors[key]
	if !exists {
		v = &visitor{limiter: rate.NewLimiter(rate.Limit(limits.limit), limits.burst)}
		s.visitors[key] = v
	}
	v.lastSeen = now
	return v.limiter
}

var (
	authVisitors = newVisitorStore()
	userVisitors = newVisitorStore()
)

func RateLimitMiddleware() gin.HandlerFunc {
	return rateLimitBy(func(c *gin.Context) string { return c.ClientIP() }, nonUserLimits, authVisitors)
}

func UserRateLimitMiddleware() gin.HandlerFunc {
	return rateLimitBy(func(c *gin.Context) string {
		if user, exists := c.Get(helpers.UserCtxKey); exists {
			if typed, ok := user.(*users.User); ok {
				return typed.ID.String()
			}
		}
		return c.ClientIP()
	}, userLimits, userVisitors)
}

func rateLimitBy(key func(*gin.Context) string, limits Limit, store *visitorStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !store.get(key(c), limits).Allow() {
			c.Header("Retry-After", "60")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
			})
			return
		}

		c.Next()
	}
}
