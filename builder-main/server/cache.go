package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"
)

// CacheTTL is the time-to-live for cached component results
const CacheTTL = 24 * time.Hour // 1 day

const cacheFilePath = "data/cache.json"

// MaxCacheEntries is the maximum number of entries before eviction
const MaxCacheEntries = 15000

// CacheEntry holds a cached component result with metadata
type CacheEntry struct {
	Data      ComponentResult `json:"data"`
	CreatedAt time.Time       `json:"createdAt"`
	YAMLHash  string          `json:"yamlHash"`
}

// ComponentCache provides thread-safe caching for component results
type ComponentCache struct {
	entries    sync.Map
	ttl        time.Duration
	maxEntries int
	hits       atomic.Int64
	misses     atomic.Int64
}

// Global cache instance
var componentCache = &ComponentCache{
	ttl:        CacheTTL,
	maxEntries: MaxCacheEntries,
}

// cacheFlight deduplicates concurrent cache misses for the same component+filters
var cacheFlight singleflight.Group

// generatedAtYAMLLine matches top-level or indented generated_at keys so routine
// timestamp edits do not invalidate the entire component cache.
var generatedAtYAMLLine = regexp.MustCompile(`(?m)^[ \t]*generated_at:\s*.*\r?\n?`)

// normalizeYAMLForCacheHash strips volatile keys that should not bust cache when unchanged logically.
func normalizeYAMLForCacheHash(yamlContent []byte) []byte {
	return generatedAtYAMLLine.ReplaceAll(yamlContent, nil)
}

// ComputeYAMLHash generates a SHA256 hash of the YAML content
// Used to detect when a report definition has changed
func ComputeYAMLHash(yamlContent []byte) string {
	normalized := normalizeYAMLForCacheHash(yamlContent)
	hash := sha256.Sum256(normalized)
	return hex.EncodeToString(hash[:])
}

// appendYearMonthsCacheKey appends a stable encoding of YearMonths to filter key parts.
func appendYearMonthsCacheKey(parts []string, ym map[string][]int) []string {
	if len(ym) == 0 {
		return parts
	}
	years := make([]string, 0, len(ym))
	for y := range ym {
		years = append(years, y)
	}
	sort.Strings(years)
	var b strings.Builder
	b.WriteString("yearmonths=")
	for i, y := range years {
		if i > 0 {
			b.WriteString("|")
		}
		months := append([]int(nil), ym[y]...)
		sort.Ints(months)
		b.WriteString(y)
		b.WriteString(":")
		for j, m := range months {
			if j > 0 {
				b.WriteString(",")
			}
			b.WriteString(strconv.Itoa(m))
		}
	}
	return append(parts, b.String())
}

// appendYearQuartersCacheKey appends a stable encoding of YearQuarters to filter key parts.
func appendYearQuartersCacheKey(parts []string, yq map[string][]int) []string {
	if len(yq) == 0 {
		return parts
	}
	years := make([]string, 0, len(yq))
	for y := range yq {
		years = append(years, y)
	}
	sort.Strings(years)
	var b strings.Builder
	b.WriteString("yearquarters=")
	for i, y := range years {
		if i > 0 {
			b.WriteString("|")
		}
		quarters := append([]int(nil), yq[y]...)
		sort.Ints(quarters)
		b.WriteString(y)
		b.WriteString(":")
		for j, q := range quarters {
			if j > 0 {
				b.WriteString(",")
			}
			b.WriteString(strconv.Itoa(q))
		}
	}
	return append(parts, b.String())
}

// FilterFingerprint returns a stable string for ReportFilters (districts, time, custom, etc.).
// Used by component cache keys and report-level response cache keys.
func FilterFingerprint(filters ReportFilters) string {
	filterParts := []string{}

	if len(filters.Districts) > 0 {
		sorted := make([]string, len(filters.Districts))
		copy(sorted, filters.Districts)
		sort.Strings(sorted)
		filterParts = append(filterParts, fmt.Sprintf("district=%s", strings.Join(sorted, ",")))
	}

	if len(filters.Regions) > 0 {
		sorted := make([]string, len(filters.Regions))
		copy(sorted, filters.Regions)
		sort.Strings(sorted)
		filterParts = append(filterParts, fmt.Sprintf("region=%s", strings.Join(sorted, ",")))
	}

	if len(filters.Facilities) > 0 {
		sorted := make([]string, len(filters.Facilities))
		copy(sorted, filters.Facilities)
		sort.Strings(sorted)
		filterParts = append(filterParts, fmt.Sprintf("facility=%s", strings.Join(sorted, ",")))
	}

	if filters.Year != "" {
		filterParts = append(filterParts, fmt.Sprintf("year=%s", filters.Year))
	}
	if filters.Month != "" {
		filterParts = append(filterParts, fmt.Sprintf("month=%s", filters.Month))
	}
	if filters.Quarter != "" {
		filterParts = append(filterParts, fmt.Sprintf("quarter=%s", filters.Quarter))
	}
	if filters.Week != "" {
		filterParts = append(filterParts, fmt.Sprintf("week=%s", filters.Week))
	}

	filterParts = appendYearMonthsCacheKey(filterParts, filters.YearMonths)
	filterParts = appendYearQuartersCacheKey(filterParts, filters.YearQuarters)

	if len(filters.CustomValues) > 0 {
		keys := make([]string, 0, len(filters.CustomValues))
		for k := range filters.CustomValues {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			filterParts = append(filterParts, fmt.Sprintf("%s=%s", k, filters.CustomValues[k]))
		}
	}

	return strings.Join(filterParts, "&")
}

