package middleware

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
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
		limiter = rate.NewLimiter(20, 50)
		visitors[ip] = limiter
	}
	return limiter
}

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
