package main

import (
	"testing"
	"time"
)

func TestComputeYAMLHash(t *testing.T) {
	// Same content should produce same hash
	content1 := []byte("title: Test Report\nsections: []")
	content2 := []byte("title: Test Report\nsections: []")

	hash1 := ComputeYAMLHash(content1)
	hash2 := ComputeYAMLHash(content2)

	if hash1 != hash2 {
		t.Errorf("Same content should produce same hash: %s != %s", hash1, hash2)
	}

	// Different content should produce different hash
	content3 := []byte("title: Different Report\nsections: []")
	hash3 := ComputeYAMLHash(content3)

	if hash1 == hash3 {
		t.Errorf("Different content should produce different hash")
	}

	// Only generated_at changes — hash should stay stable (cache should not bust on timestamp churn)
	withTs := []byte("title: Test Report\ngenerated_at: \"2026-01-01 00:00:00\"\nsections: []")
	withTs2 := []byte("title: Test Report\ngenerated_at: \"2026-12-31 23:59:59\"\nsections: []")
	if ComputeYAMLHash(withTs) != ComputeYAMLHash(withTs2) {
		t.Errorf("generated_at-only changes should not change cache hash")
	}
}

func TestComponentCacheBuildKey(t *testing.T) {
	cache := &ComponentCache{ttl: 3 * 24 * time.Hour}

	filters := ReportFilters{
		Districts: []string{"Kampala", "Central"},
		Year:      "2024",
		Month:     "1",
	}

	key1 := cache.buildCacheKey("community/report", 0, 1, filters)
	key2 := cache.buildCacheKey("community/report", 0, 1, filters)

	if key1 != key2 {
		t.Errorf("Same inputs should produce same key: %s != %s", key1, key2)
	}

	// Different filters should produce different key
	filters2 := ReportFilters{
		Districts: []string{"Kampala"},
		Year:      "2024",
		Month:     "1",
	}
	key3 := cache.buildCacheKey("community/report", 0, 1, filters2)

	if key1 == key3 {
		t.Errorf("Different filters should produce different key")
	}

	// District order shouldn't matter (should be sorted)
	filters3 := ReportFilters{
		Districts: []string{"Central", "Kampala"}, // Reversed order
		Year:      "2024",
		Month:     "1",
	}
	key4 := cache.buildCacheKey("community/report", 0, 1, filters3)

	if key1 != key4 {
		t.Errorf("District order should not affect key: %s != %s", key1, key4)
	}

	// YearMonths must distinguish expanded multi-month ranges from single month
	filtersYM := ReportFilters{
		Year:  "2024",
		Month: "3",
		YearMonths: map[string][]int{
			"2024": {2, 3, 4},
		},
	}
	keyYM := cache.buildCacheKey("community/report", 0, 0, filtersYM)
	filtersNoYM := ReportFilters{Year: "2024", Month: "3"}
	keyNoYM := cache.buildCacheKey("community/report", 0, 0, filtersNoYM)
	if keyYM == keyNoYM {
		t.Errorf("YearMonths should produce a different cache key than month-only filters")
	}

	filtersYM2 := ReportFilters{
		Year:  "2024",
		Month: "3",
		YearMonths: map[string][]int{
			"2024": {1, 2, 3},
		},
	}
	keyYM2 := cache.buildCacheKey("community/report", 0, 0, filtersYM2)
	if keyYM == keyYM2 {
		t.Errorf("Different YearMonths maps should produce different keys")
	}
}

func TestComponentCacheGetSet(t *testing.T) {
	cache := &ComponentCache{ttl: 3 * 24 * time.Hour}

	filters := ReportFilters{
		Districts: []string{"Kampala"},
		Year:      "2024",
	}

	yamlHash := "abc123"
	result := ComponentResult{
		SectionIdx:   0,
		ComponentIdx: 1,
		Data:         Data{Labels: []string{"Jan", "Feb"}},
	}

	// Cache should be empty initially
	cached := cache.Get("test/report", 0, 1, filters, yamlHash)
	if cached != nil {
		t.Error("Cache should be empty initially")
	}

	// Set and retrieve
	cache.Set("test/report", 0, 1, filters, result, yamlHash)

	cached = cache.Get("test/report", 0, 1, filters, yamlHash)
	if cached == nil {
		t.Error("Cache should return stored result")
	}

	if len(cached.Data.Labels) != 2 {
		t.Errorf("Cached data should match: expected 2 labels, got %d", len(cached.Data.Labels))
	}
}