// buildCacheKey creates a unique cache key from report ID, component position, and filters
func (c *ComponentCache) buildCacheKey(reportID string, sectionIdx, componentIdx int, filters ReportFilters) string {
	fp := FilterFingerprint(filters)
	return fmt.Sprintf("%s:%d:%d:%s", reportID, sectionIdx, componentIdx, fp)
}

// Get retrieves a cached component result if valid
// Returns nil if not found, expired, or YAML has changed
func (c *ComponentCache) Get(reportID string, sectionIdx, componentIdx int, filters ReportFilters, currentYAMLHash string) *ComponentResult {
	key := c.buildCacheKey(reportID, sectionIdx, componentIdx, filters)

	value, ok := c.entries.Load(key)
	if !ok {
		c.misses.Add(1)
		return nil
	}

	entry, ok := value.(*CacheEntry)
	if !ok {
		c.entries.Delete(key)
		c.misses.Add(1)
		return nil
	}

	// Check if YAML has changed (invalidate immediately)
	if entry.YAMLHash != currentYAMLHash {
		c.entries.Delete(key)
		c.misses.Add(1)
		logDebug("Cache MISS (YAML changed): %s", key)
		return nil
	}

	// Check if TTL has expired
	if time.Since(entry.CreatedAt) > c.ttl {
		c.entries.Delete(key)
		c.misses.Add(1)
		logDebug("Cache MISS (expired): %s", key)
		return nil
	}

	c.hits.Add(1)
	logDebug("Cache HIT: %s (age: %v)", key, time.Since(entry.CreatedAt).Round(time.Second))
	return &entry.Data
}

// Set stores a component result in the cache
// Evicts oldest entries if cache exceeds maxEntries
func (c *ComponentCache) Set(reportID string, sectionIdx, componentIdx int, filters ReportFilters, result ComponentResult, yamlHash string) {
	key := c.buildCacheKey(reportID, sectionIdx, componentIdx, filters)

	entry := &CacheEntry{
		Data:      result,
		CreatedAt: time.Now(),
		YAMLHash:  yamlHash,
	}

	c.entries.Store(key, entry)
	logDebug("Cache SET: %s", key)

	// Evict oldest if over limit
	c.evictIfNeeded()
}

// evictIfNeeded removes oldest entries if cache exceeds maxEntries
func (c *ComponentCache) evictIfNeeded() {
	// Skip if no limit set
	if c.maxEntries <= 0 {
		return
	}

	// Count entries
	count := 0
	c.entries.Range(func(_, _ any) bool {
		count++
		return true
	})

	if count <= c.maxEntries {
		return
	}

	// Collect all entries with their ages
	type keyAge struct {
		key       string
		createdAt time.Time
	}
	var entries []keyAge

	c.entries.Range(func(key, value any) bool {
		entry, ok := value.(*CacheEntry)
		if !ok {
			return true
		}
		entries = append(entries, keyAge{key: key.(string), createdAt: entry.CreatedAt})
		return true
	})

	// Sort by age (oldest first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].createdAt.Before(entries[j].createdAt)
	})

	// Delete oldest entries to get back under limit
	toDelete := count - c.maxEntries
	for i := 0; i < toDelete && i < len(entries); i++ {
		c.entries.Delete(entries[i].key)
	}

	logDebug("Cache evicted %d oldest entries (was %d, max %d)", toDelete, count, c.maxEntries)
}

// InvalidateReport removes all cached entries for a specific report
func (c *ComponentCache) InvalidateReport(reportID string) {
	prefix := reportID + ":"
	count := 0

	c.entries.Range(func(key, value interface{}) bool {
		if strings.HasPrefix(key.(string), prefix) {
			c.entries.Delete(key)
			count++
		}
		return true
	})

	if count > 0 {
		logDebug("Cache invalidated %d entries for report: %s", count, reportID)
	}
}

