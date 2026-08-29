package middleware

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type rateLimitEntry struct {
	started time.Time
	count   int
}

var rateLimitState = struct {
	sync.Mutex
	entries map[string]rateLimitEntry
}{entries: make(map[string]rateLimitEntry)}

var redisRateLimitState = struct {
	sync.Once
	client   *redis.Client
	enabled  bool
	required bool
	err      error
}{}

// RateLimit protects sensitive endpoints with a Redis-backed counter when
// configured. Local development falls back to the in-process limiter, while
// production can fail closed with RATE_LIMIT_REDIS_REQUIRED=true.
func RateLimit(name string, max int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := name + ":" + c.ClientIP()
		if allowed, remaining, retryAfter, handled := applyRedisRateLimit(c, key, max, window); handled {
			if c.IsAborted() {
				return
			}
			writeRateLimitHeaders(c, max, remaining)
			if !allowed {
				c.Header("Retry-After", stringInt(retryAfter))
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests", "retry_after_seconds": retryAfter})
				return
			}
			c.Next()
			return
		}

		now := time.Now()
		rateLimitState.Lock()
		entry := rateLimitState.entries[key]
		if entry.started.IsZero() || now.Sub(entry.started) >= window {
			entry = rateLimitEntry{started: now}
		}
		if len(rateLimitState.entries) > 10000 {
			for existingKey, existing := range rateLimitState.entries {
				if now.Sub(existing.started) >= window {
					delete(rateLimitState.entries, existingKey)
				}
			}
		}
		entry.count++
		rateLimitState.entries[key] = entry
		allowed := entry.count <= max
		remaining := max - entry.count
		if remaining < 0 {
			remaining = 0
		}
		rateLimitState.Unlock()

		writeRateLimitHeaders(c, max, remaining)
		if !allowed {
			c.Header("Retry-After", stringInt(int(window.Seconds())))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests", "retry_after_seconds": int(window.Seconds())})
			return
		}
		c.Next()
	}
}

func applyRedisRateLimit(c *gin.Context, key string, max int, window time.Duration) (bool, int, int, bool) {
	client, required, err := redisRateLimiter()
	if err != nil {
		if required {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "rate limiter unavailable"})
			return false, 0, 0, true
		}
		return false, 0, 0, false
	}
	if client == nil {
		if required {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "rate limiter unavailable"})
			return false, 0, 0, true
		}
		return false, 0, 0, false
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 500*time.Millisecond)
	defer cancel()

	redisKey := "statgate:registry:ratelimit:" + key
	count, err := client.Incr(ctx, redisKey).Result()
	if err == nil && count == 1 {
		err = client.Expire(ctx, redisKey, window).Err()
	}
	if err != nil {
		if required {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "rate limiter unavailable"})
			return false, 0, 0, true
		}
		return false, 0, 0, false
	}

	remaining := max - int(count)
	if remaining < 0 {
		remaining = 0
	}
	retryAfter := int(window.Seconds())
	if ttl, err := client.TTL(ctx, redisKey).Result(); err == nil && ttl > 0 {
		retryAfter = int(ttl.Seconds())
	}
	return count <= int64(max), remaining, retryAfter, true
}

func redisRateLimiter() (*redis.Client, bool, error) {
	redisRateLimitState.Do(func() {
		redisRateLimitState.required = strings.EqualFold(os.Getenv("RATE_LIMIT_REDIS_REQUIRED"), "true")
		addr := strings.TrimSpace(os.Getenv("REDIS_ADDR"))
		if addr == "" {
			host := strings.TrimSpace(os.Getenv("REDIS_HOST"))
			if host == "" {
				return
			}
			port := strings.TrimSpace(os.Getenv("REDIS_PORT"))
			if port == "" {
				port = "6379"
			}
			addr = host + ":" + port
		}

		dbIndex, _ := strconv.Atoi(strings.TrimSpace(os.Getenv("REDIS_DB")))
		client := redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: os.Getenv("REDIS_PASSWORD"),
			DB:       dbIndex,
		})

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		if err := client.Ping(ctx).Err(); err != nil {
			_ = client.Close()
			redisRateLimitState.err = err
			return
		}

		redisRateLimitState.client = client
		redisRateLimitState.enabled = true
	})
	if !redisRateLimitState.enabled {
		return nil, redisRateLimitState.required, redisRateLimitState.err
	}
	return redisRateLimitState.client, redisRateLimitState.required, nil
}

func writeRateLimitHeaders(c *gin.Context, max int, remaining int) {
	c.Header("X-RateLimit-Limit", stringInt(max))
	c.Header("X-RateLimit-Remaining", stringInt(remaining))
}

func stringInt(value int) string {
	if value == 0 {
		return "0"
	}
	const digits = "0123456789"
	result := ""
	for value > 0 {
		result = string(digits[value%10]) + result
		value /= 10
	}
	return result
}
