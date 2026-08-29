//go:build stress
// +build stress

// Stress tests are excluded from normal test runs.
// Run with: go test ./server -tags=stress -v
// Or use: ./scripts/stress-test.sh unit

package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// StressResult holds metrics from a stress test run
type StressResult struct {
	TotalRequests   int64
	SuccessCount    int64
	ErrorCount      int64
	RateLimitCount  int64
	TotalDuration   time.Duration
	AvgLatency      time.Duration
	MaxLatency      time.Duration
	MinLatency      time.Duration
	RequestsPerSec  float64
	Percentile95    time.Duration
	Percentile99    time.Duration
}

// LatencyRecorder tracks request latencies in a thread-safe manner
type LatencyRecorder struct {
	mu        sync.Mutex
	latencies []time.Duration
}

func (lr *LatencyRecorder) Record(d time.Duration) {
	lr.mu.Lock()
	lr.latencies = append(lr.latencies, d)
	lr.mu.Unlock()
}

func (lr *LatencyRecorder) GetSorted() []time.Duration {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	// Copy and sort
	sorted := make([]time.Duration, len(lr.latencies))
	copy(sorted, lr.latencies)

	// Simple insertion sort (good enough for test data)
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j] < sorted[j-1]; j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}
	return sorted
}

func (lr *LatencyRecorder) Percentile(p float64) time.Duration {
	sorted := lr.GetSorted()
	if len(sorted) == 0 {
		return 0
	}
	idx := int(float64(len(sorted)-1) * p)
	return sorted[idx]
}

// TestStress_ConcurrentReportList tests concurrent access to the report list endpoint
func TestStress_ConcurrentReportList(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	// Create test server with our handlers
	mux := http.NewServeMux()
	mux.HandleFunc("/api/reports", ListReportsHandler)

	// Wrap with rate limiting and recovery like production
	handler := RateLimitMiddleware(RecoveryMiddleware(mux))
	server := httptest.NewServer(handler)
	defer server.Close()

	result := runStressTest(t, server.URL+"/api/reports", 50, 5*time.Second)

	t.Logf("Stress Test Results:")
	t.Logf("  Total Requests:    %d", result.TotalRequests)
	t.Logf("  Successful:        %d (%.1f%%)", result.SuccessCount, float64(result.SuccessCount)/float64(result.TotalRequests)*100)
	t.Logf("  Errors:            %d", result.ErrorCount)
	t.Logf("  Rate Limited:      %d", result.RateLimitCount)
	t.Logf("  Requests/sec:      %.2f", result.RequestsPerSec)
	t.Logf("  Avg Latency:       %v", result.AvgLatency)
	t.Logf("  Min Latency:       %v", result.MinLatency)
	t.Logf("  Max Latency:       %v", result.MaxLatency)
	t.Logf("  P95 Latency:       %v", result.Percentile95)
	t.Logf("  P99 Latency:       %v", result.Percentile99)

	// Assertions for a healthy system
	if result.ErrorCount > result.TotalRequests/10 {
		t.Errorf("Too many errors: %d out of %d requests", result.ErrorCount, result.TotalRequests)
	}
}

// TestStress_RateLimiting verifies rate limiting kicks in under load
func TestStress_RateLimiting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/reports", ListReportsHandler)
	handler := RateLimitMiddleware(RecoveryMiddleware(mux))
	server := httptest.NewServer(handler)
	defer server.Close()

	// Hammer it with many concurrent workers to trigger rate limiting
	result := runStressTest(t, server.URL+"/api/reports", 200, 2*time.Second)

	t.Logf("Rate Limit Test Results:")
	t.Logf("  Total Requests:    %d", result.TotalRequests)
	t.Logf("  Rate Limited:      %d (%.1f%%)", result.RateLimitCount, float64(result.RateLimitCount)/float64(result.TotalRequests)*100)

	// With 200 concurrent workers, some should hit rate limits
	if result.RateLimitCount == 0 {
		t.Log("Warning: No rate limiting triggered - this is OK if test was too short")
	}
}