// Clear removes all entries from the cache
func (c *ComponentCache) Clear() {
	count := 0
	c.entries.Range(func(key, value interface{}) bool {
		c.entries.Delete(key)
		count++
		return true
	})
	logDebug("Cache cleared: %d entries removed", count)
}

// Stats returns cache statistics for monitoring
type CacheStats struct {
	TotalEntries int         `json:"totalEntries"`
	MaxEntries   int         `json:"maxEntries"`
	TTL          string      `json:"ttl"`
	Hits         int64       `json:"hits"`
	Misses       int64       `json:"misses"`
	HitRate      float64     `json:"hitRate"`
	Entries      []CacheInfo `json:"entries,omitempty"`
}

type CacheInfo struct {
	Key       string `json:"key"`
	Age       string `json:"age"`
	ExpiresIn string `json:"expiresIn"`
}

// GetStats returns current cache statistics
func (c *ComponentCache) GetStats(includeEntries bool) CacheStats {
	hits := c.hits.Load()
	misses := c.misses.Load()
	total := hits + misses

	var hitRate float64
	if total > 0 {
		hitRate = float64(hits) / float64(total) * 100
	}

	stats := CacheStats{
		TTL:        c.ttl.String(),
		MaxEntries: c.maxEntries,
		Hits:       hits,
		Misses:     misses,
		HitRate:    hitRate,
	}

	c.entries.Range(func(key, value interface{}) bool {
		stats.TotalEntries++

		if includeEntries {
			entry, ok := value.(*CacheEntry)
			if !ok {
				return true
			}
			age := time.Since(entry.CreatedAt)
			expiresIn := c.ttl - age

			stats.Entries = append(stats.Entries, CacheInfo{
				Key:       key.(string),
				Age:       age.Round(time.Second).String(),
				ExpiresIn: expiresIn.Round(time.Second).String(),
			})
		}
		return true
	})

	return stats
}

// PersistedCache is the JSON-serializable format for saving cache to disk
type PersistedCache struct {
	SavedAt time.Time              `json:"savedAt"`
	Hits    int64                  `json:"hits"`
	Misses  int64                  `json:"misses"`
	Entries map[string]*CacheEntry `json:"entries"`
}

// SaveCache writes non-expired, non-error cache entries to data/cache.json atomically
func SaveCache() error {
	if err := os.MkdirAll("data", 0755); err != nil {
		return err
	}

	entries := make(map[string]*CacheEntry)
	componentCache.entries.Range(func(key, value any) bool {
		entry, ok := value.(*CacheEntry)
		if !ok {
			return true
		}
		// Skip expired entries
		if time.Since(entry.CreatedAt) > componentCache.ttl {
			return true
		}
		// Skip entries with errors (shouldn't exist in cache, but be defensive)
		if entry.Data.Error != nil {
			return true
		}
		entries[key.(string)] = entry
		return true
	})

	persisted := PersistedCache{
		SavedAt: time.Now(),
		Hits:    componentCache.hits.Load(),
		Misses:  componentCache.misses.Load(),
		Entries: entries,
	}

	data, err := json.Marshal(persisted)
	if err != nil {
		return err
	}

	tmpPath := cacheFilePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmpPath, cacheFilePath)
}

// LoadCache reads persisted cache entries from disk into the in-memory cache
func LoadCache() {
	data, err := os.ReadFile(cacheFilePath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Warning: Could not read cache file: %v", err)
		}
		return
	}

	var persisted PersistedCache
	if err := json.Unmarshal(data, &persisted); err != nil {
		log.Printf("Warning: Could not parse cache file, starting cold: %v", err)
		return
	}

	loaded := 0
	skipped := 0
	for key, entry := range persisted.Entries {
		if time.Since(entry.CreatedAt) > componentCache.ttl {
			skipped++
			continue
		}
		componentCache.entries.Store(key, entry)
		loaded++
	}

	componentCache.hits.Store(persisted.Hits)
	componentCache.misses.Store(persisted.Misses)

	componentCache.evictIfNeeded()
	log.Printf("Loaded %d cache entries from disk (%d expired, hits: %d, misses: %d, saved at %s)", loaded, skipped, persisted.Hits, persisted.Misses, persisted.SavedAt.Format("2006-01-02 15:04:05"))
}

// StartPeriodicCacheSave starts a goroutine that saves cache every 30 minutes.
// Close the returned channel to stop it.
func StartPeriodicCacheSave() chan struct{} {
	stop := make(chan struct{})
	ticker := time.NewTicker(30 * time.Minute)

	go func() {
		for {
			select {
			case <-ticker.C:
				if err := SaveCache(); err != nil {
					log.Printf("Periodic cache save failed: %v", err)
				} else {
					log.Println("Periodic cache save completed")
				}
			case <-stop:
				ticker.Stop()
				return
			}
		}
	}()

	return stop
}
