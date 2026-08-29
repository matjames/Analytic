package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func TestRateLimitFallsBackToMemoryWhenRedisIsNotConfigured(t *testing.T) {
	t.Setenv("REDIS_ADDR", "")
	t.Setenv("REDIS_HOST", "")
	t.Setenv("RATE_LIMIT_REDIS_REQUIRED", "")
	resetRateLimitForTest()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/limited", RateLimit("test", 2, time.Minute), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	for i := 0; i < 2; i++ {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/limited", nil)
		req.RemoteAddr = "192.0.2.10:1234"
		router.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusNoContent {
			t.Fatalf("request %d expected 204, got %d", i+1, recorder.Code)
		}
	}

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/limited", nil)
	req.RemoteAddr = "192.0.2.10:1234"
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("expected third request to be limited, got %d", recorder.Code)
	}
}

func TestRateLimitFailsClosedWhenRedisIsRequired(t *testing.T) {
	t.Setenv("REDIS_ADDR", "127.0.0.1:1")
	t.Setenv("RATE_LIMIT_REDIS_REQUIRED", "true")
	resetRateLimitForTest()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/limited", RateLimit("test-required", 2, time.Minute), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/limited", nil)
	req.RemoteAddr = "192.0.2.20:1234"
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected required Redis outage to fail closed, got %d", recorder.Code)
	}
}

func TestRateLimitFailsClosedWhenRequiredRedisIsNotConfigured(t *testing.T) {
	t.Setenv("REDIS_ADDR", "")
	t.Setenv("REDIS_HOST", "")
	t.Setenv("RATE_LIMIT_REDIS_REQUIRED", "true")
	resetRateLimitForTest()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/limited", RateLimit("test-required-missing", 2, time.Minute), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/limited", nil)
	req.RemoteAddr = "192.0.2.25:1234"
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected missing required Redis to fail closed, got %d", recorder.Code)
	}
}

func TestRateLimitUsesRedisWhenConfigured(t *testing.T) {
	redisAddr := os.Getenv("REGISTRY_TEST_REDIS_ADDR")
	if redisAddr == "" {
		t.Skip("REGISTRY_TEST_REDIS_ADDR is not set")
	}
	t.Setenv("REDIS_ADDR", redisAddr)
	t.Setenv("RATE_LIMIT_REDIS_REQUIRED", "true")
	resetRateLimitForTest()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/limited", RateLimit("test-redis", 2, time.Minute), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	for i := 0; i < 2; i++ {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/limited", nil)
		req.RemoteAddr = "192.0.2.30:1234"
		router.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusNoContent {
			t.Fatalf("redis-backed request %d expected 204, got %d", i+1, recorder.Code)
		}
	}

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/limited", nil)
	req.RemoteAddr = "192.0.2.30:1234"
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("expected Redis-backed limiter to reject third request, got %d", recorder.Code)
	}
}

func resetRateLimitForTest() {
	rateLimitState.Lock()
	rateLimitState.entries = make(map[string]rateLimitEntry)
	rateLimitState.Unlock()

	if redisRateLimitState.client != nil {
		_ = redisRateLimitState.client.Close()
	}
	redisRateLimitState = struct {
		sync.Once
		client   *redis.Client
		enabled  bool
		required bool
		err      error
	}{}
}