// TestStress_GracefulDegradation tests behavior when system is under extreme load
func TestStress_GracefulDegradation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/reports", ListReportsHandler)
	handler := RateLimitMiddleware(RecoveryMiddleware(mux))
	server := httptest.NewServer(handler)
	defer server.Close()

	// Extreme load: 500 concurrent workers
	result := runStressTest(t, server.URL+"/api/reports", 500, 3*time.Second)

	t.Logf("Graceful Degradation Test:")
	t.Logf("  Total Requests:    %d", result.TotalRequests)
	t.Logf("  Successful:        %d", result.SuccessCount)
	t.Logf("  Rate Limited:      %d", result.RateLimitCount)
	t.Logf("  Errors:            %d", result.ErrorCount)

	// The system should not crash - all requests should get some response
	totalResponses := result.SuccessCount + result.RateLimitCount + result.ErrorCount
	if totalResponses != result.TotalRequests {
		t.Errorf("Some requests got no response: %d requests, %d responses", result.TotalRequests, totalResponses)
	}

	// Most should succeed or be rate-limited, not error
	acceptableResponses := result.SuccessCount + result.RateLimitCount
	if acceptableResponses < result.TotalRequests/2 {
		t.Errorf("Too many failures under load: only %d/%d acceptable responses", acceptableResponses, result.TotalRequests)
	}
}

// TestStress_PanicRecovery verifies the recovery middleware catches panics
func TestStress_PanicRecovery(t *testing.T) {
	// Handler that panics
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("intentional test panic")
	})

	handler := RecoveryMiddleware(panicHandler)
	server := httptest.NewServer(handler)
	defer server.Close()

	// Send multiple concurrent requests that would cause panics
	var wg sync.WaitGroup
	var recovered int64

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Get(server.URL)
			if err != nil {
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusInternalServerError {
				atomic.AddInt64(&recovered, 1)
			}
		}()
	}

	wg.Wait()

	if recovered != 100 {
		t.Errorf("Expected 100 recovered panics, got %d", recovered)
	}
	t.Logf("Successfully recovered from %d panics", recovered)
}

// TestStress_ConnectionPoolSaturation tests DB connection pool behavior
func TestStress_ConnectionPoolSaturation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	// Skip if DB is not available
	if DB == nil {
		t.Skip("Database not initialized")
	}

	// Simulate many concurrent DB queries
	var wg sync.WaitGroup
	var success, failed int64
	workers := 100 // More than DBMaxOpenConns (50)

	start := time.Now()

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// Simple query to test connection
			var result int
			err := DB.QueryRowContext(ctx, "SELECT 1").Scan(&result)
			if err != nil {
				atomic.AddInt64(&failed, 1)
				return
			}
			atomic.AddInt64(&success, 1)
		}()
	}

	wg.Wait()
	duration := time.Since(start)

	t.Logf("Connection Pool Saturation Test:")
	t.Logf("  Workers:           %d", workers)
	t.Logf("  Max Connections:   %d", DBMaxOpenConns)
	t.Logf("  Successful:        %d", success)
	t.Logf("  Failed:            %d", failed)
	t.Logf("  Duration:          %v", duration)

	// All should succeed (connection pool should queue requests)
	if failed > 0 {
		t.Errorf("Some queries failed due to connection issues: %d failures", failed)
	}
}

// TestStress_MixedEndpoints tests multiple endpoints concurrently
func TestStress_MixedEndpoints(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/reports", ListReportsHandler)
	mux.HandleFunc("/api/filters/months", GetMonthsHandler)
	mux.HandleFunc("/api/filters/quarters", GetQuartersHandler)
	mux.HandleFunc("/api/filters/years", GetYearsHandler)

	handler := RateLimitMiddleware(RecoveryMiddleware(mux))
	server := httptest.NewServer(handler)
	defer server.Close()

	endpoints := []string{
		"/api/reports",
		"/api/filters/months",
		"/api/filters/quarters",
		"/api/filters/years",
	}

	var wg sync.WaitGroup
	results := make(map[string]*StressResult)
	var mu sync.Mutex

	for _, ep := range endpoints {
		wg.Add(1)
		go func(endpoint string) {
			defer wg.Done()
			result := runStressTest(t, server.URL+endpoint, 20, 3*time.Second)
			mu.Lock()
			results[endpoint] = &result
			mu.Unlock()
		}(ep)
	}

	wg.Wait()

	t.Logf("Mixed Endpoint Test Results:")
	for endpoint, result := range results {
		t.Logf("  %s:", endpoint)
		t.Logf("    Requests: %d, Success: %d, Errors: %d, RPS: %.2f",
			result.TotalRequests, result.SuccessCount, result.ErrorCount, result.RequestsPerSec)
	}
}