func TestComponentCacheYAMLHashInvalidation(t *testing.T) {
	cache := &ComponentCache{ttl: 3 * 24 * time.Hour}

	filters := ReportFilters{Year: "2024"}

	result := ComponentResult{SectionIdx: 0, ComponentIdx: 0}

	// Store with hash1
	cache.Set("test/report", 0, 0, filters, result, "hash1")

	// Should retrieve with same hash
	cached := cache.Get("test/report", 0, 0, filters, "hash1")
	if cached == nil {
		t.Error("Should retrieve with same hash")
	}

	// Should NOT retrieve with different hash (YAML changed)
	cached = cache.Get("test/report", 0, 0, filters, "hash2")
	if cached != nil {
		t.Error("Should not retrieve when YAML hash changed")
	}
}

func TestComponentCacheInvalidateReport(t *testing.T) {
	cache := &ComponentCache{ttl: 3 * 24 * time.Hour}

	filters := ReportFilters{Year: "2024"}
	result := ComponentResult{}

	// Store entries for two reports
	cache.Set("report-a", 0, 0, filters, result, "hash")
	cache.Set("report-a", 0, 1, filters, result, "hash")
	cache.Set("report-b", 0, 0, filters, result, "hash")

	// Invalidate report-a
	cache.InvalidateReport("report-a")

	// report-a entries should be gone
	if cache.Get("report-a", 0, 0, filters, "hash") != nil {
		t.Error("report-a should be invalidated")
	}
	if cache.Get("report-a", 0, 1, filters, "hash") != nil {
		t.Error("report-a component 1 should be invalidated")
	}

	// report-b should still exist
	if cache.Get("report-b", 0, 0, filters, "hash") == nil {
		t.Error("report-b should still be cached")
	}
}

func TestComponentCacheClear(t *testing.T) {
	cache := &ComponentCache{ttl: 3 * 24 * time.Hour}

	filters := ReportFilters{Year: "2024"}
	result := ComponentResult{}

	cache.Set("report-1", 0, 0, filters, result, "hash")
	cache.Set("report-2", 0, 0, filters, result, "hash")

	stats := cache.GetStats(false)
	if stats.TotalEntries != 2 {
		t.Errorf("Expected 2 entries, got %d", stats.TotalEntries)
	}

	cache.Clear()

	stats = cache.GetStats(false)
	if stats.TotalEntries != 0 {
		t.Errorf("Expected 0 entries after clear, got %d", stats.TotalEntries)
	}
}

func TestComponentCacheStats(t *testing.T) {
	cache := &ComponentCache{ttl: 3 * 24 * time.Hour}

	filters := ReportFilters{Year: "2024"}
	result := ComponentResult{}

	cache.Set("test/report", 0, 0, filters, result, "hash")

	stats := cache.GetStats(true)

	if stats.TotalEntries != 1 {
		t.Errorf("Expected 1 entry, got %d", stats.TotalEntries)
	}

	if stats.TTL != "72h0m0s" {
		t.Errorf("Expected TTL 72h0m0s, got %s", stats.TTL)
	}

	if len(stats.Entries) != 1 {
		t.Errorf("Expected 1 entry in details, got %d", len(stats.Entries))
	}
}

// ============== buildCacheKey Additional Tests ==============

func TestBuildCacheKeyWithRegions(t *testing.T) {
	cache := &ComponentCache{ttl: time.Hour}

	filters := ReportFilters{
		Regions: []string{"Central", "Eastern"},
		Year:    "2024",
	}
	key := cache.buildCacheKey("test/report", 0, 0, filters)

	if key == "" {
		t.Error("Cache key should not be empty")
	}

	// Reversed region order should produce same key (sorted)
	filters2 := ReportFilters{
		Regions: []string{"Eastern", "Central"},
		Year:    "2024",
	}
	key2 := cache.buildCacheKey("test/report", 0, 0, filters2)
	if key != key2 {
		t.Errorf("Region order should not affect key: %s != %s", key, key2)
	}
}

func TestBuildCacheKeyWithFacilities(t *testing.T) {
	cache := &ComponentCache{ttl: time.Hour}

	filters := ReportFilters{
		Facilities: []string{"Hospital A", "Clinic B"},
		Year:       "2024",
	}
	key := cache.buildCacheKey("test/report", 0, 0, filters)

	// Reversed facility order should produce same key (sorted)
	filters2 := ReportFilters{
		Facilities: []string{"Clinic B", "Hospital A"},
		Year:       "2024",
	}
	key2 := cache.buildCacheKey("test/report", 0, 0, filters2)
	if key != key2 {
		t.Errorf("Facility order should not affect key: %s != %s", key, key2)
	}
}

