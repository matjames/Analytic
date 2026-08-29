package main

import (
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Report-level JSON response cache (full GET /api/report/... body).
// Opt-in via REPORT_CACHE_ENABLED=true. Separate from per-component cache.

const reportCacheKeySep = "\x1e" // record separator — not used in report IDs or fingerprints

var (
	fullReportCache         *fullReportResponseCache
	reportCacheAllowlistSet map[string]struct{} // empty = all reports allowed
)

func init() {
	fullReportCache = newFullReportResponseCacheFromEnv()
	reportCacheAllowlistSet = parseReportCacheAllowlist()
}

func parseReportCacheAllowlist() map[string]struct{} {
	raw := strings.TrimSpace(os.Getenv("REPORT_CACHE_ALLOWLIST"))
	if raw == "" {
		return nil
	}
	m := make(map[string]struct{})
	for _, id := range strings.Split(raw, ",") {
		id = strings.TrimSpace(id)
		if id != "" {
			m[id] = struct{}{}
		}
	}
	return m
}

type fullReportResponseCache struct {
	enabled    bool
	ttl        time.Duration
	maxEntries int
	maxBytes   int64
	entries    sync.Map // key string -> *fullReportResponseCacheEntry
	hits       atomic.Int64
	misses     atomic.Int64
}

type fullReportResponseCacheEntry struct {
	body      []byte
	createdAt time.Time
	yamlHash  string
}

func newFullReportResponseCacheFromEnv() *fullReportResponseCache {
	c := &fullReportResponseCache{
		ttl:        24 * time.Hour,
		maxEntries: 500,
		maxBytes:   10 * 1024 * 1024,
	}
	if strings.EqualFold(os.Getenv("REPORT_CACHE_ENABLED"), "true") {
		c.enabled = true
	}
	if s := os.Getenv("REPORT_CACHE_TTL"); s != "" {
		if d, err := time.ParseDuration(s); err == nil && d > 0 {
			c.ttl = d
		}
	}
	if s := os.Getenv("REPORT_CACHE_MAX_ENTRIES"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			c.maxEntries = n
		}
	}
	if s := os.Getenv("REPORT_CACHE_MAX_BYTES"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			c.maxBytes = int64(n)
		}
	}
	return c
}

// reportCacheEnabled returns whether report-level caching is turned on (env).
func reportCacheEnabled() bool {
	return fullReportCache != nil && fullReportCache.enabled
}

// reportCacheAllowed returns true if this report ID may use report-level cache.
func reportCacheAllowed(reportID string) bool {
	if reportCacheAllowlistSet == nil {
		return true
	}
	_, ok := reportCacheAllowlistSet[reportID]
	return ok
}

// buildReportCacheKey is stable for a given report definition hash and filters.
func buildReportCacheKey(reportID, yamlHash string, filters ReportFilters) string {
	return reportID + reportCacheKeySep + yamlHash + reportCacheKeySep + FilterFingerprint(filters)
}

// Get returns cached JSON body if valid; otherwise nil.
func (c *fullReportResponseCache) Get(key, currentYAMLHash string) []byte {
	if c == nil || !c.enabled {
		return nil
	}

	v, ok := c.entries.Load(key)
	if !ok {
		c.misses.Add(1)
		logDebug("Report cache MISS (missing): %s", key)
		return nil
	}
	entry, ok := v.(*fullReportResponseCacheEntry)
	if !ok {
		c.entries.Delete(key)
		c.misses.Add(1)
		return nil
	}
	if entry.yamlHash != currentYAMLHash {
		c.entries.Delete(key)
		c.misses.Add(1)
		logDebug("Report cache MISS (YAML changed): %s", key)
		return nil
	}
	if time.Since(entry.createdAt) > c.ttl {
		c.entries.Delete(key)
		c.misses.Add(1)
		logDebug("Report cache MISS (expired): %s", key)
		return nil
	}
	c.hits.Add(1)
	logDebug("Report cache HIT: %s (age: %v)", key, time.Since(entry.createdAt).Round(time.Second))
	return entry.body
}

// Set stores a full JSON response. Skips if over max body size.
func (c *fullReportResponseCache) Set(key string, body []byte, yamlHash string) {
	if c == nil || !c.enabled {
		return
	}
	if int64(len(body)) > c.maxBytes {
		logDebug("Report cache SET skipped (body %d > max %d)", len(body), c.maxBytes)
		return
	}
	c.entries.Store(key, &fullReportResponseCacheEntry{
		body:      append([]byte(nil), body...),
		createdAt: time.Now(),
		yamlHash:  yamlHash,
	})
	logDebug("Report cache SET: %s (%d bytes)", key, len(body))
	c.evictIfNeeded()
}

func (c *fullReportResponseCache) evictIfNeeded() {
	if c.maxEntries <= 0 {
		return
	}
	count := 0
	c.entries.Range(func(_, _ any) bool {
		count++
		return true
	})
	if count <= c.maxEntries {
		return
	}
	type keyAge struct {
		key       string
		createdAt time.Time
	}
	var rows []keyAge
	c.entries.Range(func(key, value any) bool {
		entry, ok := value.(*fullReportResponseCacheEntry)
		if !ok {
			return true
		}
		rows = append(rows, keyAge{key: key.(string), createdAt: entry.createdAt})
		return true
	})
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].createdAt.Before(rows[j].createdAt)
	})
	toDelete := count - c.maxEntries
	for i := 0; i < toDelete && i < len(rows); i++ {
		c.entries.Delete(rows[i].key)
	}
	if toDelete > 0 {
		logDebug("Report cache evicted %d oldest entries (was %d, max %d)", toDelete, count, c.maxEntries)
	}
}

// Clear removes all report-level cache entries.
func (c *fullReportResponseCache) Clear() {
	if c == nil {
		return
	}
	n := 0
	c.entries.Range(func(key, _ any) bool {
		c.entries.Delete(key)
		n++
		return true
	})
	logDebug("Report cache cleared: %d entries", n)
}

// ReportCacheStats is returned alongside component cache stats for monitoring.
type ReportCacheStats struct {
	Enabled      bool    `json:"enabled"`
	TTL          string  `json:"ttl"`
	MaxEntries   int     `json:"maxEntries"`
	MaxBytes     int64   `json:"maxBytes"`
	TotalEntries int     `json:"totalEntries"`
	Hits         int64   `json:"hits"`
	Misses       int64   `json:"misses"`
	HitRate      float64 `json:"hitRate"`
}

func (c *fullReportResponseCache) stats() ReportCacheStats {
	if c == nil {
		return ReportCacheStats{}
	}
	hits := c.hits.Load()
	misses := c.misses.Load()
	total := hits + misses
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(hits) / float64(total) * 100
	}
	entries := 0
	c.entries.Range(func(_, _ any) bool {
		entries++
		return true
	})
	return ReportCacheStats{
		Enabled:      c.enabled,
		TTL:          c.ttl.String(),
		MaxEntries:   c.maxEntries,
		MaxBytes:     c.maxBytes,
		TotalEntries: entries,
		Hits:         hits,
		Misses:       misses,
		HitRate:      hitRate,
	}
}

// shouldStoreReportCache is true when the assembled report is safe to cache as a full JSON blob.
func shouldStoreReportCache(report *Report, reportID string, jsonLen int) bool {
	if !reportCacheEnabled() || !reportCacheAllowed(reportID) {
		return false
	}
	if report == nil || report.PartialFailure {
		return false
	}
	if fullReportCache == nil {
		return false
	}
	if int64(jsonLen) > fullReportCache.maxBytes {
		return false
	}
	return true
}