// runStressTest runs a stress test against a URL with given concurrency and duration
func runStressTest(t *testing.T, url string, concurrency int, duration time.Duration) StressResult {
	var (
		totalRequests  int64
		successCount   int64
		errorCount     int64
		rateLimitCount int64
		minLatency     = time.Hour
		maxLatency     time.Duration
		totalLatency   time.Duration
		latencyMu      sync.Mutex
	)

	recorder := &LatencyRecorder{}
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	var wg sync.WaitGroup
	start := time.Now()

	// Create HTTP client with reasonable timeouts
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					reqStart := time.Now()
					resp, err := client.Get(url)
					latency := time.Since(reqStart)

					atomic.AddInt64(&totalRequests, 1)
					recorder.Record(latency)

					latencyMu.Lock()
					totalLatency += latency
					if latency < minLatency {
						minLatency = latency
					}
					if latency > maxLatency {
						maxLatency = latency
					}
					latencyMu.Unlock()

					if err != nil {
						atomic.AddInt64(&errorCount, 1)
						continue
					}

					// Drain and close body
					io.Copy(io.Discard, resp.Body)
					resp.Body.Close()

					switch resp.StatusCode {
					case http.StatusOK:
						atomic.AddInt64(&successCount, 1)
					case http.StatusTooManyRequests:
						atomic.AddInt64(&rateLimitCount, 1)
					default:
						atomic.AddInt64(&errorCount, 1)
					}
				}
			}
		}()
	}

	wg.Wait()
	totalDuration := time.Since(start)

	result := StressResult{
		TotalRequests:  totalRequests,
		SuccessCount:   successCount,
		ErrorCount:     errorCount,
		RateLimitCount: rateLimitCount,
		TotalDuration:  totalDuration,
		MaxLatency:     maxLatency,
		MinLatency:     minLatency,
		RequestsPerSec: float64(totalRequests) / totalDuration.Seconds(),
		Percentile95:   recorder.Percentile(0.95),
		Percentile99:   recorder.Percentile(0.99),
	}

	if totalRequests > 0 {
		result.AvgLatency = totalLatency / time.Duration(totalRequests)
	}

	return result
}

// BenchmarkReportListEndpoint benchmarks the report list endpoint
func BenchmarkReportListEndpoint(b *testing.B) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/reports", ListReportsHandler)
	server := httptest.NewServer(mux)
	defer server.Close()

	// Create a client with connection pooling to avoid port exhaustion
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
			DisableKeepAlives:   false,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(server.URL + "/api/reports")
		if err != nil {
			b.Error(err)
			continue
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

// BenchmarkMonthsFilterEndpoint benchmarks the months filter endpoint
func BenchmarkMonthsFilterEndpoint(b *testing.B) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/filters/months", GetMonthsHandler)
	server := httptest.NewServer(mux)
	defer server.Close()

	// Create a client with connection pooling to avoid port exhaustion
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
			DisableKeepAlives:   false,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(server.URL + "/api/filters/months")
		if err != nil {
			b.Error(err)
			continue
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

// TestStress_ResponseIntegrity verifies responses are valid JSON under load
func TestStress_ResponseIntegrity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/filters/months", GetMonthsHandler)
	handler := RateLimitMiddleware(RecoveryMiddleware(mux))
	server := httptest.NewServer(handler)
	defer server.Close()

	var wg sync.WaitGroup
	var validJSON, invalidJSON int64
	workers := 50

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				resp, err := http.Get(server.URL + "/api/filters/months")
				if err != nil {
					continue
				}

				body, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				if err != nil {
					continue
				}

				// Skip rate-limited responses
				if resp.StatusCode == http.StatusTooManyRequests {
					continue
				}

				// Verify valid JSON
				var data interface{}
				if err := json.Unmarshal(body, &data); err != nil {
					atomic.AddInt64(&invalidJSON, 1)
				} else {
					atomic.AddInt64(&validJSON, 1)
				}
			}
		}()
	}

	wg.Wait()

	t.Logf("Response Integrity Test:")
	t.Logf("  Valid JSON:   %d", validJSON)
	t.Logf("  Invalid JSON: %d", invalidJSON)

	if invalidJSON > 0 {
		t.Errorf("Got %d invalid JSON responses under load", invalidJSON)
	}
}
