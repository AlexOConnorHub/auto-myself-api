package middleware

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type Limit struct {
	limit int
	burst int
}

var userLimits = Limit{
	limit: 100,
	burst: 150,
}

var nonUserLimits = Limit{
	limit: 20,
	burst: 50,
}

var (
	visitors = make(map[string]*rate.Limiter)
	mu       sync.Mutex
)

func getVisitor(ip string, isUser bool) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()

	limiter, exists := visitors[ip]
	if !exists {
		if isUser {
			limiter = rate.NewLimiter(rate.Limit(userLimits.limit), userLimits.burst)
		} else {
			limiter = rate.NewLimiter(rate.Limit(nonUserLimits.limit), nonUserLimits.burst)
		}
		visitors[ip] = limiter
	}
	return limiter
}

func RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		_, isUser := c.Get("user")
		limiter := getVisitor(ip, isUser)

		if !limiter.Allow() {
			c.Header("Retry-After", "60") // Optional: inform the client to retry after 60 seconds
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
			})
			return
		}

		c.Next()
	}
}
