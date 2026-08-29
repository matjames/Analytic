package main

import (
	"maps"
	"testing"
	"time"
)

// ============== durationBucketIndex Tests ==============

func TestDurationBucketIndex(t *testing.T) {
	tests := []struct {
		ms       int64
		expected int
	}{
		{0, 0},
		{50, 0},
		{99, 0},
		{100, 1},
		{250, 1},
		{499, 1},
		{500, 2},
		{999, 2},
		{1000, 3},
		{2999, 3},
		{3000, 4},
		{9999, 4},
		{10000, 5},
		{100000, 5},
	}

	for _, tc := range tests {
		result := durationBucketIndex(tc.ms)
		if result != tc.expected {
			t.Errorf("durationBucketIndex(%d) = %d, want %d", tc.ms, result, tc.expected)
		}
	}
}

// ============== intToStr Tests ==============

func TestIntToStr(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{9, "9"},
		{10, "10"},
		{42, "42"},
		{100, "100"},
		{999, "999"},
		{12345, "12345"},
	}

	for _, tc := range tests {
		result := intToStr(tc.input)
		if result != tc.expected {
			t.Errorf("intToStr(%d) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

// ============== maps.Clone Tests ==============

func TestMapsClone(t *testing.T) {
	orig := map[string]int{"a": 1, "b": 2}
	cp := maps.Clone(orig)

	if len(cp) != 2 {
		t.Errorf("expected 2 entries, got %d", len(cp))
	}
	if cp["a"] != 1 || cp["b"] != 2 {
		t.Errorf("unexpected values: %v", cp)
	}

	// Mutating copy should not affect original
	cp["c"] = 3
	if _, ok := orig["c"]; ok {
		t.Error("mutating copy should not affect original")
	}
}

func TestMapsCloneEmpty(t *testing.T) {
	cp := maps.Clone(map[string]int{})
	if len(cp) != 0 {
		t.Errorf("expected empty map, got %d entries", len(cp))
	}
}

// ============== RecordRequest Tests ==============

func TestRecordRequest(t *testing.T) {
	// Initialize metrics
	InitMetrics()

	metric := RequestMetric{
		ReportID:   "test/report",
		Timestamp:  time.Now(),
		DurationMs: 150,
		Filters:    map[string]string{"district": "Kampala", "year": "2024"},
		StatusCode: 200,
		IsError:    false,
	}

	RecordRequest(metric)

	metricsStore.RLock()
	defer metricsStore.RUnlock()

	if metricsStore.reportCounts["test/report"] != 1 {
		t.Errorf("expected 1 request count, got %d", metricsStore.reportCounts["test/report"])
	}
	if metricsStore.reportDurMax["test/report"] != 150 {
		t.Errorf("expected max duration 150, got %d", metricsStore.reportDurMax["test/report"])
	}
	if metricsStore.durationBuckets[1] != 1 { // 100-500ms bucket
		t.Errorf("expected 1 in 100-500ms bucket, got %d", metricsStore.durationBuckets[1])
	}
}

func TestRecordRequestError(t *testing.T) {
	InitMetrics()

	metric := RequestMetric{
		ReportID:   "error/report",
		Timestamp:  time.Now(),
		DurationMs: 50,
		StatusCode: 500,
		IsError:    true,
	}

	RecordRequest(metric)

	metricsStore.RLock()
	defer metricsStore.RUnlock()

	if metricsStore.reportErrors["error/report"] != 1 {
		t.Errorf("expected 1 error count, got %d", metricsStore.reportErrors["error/report"])
	}
}

func TestRecordRequestNilStore(t *testing.T) {
	old := metricsStore
	metricsStore = nil
	// Should not panic
	RecordRequest(RequestMetric{ReportID: "test"})
	metricsStore = old
}

// ============== IncrementActiveRequests / DecrementActiveRequests Tests ==============

func TestActiveRequestTracking(t *testing.T) {
	InitMetrics()

	IncrementActiveRequests()
	IncrementActiveRequests()

	metricsStore.RLock()
	if metricsStore.activeRequests != 2 {
		t.Errorf("expected 2 active requests, got %d", metricsStore.activeRequests)
	}
	if metricsStore.peakConcurrent < 2 {
		t.Errorf("expected peak concurrent >= 2, got %d", metricsStore.peakConcurrent)
	}
	metricsStore.RUnlock()

	DecrementActiveRequests()

	metricsStore.RLock()
	if metricsStore.activeRequests != 1 {
		t.Errorf("expected 1 active request after decrement, got %d", metricsStore.activeRequests)
	}
	metricsStore.RUnlock()
}

func TestActiveRequestNilStore(t *testing.T) {
	old := metricsStore
	metricsStore = nil
	// Should not panic
	IncrementActiveRequests()
	DecrementActiveRequests()
	metricsStore = old
}

// ============== RecordRequest ring buffer Tests ==============

func TestRecordRequestRingBuffer(t *testing.T) {
	InitMetrics()
	metricsStore.maxRequests = 5

	for i := 0; i < 10; i++ {
		RecordRequest(RequestMetric{
			ReportID:   "test",
			Timestamp:  time.Now(),
			DurationMs: int64(i * 100),
		})
	}

	metricsStore.RLock()
	defer metricsStore.RUnlock()

	if len(metricsStore.requests) > 5 {
		t.Errorf("expected at most 5 requests in buffer, got %d", len(metricsStore.requests))
	}
}