func TestBuildCacheKeyWithCustomValues(t *testing.T) {
	cache := &ComponentCache{ttl: time.Hour}

	filters := ReportFilters{
		CustomValues: map[string]string{
			"facility_type": "hospital",
			"age_group":     "under_5",
		},
	}
	key1 := cache.buildCacheKey("test/report", 0, 0, filters)

	// Same values in different insertion order should produce same key (sorted by key)
	filters2 := ReportFilters{
		CustomValues: map[string]string{
			"age_group":     "under_5",
			"facility_type": "hospital",
		},
	}
	key2 := cache.buildCacheKey("test/report", 0, 0, filters2)
	if key1 != key2 {
		t.Errorf("Custom value order should not affect key: %s != %s", key1, key2)
	}
}

func TestBuildCacheKeyWithQuarterAndWeek(t *testing.T) {
	cache := &ComponentCache{ttl: time.Hour}

	filters := ReportFilters{
		Quarter: "2",
		Week:    "2024W15",
	}
	key := cache.buildCacheKey("report", 1, 2, filters)

	if key == "" {
		t.Error("Cache key should not be empty")
	}

	// Different quarter should produce different key
	filters2 := ReportFilters{
		Quarter: "3",
		Week:    "2024W15",
	}
	key2 := cache.buildCacheKey("report", 1, 2, filters2)
	if key == key2 {
		t.Error("Different quarter should produce different key")
	}
}

func TestBuildCacheKeyEmptyFilters(t *testing.T) {
	cache := &ComponentCache{ttl: time.Hour}

	key := cache.buildCacheKey("report", 0, 0, ReportFilters{})
	expected := "report:0:0:"
	if key != expected {
		t.Errorf("expected %q, got %q", expected, key)
	}
}

// ============== evictIfNeeded Tests ==============

func TestEvictIfNeeded(t *testing.T) {
	cache := &ComponentCache{
		ttl:        time.Hour,
		maxEntries: 3,
	}

	result := ComponentResult{}
	filters := ReportFilters{}

	// Add 5 entries (over the limit of 3)
	for i := 0; i < 5; i++ {
		cache.Set("report", 0, i, filters, result, "hash")
		// Small delay so entries have different timestamps
		time.Sleep(time.Millisecond)
	}

	// After eviction, should be at or below max
	count := 0
	cache.entries.Range(func(_, _ any) bool {
		count++
		return true
	})

	if count > 3 {
		t.Errorf("expected at most 3 entries after eviction, got %d", count)
	}
}

func TestEvictIfNeededNoLimitSet(t *testing.T) {
	cache := &ComponentCache{
		ttl:        time.Hour,
		maxEntries: 0, // no limit
	}

	result := ComponentResult{}
	filters := ReportFilters{}

	// Add entries — should not evict
	for i := 0; i < 10; i++ {
		cache.Set("report", 0, i, filters, result, "hash")
	}

	count := 0
	cache.entries.Range(func(_, _ any) bool {
		count++
		return true
	})

	if count != 10 {
		t.Errorf("expected 10 entries with no limit, got %d", count)
	}
}

func TestCacheTTLExpiration(t *testing.T) {
	cache := &ComponentCache{
		ttl:        1 * time.Millisecond, // Very short TTL
		maxEntries: 100,
	}

	result := ComponentResult{}
	filters := ReportFilters{Year: "2024"}

	cache.Set("report", 0, 0, filters, result, "hash")
	time.Sleep(5 * time.Millisecond)

	// Should be expired
	cached := cache.Get("report", 0, 0, filters, "hash")
	if cached != nil {
		t.Error("Expected nil for expired entry")
	}
}

func TestCacheHitMissTracking(t *testing.T) {
	cache := &ComponentCache{
		ttl:        time.Hour,
		maxEntries: 100,
	}

	result := ComponentResult{}
	filters := ReportFilters{Year: "2024"}

	// Miss
	cache.Get("report", 0, 0, filters, "hash")
	// Set then hit
	cache.Set("report", 0, 0, filters, result, "hash")
	cache.Get("report", 0, 0, filters, "hash")

	stats := cache.GetStats(false)
	if stats.Misses < 1 {
		t.Errorf("expected at least 1 miss, got %d", stats.Misses)
	}
	if stats.Hits < 1 {
		t.Errorf("expected at least 1 hit, got %d", stats.Hits)
	}
	if stats.HitRate <= 0 {
		t.Errorf("expected positive hit rate, got %f", stats.HitRate)
	}
}
