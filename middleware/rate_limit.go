package middleware

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
	// "github.com/go-redis/redis/v8"
	// "context"
)

var (
	visitors = make(map[string]*rate.Limiter)
	mu       sync.Mutex
)

func getVisitor(ip string) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()

	limiter, exists := visitors[ip]
	if !exists {
		limiter = rate.NewLimiter(1, 5) // 1 request/sec with burst of 5
		visitors[ip] = limiter
	}
	return limiter
}

// RateLimitMiddleware is an in-memory IP-based rate limiter.
func RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		limiter := getVisitor(ip)

		if !limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
			})
			return
		}

		c.Next()
	}
}

/*
   // Future: Redis-backed limiter (uncomment and complete configuration)
   var (
       ctx = context.Background()
       rdb = redis.NewClient(&redis.Options{
           Addr: "localhost:6379",
       })
   )

   func RedisRateLimitMiddleware() gin.HandlerFunc {
       return func(c *gin.Context) {
           ip := c.ClientIP()
           key := "ratelimit:" + ip
           count, _ := rdb.Incr(ctx, key).Result()

           if count == 1 {
               rdb.Expire(ctx, key, time.Minute)
           }

           if count > 60 {
               c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
                   "error": "Too many requests",
               })
               return
           }

           c.Next()
       }
   }
*/
