package main

import (
	"encoding/json"
	"log"
	"maps"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"
)

// RequestMetric stores data about a single API request
type RequestMetric struct {
	ReportID   string            `json:"reportId"`
	Timestamp  time.Time         `json:"timestamp"`
	DurationMs int64             `json:"durationMs"`
	Filters    map[string]string `json:"filters"`
	StatusCode int               `json:"statusCode"`
	IsError    bool              `json:"isError"`
	Username   string            `json:"username"`
	Email      string            `json:"email"`
}

// ReportStats holds aggregated statistics for a single report
type ReportStats struct {
	ReportID      string    `json:"reportId"`
	RequestCount  int       `json:"requestCount"`
	ErrorCount    int       `json:"errorCount"`
	AvgDurationMs float64   `json:"avgDurationMs"`
	MaxDurationMs int64     `json:"maxDurationMs"`
	LastAccessed  time.Time `json:"lastAccessed"`
}

// UserStats holds aggregated statistics for a single user
type UserStats struct {
	Username     string         `json:"username"`
	Email        string         `json:"email"`
	RequestCount int            `json:"requestCount"`
	LastSeen     time.Time      `json:"lastSeen"`
	TopReports   map[string]int `json:"topReports"`
}

// DBPoolStats holds database connection pool statistics
type DBPoolStats struct {
	MaxOpenConnections int   `json:"maxOpenConnections"` // Configured max open connections
	OpenConnections    int   `json:"openConnections"`    // Current open connections
	InUse              int   `json:"inUse"`              // Connections currently in use
	Idle               int   `json:"idle"`               // Idle connections
	WaitCount          int64 `json:"waitCount"`          // Total connections waited for
	WaitDurationMs     int64 `json:"waitDurationMs"`     // Total time blocked waiting (ms)
	MaxIdleClosed      int64 `json:"maxIdleClosed"`      // Connections closed due to max idle
	MaxLifetimeClosed  int64 `json:"maxLifetimeClosed"`  // Connections closed due to max lifetime
}

// PDFStats holds aggregated PDF download statistics
type PDFStats struct {
	TotalDownloads int            `json:"totalDownloads"`
	TotalErrors    int            `json:"totalErrors"`
	ErrorRate      float64        `json:"errorRate"`
	AvgDurationMs  float64        `json:"avgDurationMs"`
	MaxDurationMs  int64          `json:"maxDurationMs"`
	ByReport       map[string]int `json:"byReport"`
}

// CSVStats holds aggregated CSV download statistics
type CSVStats struct {
	TotalDownloads int            `json:"totalDownloads"`
	TotalRows      int            `json:"totalRows"`
	ByReport       map[string]int `json:"byReport"`
}

// ComponentFailureStats holds aggregated component failure statistics
type ComponentFailureStats struct {
	TotalExecuted    int                `json:"totalExecuted"`
	TotalFailed      int                `json:"totalFailed"`
	FailureRate      float64            `json:"failureRate"`      // percentage
	FailuresByReport map[string]int     `json:"failuresByReport"` // reportID -> fail count
	RecentFailures   []ComponentFailure `json:"recentFailures"`   // last 50
}

// ComponentFailure records a single component that failed to load data
type ComponentFailure struct {
	Timestamp time.Time `json:"timestamp"`
	ReportID  string    `json:"reportId"`
	Title     string    `json:"title"`
	Error     string    `json:"error"`
}

// ChatStats holds aggregated Ask module statistics
type ChatStats struct {
	TotalQuestions     int               `json:"totalQuestions"`
	TotalExecutions    int               `json:"totalExecutions"`
	TotalErrors        int               `json:"totalErrors"`
	AvgAskDurationMs   float64           `json:"avgAskDurationMs"`
	AvgExecDurationMs  float64           `json:"avgExecDurationMs"`
	ByTable            map[string]int    `json:"byTable"`
	RecentInteractions []ChatInteraction `json:"recentInteractions"`
}

// ChatInteraction records a single Ask module interaction for audit trail
type ChatInteraction struct {
	Timestamp   time.Time `json:"timestamp"`
	Question    string    `json:"question"`
	Correction  string    `json:"correction,omitempty"`
	Table       string    `json:"table"`
	Confidence  string    `json:"confidence"`
	Phase       string    `json:"phase"`              // "ask" or "execute"
	Category    string    `json:"category,omitempty"` // "query", "out_of_scope", "security"
	DurationMs  int64     `json:"durationMs"`
	IsError     bool      `json:"isError"`
	ErrorMsg    string    `json:"errorMsg,omitempty"`
	RowCount    int       `json:"rowCount,omitempty"`
	Username    string    `json:"username,omitempty"`
	Provider    string    `json:"provider,omitempty"`
	AIResponse  string    `json:"aiResponse,omitempty"`
	Explanation string    `json:"explanation,omitempty"`
	SQL         string    `json:"sql,omitempty"`
	Summary     string    `json:"summary,omitempty"`
}

// QueryMetric represents a single notable query execution (slow or large)
type QueryMetric struct {
	Timestamp  time.Time `json:"timestamp"`
	Table      string    `json:"table"`
	DurationMs float64   `json:"durationMs"`
	RowCount   int       `json:"rowCount"`
	IsError    bool      `json:"isError"`
	ErrorMsg   string    `json:"errorMsg,omitempty"`
}

// QueryTableStats holds aggregated query statistics for a single table
type QueryTableStats struct {
	Table          string  `json:"table"`
	ExecutionCount int     `json:"executionCount"`
	ErrorCount     int     `json:"errorCount"`
	AvgDurationMs  float64 `json:"avgDurationMs"`
	MaxDurationMs  float64 `json:"maxDurationMs"`
	AvgRows        int     `json:"avgRows"`
	MaxRows        int     `json:"maxRows"`
}

// QueryPerformanceStats holds all query performance metrics for the API response
type QueryPerformanceStats struct {
	TotalExecuted int               `json:"totalExecuted"`
	TotalErrors   int               `json:"totalErrors"`
	ErrorRate     float64           `json:"errorRate"`
	ByTable       []QueryTableStats `json:"byTable"`
	SlowQueries   []QueryMetric     `json:"slowQueries"`
	LargeResults  []QueryMetric     `json:"largeResults"`
}

// queryTableAccum holds per-table accumulators for computing averages
type queryTableAccum struct {
	Count  int     `json:"count"`
	Errors int     `json:"errors"`
	DurSum float64 `json:"durSum"`
	DurMax float64 `json:"durMax"`
	RowSum int64   `json:"rowSum"`
	RowMax int     `json:"rowMax"`
}

// MetricsResponse is the API response structure
type MetricsResponse struct {
	ServerStartTime       time.Time             `json:"serverStartTime"`
	TotalRequests         int                   `json:"totalRequests"`
	TotalErrors           int                   `json:"totalErrors"`
	ErrorRate             float64               `json:"errorRate"`
	ActiveRequests        int                   `json:"activeRequests"`
	PeakConcurrent        int                   `json:"peakConcurrent"`
	UniqueReports         int                   `json:"uniqueReports"`
	TotalReports          int                   `json:"totalReports"`
	ReportStats           []ReportStats         `json:"reportStats"`
	TopReports            []ReportStats         `json:"topReports"`
	LeastRequestedReports []ReportStats         `json:"leastRequestedReports"`
	RecentRequests        []RequestMetric       `json:"recentRequests"`
	Cache                 CacheStats            `json:"cache"`
	DBPool                DBPoolStats           `json:"dbPool"`
	HourlyHits            map[string]int        `json:"hourlyHits"`
	DailyHits             map[string]int        `json:"dailyHits"`
	DateHits              map[string]int        `json:"dateHits"`
	DurationBuckets       map[string]int        `json:"durationBuckets"`
	PDFStats              PDFStats              `json:"pdfStats"`
	CSVStats              CSVStats              `json:"csvStats"`
	ChatStats             ChatStats             `json:"chatStats"`
	ComponentStats        ComponentFailureStats `json:"componentStats"`
	QueryStats            QueryPerformanceStats `json:"queryStats"`
	UserStats             []UserStats           `json:"userStats"`
	UIStats               UIEventStats          `json:"uiStats"`
}

// MetricsStore holds all metrics data in memory
type MetricsStore struct {
	sync.RWMutex
	serverStartTime time.Time
	requests        []RequestMetric      // Ring buffer of recent requests
	maxRequests     int                  // Max requests to keep in buffer
	reportCounts    map[string]int       // reportID -> request count
	reportErrors    map[string]int       // reportID -> error count
	reportDurSum    map[string]int64     // reportID -> cumulative duration sum
	reportDurCount  map[string]int       // reportID -> count of requests with durations
	reportDurMax    map[string]int64     // reportID -> max duration
	lastAccessed    map[string]time.Time // reportID -> last access time
	hourlyHits      map[int]int          // hour 0-23 -> count
	dailyHits       map[string]int       // weekday name -> count
	dateHits        map[string]int       // calendar date "2006-01-02" -> count
	durationBuckets [6]int               // <100ms, 100-500ms, 500ms-1s, 1-3s, 3-10s, >10s
	activeRequests  int                  // current in-flight requests
	peakConcurrent  int                  // max concurrent seen
	pdfDownloads    int
	pdfErrors       int
	pdfDurSum       int64
	pdfDurMax       int64
	pdfByReport     map[string]int
	csvDownloads    int
	csvTotalRows    int
	csvByReport     map[string]int
	userStats       map[string]*UserStats // username -> stats
	// Component failure metrics
	componentExecTotal    int
	componentFailTotal    int
	componentFailByReport map[string]int
	componentFailures     []ComponentFailure // ring buffer
	componentMaxFailures  int
	// Query performance metrics
	queryExecTotal  int
	queryErrorTotal int
	queryByTable    map[string]*queryTableAccum
	slowQueries     []QueryMetric // ring buffer of slowest queries (>1s)
	maxSlowQueries  int
	largeResults    []QueryMetric // ring buffer of largest result sets (>5000 rows)
	maxLargeResults int
	// Chat (Ask module) metrics
	chatQuestions       int
	chatExecutions      int
	chatErrors          int
	chatAskDurSum       int64
	chatAskDurCount     int
	chatExecDurSum      int64
	chatExecDurCount    int
	chatByTable         map[string]int
	chatInteractions    []ChatInteraction // ring buffer of recent interactions
	chatMaxInteractions int
}

// PersistedMetrics is the JSON-serializable format for saving metrics to disk
type PersistedMetrics struct {
	SavedAt         time.Time             `json:"savedAt"`
	ReportCounts    map[string]int        `json:"reportCounts"`
	ReportErrors    map[string]int        `json:"reportErrors"`
	ReportDurSum    map[string]int64      `json:"reportDurSum"`
	ReportDurCount  map[string]int        `json:"reportDurCount"`
	ReportDurMax    map[string]int64      `json:"reportDurMax"`
	LastAccessed    map[string]time.Time  `json:"lastAccessed"`
	HourlyHits      map[int]int           `json:"hourlyHits"`
	DailyHits       map[string]int        `json:"dailyHits"`
	DateHits        map[string]int        `json:"dateHits"`
	DurationBuckets [6]int                `json:"durationBuckets"`
	PeakConcurrent  int                   `json:"peakConcurrent"`
	PDFDownloads    int                   `json:"pdfDownloads"`
	PDFErrors       int                   `json:"pdfErrors"`
	PDFDurSum       int64                 `json:"pdfDurSum"`
	PDFDurMax       int64                 `json:"pdfDurMax"`
	PDFByReport     map[string]int        `json:"pdfByReport"`
	CSVDownloads    int                   `json:"csvDownloads,omitempty"`
	CSVTotalRows    int                   `json:"csvTotalRows,omitempty"`
	CSVByReport     map[string]int        `json:"csvByReport,omitempty"`
	UserStats       map[string]*UserStats `json:"userStats,omitempty"`
	// Component failure metrics
	ComponentExecTotal    int            `json:"componentExecTotal,omitempty"`
	ComponentFailTotal    int            `json:"componentFailTotal,omitempty"`
	ComponentFailByReport map[string]int `json:"componentFailByReport,omitempty"`
	// Query performance metrics
	QueryExecTotal  int                         `json:"queryExecTotal,omitempty"`
	QueryErrorTotal int                         `json:"queryErrorTotal,omitempty"`
	QueryByTable    map[string]*queryTableAccum `json:"queryByTable,omitempty"`
	// Chat (Ask module) metrics
	ChatQuestions    int            `json:"chatQuestions,omitempty"`
	ChatExecutions   int            `json:"chatExecutions,omitempty"`
	ChatErrors       int            `json:"chatErrors,omitempty"`
	ChatAskDurSum    int64          `json:"chatAskDurSum,omitempty"`
	ChatAskDurCount  int            `json:"chatAskDurCount,omitempty"`
	ChatExecDurSum   int64          `json:"chatExecDurSum,omitempty"`
	ChatExecDurCount int            `json:"chatExecDurCount,omitempty"`
	ChatByTable      map[string]int `json:"chatByTable,omitempty"`
}

const metricsFilePath = "data/metrics.json"

// Global metrics store
var metricsStore *MetricsStore

// InitMetrics initializes the metrics store
func InitMetrics() {
	metricsStore = &MetricsStore{
		serverStartTime:       time.Now(),
		requests:              make([]RequestMetric, 0, 1000),
		maxRequests:           1000,
		reportCounts:          make(map[string]int),
		reportErrors:          make(map[string]int),
		reportDurSum:          make(map[string]int64),
		reportDurCount:        make(map[string]int),
		reportDurMax:          make(map[string]int64),
		lastAccessed:          make(map[string]time.Time),
		hourlyHits:            make(map[int]int),
		dailyHits:             make(map[string]int),
		dateHits:              make(map[string]int),
		pdfByReport:           make(map[string]int),
		csvByReport:           make(map[string]int),
		userStats:             make(map[string]*UserStats),
		chatByTable:           make(map[string]int),
		chatInteractions:      make([]ChatInteraction, 0, 200),
		chatMaxInteractions:   200,
		componentFailByReport: make(map[string]int),
		componentFailures:     make([]ComponentFailure, 0, 50),
		componentMaxFailures:  50,
		queryByTable:          make(map[string]*queryTableAccum),
		slowQueries:           make([]QueryMetric, 0, 20),
		maxSlowQueries:        20,
		largeResults:          make([]QueryMetric, 0, 20),
		maxLargeResults:       20,
	}
	InitUIEvents()
}

// durationBucketIndex returns the bucket index for a given duration in ms
func durationBucketIndex(ms int64) int {
	switch {
	case ms < 100:
		return 0
	case ms < 500:
		return 1
	case ms < 1000:
		return 2
	case ms < 3000:
		return 3
	case ms < 10000:
		return 4
	default:
		return 5
	}
}

// RecordRequest records a new request metric
func RecordRequest(metric RequestMetric) {
	if metricsStore == nil {
		return
	}

	metricsStore.Lock()
	defer metricsStore.Unlock()

	// Add to ring buffer
	if len(metricsStore.requests) >= metricsStore.maxRequests {
		metricsStore.requests = metricsStore.requests[1:]
	}
	metricsStore.requests = append(metricsStore.requests, metric)

	// Update aggregates
	metricsStore.reportCounts[metric.ReportID]++
	if metric.IsError {
		metricsStore.reportErrors[metric.ReportID]++
	}
	metricsStore.reportDurSum[metric.ReportID] += metric.DurationMs
	metricsStore.reportDurCount[metric.ReportID]++
	if metric.DurationMs > metricsStore.reportDurMax[metric.ReportID] {
		metricsStore.reportDurMax[metric.ReportID] = metric.DurationMs
	}
	metricsStore.lastAccessed[metric.ReportID] = metric.Timestamp

	// Hourly hits
	metricsStore.hourlyHits[metric.Timestamp.Hour()]++

	// Daily hits
	metricsStore.dailyHits[metric.Timestamp.Weekday().String()]++

	// Date hits (calendar date for adoption tracking)
	metricsStore.dateHits[metric.Timestamp.Format("2006-01-02")]++

	// Duration bucket
	metricsStore.durationBuckets[durationBucketIndex(metric.DurationMs)]++

	// Per-user tracking (skip when auth is off / no username)
	if metric.Username != "" {
		us, ok := metricsStore.userStats[metric.Username]
		if !ok {
			us = &UserStats{
				Username:   metric.Username,
				Email:      metric.Email,
				TopReports: make(map[string]int),
			}
			metricsStore.userStats[metric.Username] = us
		}
		us.RequestCount++
		us.LastSeen = metric.Timestamp
		if metric.Email != "" {
			us.Email = metric.Email
		}
		if metric.ReportID != "" {
			us.TopReports[metric.ReportID]++
		}
	}
}

// sanitizeMetricsReportID collapses invalid report IDs to a single sentinel.
// Download endpoints accept a client-supplied reportId, so the same rule the
// request middleware applies must hold here: attacker-controlled strings never
// reach data/metrics.json and map cardinality stays bounded.
func sanitizeMetricsReportID(reportID string) string {
	if ValidateReportID(reportID) != nil {
		return "_invalid"
	}
	return reportID
}

// RecordPDFDownload records a PDF download attempt
func RecordPDFDownload(reportID string, durationMs int64, isError bool) {
	if metricsStore == nil {
		return
	}

	metricsStore.Lock()
	defer metricsStore.Unlock()

	metricsStore.pdfDownloads++
	if isError {
		metricsStore.pdfErrors++
	}
	metricsStore.pdfDurSum += durationMs
	if durationMs > metricsStore.pdfDurMax {
		metricsStore.pdfDurMax = durationMs
	}
	if reportID != "" {
		metricsStore.pdfByReport[sanitizeMetricsReportID(reportID)]++
	}
}

// RecordCSVDownload records a CSV download event
func RecordCSVDownload(reportID string, rowCount int) {
	if metricsStore == nil {
		return
	}

	metricsStore.Lock()
	defer metricsStore.Unlock()

	metricsStore.csvDownloads++
	metricsStore.csvTotalRows += rowCount
	if reportID != "" {
		metricsStore.csvByReport[sanitizeMetricsReportID(reportID)]++
	}
}

// RecordChatInteraction records an Ask module interaction
func RecordChatInteraction(interaction ChatInteraction) {
	if metricsStore == nil {
		return
	}

	metricsStore.Lock()
	defer metricsStore.Unlock()

	// Update counters
	if interaction.Phase == "ask" {
		metricsStore.chatQuestions++
		metricsStore.chatAskDurSum += interaction.DurationMs
		metricsStore.chatAskDurCount++
	} else if interaction.Phase == "execute" {
		metricsStore.chatExecutions++
		metricsStore.chatExecDurSum += interaction.DurationMs
		metricsStore.chatExecDurCount++
	}
	if interaction.IsError {
		metricsStore.chatErrors++
	}
	if interaction.Table != "" {
		metricsStore.chatByTable[interaction.Table]++
	}

	// Add to ring buffer
	if len(metricsStore.chatInteractions) >= metricsStore.chatMaxInteractions {
		metricsStore.chatInteractions = metricsStore.chatInteractions[1:]
	}
	metricsStore.chatInteractions = append(metricsStore.chatInteractions, interaction)
}

// RecordComponentExecution records component execution results for a report
func RecordComponentExecution(reportID string, totalComponents int, failed []FailedComponent) {
	if metricsStore == nil {
		return
	}

	metricsStore.Lock()
	defer metricsStore.Unlock()

	metricsStore.componentExecTotal += totalComponents
	metricsStore.componentFailTotal += len(failed)

	if len(failed) > 0 {
		metricsStore.componentFailByReport[reportID] += len(failed)
	}

	now := time.Now()
	for _, f := range failed {
		entry := ComponentFailure{
			Timestamp: now,
			ReportID:  reportID,
			Title:     f.Title,
			Error:     f.Error,
		}
		if len(metricsStore.componentFailures) >= metricsStore.componentMaxFailures {
			metricsStore.componentFailures = metricsStore.componentFailures[1:]
		}
		metricsStore.componentFailures = append(metricsStore.componentFailures, entry)
	}
}

// RecordQueryExecution records a SQL query execution for performance tracking
func RecordQueryExecution(table string, durationMs float64, rowCount int, err error) {
	if metricsStore == nil {
		return
	}

	metricsStore.Lock()
	defer metricsStore.Unlock()

	metricsStore.queryExecTotal++

	isError := err != nil
	if isError {
		metricsStore.queryErrorTotal++
	}

	// Update per-table accumulators
	if table != "" {
		accum, ok := metricsStore.queryByTable[table]
		if !ok {
			accum = &queryTableAccum{}
			metricsStore.queryByTable[table] = accum
		}
		accum.Count++
		if isError {
			accum.Errors++
		}
		accum.DurSum += durationMs
		if durationMs > accum.DurMax {
			accum.DurMax = durationMs
		}
		accum.RowSum += int64(rowCount)
		if rowCount > accum.RowMax {
			accum.RowMax = rowCount
		}
	}

	now := time.Now()

	// Track slow queries (>1 second)
	if durationMs > 1000 {
		entry := QueryMetric{
			Timestamp:  now,
			Table:      table,
			DurationMs: durationMs,
			RowCount:   rowCount,
			IsError:    isError,
		}
		if isError {
			entry.ErrorMsg = err.Error()
		}
		if len(metricsStore.slowQueries) >= metricsStore.maxSlowQueries {
			metricsStore.slowQueries = metricsStore.slowQueries[1:]
		}
		metricsStore.slowQueries = append(metricsStore.slowQueries, entry)
	}

	// Track large result sets (>5000 rows)
	if rowCount > 5000 {
		entry := QueryMetric{
			Timestamp:  now,
			Table:      table,
			DurationMs: durationMs,
			RowCount:   rowCount,
			IsError:    isError,
		}
		if isError {
			entry.ErrorMsg = err.Error()
		}
		if len(metricsStore.largeResults) >= metricsStore.maxLargeResults {
			metricsStore.largeResults = metricsStore.largeResults[1:]
		}
		metricsStore.largeResults = append(metricsStore.largeResults, entry)
	}
}

// IncrementActiveRequests increments the active request counter and updates peak
func IncrementActiveRequests() {
	if metricsStore == nil {
		return
	}
	metricsStore.Lock()
	defer metricsStore.Unlock()
	metricsStore.activeRequests++
	if metricsStore.activeRequests > metricsStore.peakConcurrent {
		metricsStore.peakConcurrent = metricsStore.activeRequests
	}
}

// DecrementActiveRequests decrements the active request counter
func DecrementActiveRequests() {
	if metricsStore == nil {
		return
	}
	metricsStore.Lock()
	defer metricsStore.Unlock()
	metricsStore.activeRequests--
}

// getDBPoolStats returns current database connection pool statistics
func getDBPoolStats() DBPoolStats {
	if DB == nil {
		return DBPoolStats{}
	}
	stats := DB.Stats()
	return DBPoolStats{
		MaxOpenConnections: stats.MaxOpenConnections,
		OpenConnections:    stats.OpenConnections,
		InUse:              stats.InUse,
		Idle:               stats.Idle,
		WaitCount:          stats.WaitCount,
		WaitDurationMs:     stats.WaitDuration.Milliseconds(),
		MaxIdleClosed:      stats.MaxIdleClosed,
		MaxLifetimeClosed:  stats.MaxLifetimeClosed,
	}
}

// GetMetrics returns the current metrics
func GetMetrics() MetricsResponse {
	if metricsStore == nil {
		return MetricsResponse{
			ServerStartTime:       time.Now(),
			ReportStats:           []ReportStats{},
			TopReports:            []ReportStats{},
			LeastRequestedReports: []ReportStats{},
			RecentRequests:        []RequestMetric{},
			Cache:                 componentCache.GetStats(false),
			DBPool:                getDBPoolStats(),
			HourlyHits:            map[string]int{},
			DailyHits:             map[string]int{},
			DateHits:              map[string]int{},
			DurationBuckets:       map[string]int{},
			UserStats:             []UserStats{},
			UIStats:               GetUIStats(),
		}
	}

	metricsStore.RLock()
	defer metricsStore.RUnlock()

	// Get current valid reports to filter out stale entries
	allReports := GetAllReports()
	validReportIDs := make(map[string]bool, len(allReports.Reports))
	for _, r := range allReports.Reports {
		validReportIDs[r.ID] = true
	}

	// Calculate report stats (only for reports that still exist)
	reportStats := make([]ReportStats, 0, len(metricsStore.reportCounts))
	
	// Calculate total historical requests to match the adoption graph
	totalRequests := 0
	for _, count := range metricsStore.dateHits {
		totalRequests += count
	}
	
	totalErrors := 0

	for reportID, count := range metricsStore.reportCounts {
		// Skip stale report IDs that no longer exist
		if !validReportIDs[reportID] {
			continue
		}
		errorCount := metricsStore.reportErrors[reportID]
		totalErrors += errorCount

		var avgDuration float64
		durCount := metricsStore.reportDurCount[reportID]
		if durCount > 0 {
			avgDuration = float64(metricsStore.reportDurSum[reportID]) / float64(durCount)
		}

		reportStats = append(reportStats, ReportStats{
			ReportID:      reportID,
			RequestCount:  count,
			ErrorCount:    errorCount,
			AvgDurationMs: avgDuration,
			MaxDurationMs: metricsStore.reportDurMax[reportID],
			LastAccessed:  metricsStore.lastAccessed[reportID],
		})
	}

	// Get recent requests (last 50 for the response)
	recentRequests := make([]RequestMetric, 0, 50)
	startIdx := len(metricsStore.requests) - 50
	if startIdx < 0 {
		startIdx = 0
	}
	for i := len(metricsStore.requests) - 1; i >= startIdx; i-- {
		recentRequests = append(recentRequests, metricsStore.requests[i])
	}

	// Build hourly hits with string keys for JSON
	hourlyHits := make(map[string]int, len(metricsStore.hourlyHits))
	for hour, count := range metricsStore.hourlyHits {
		h := hour
		key := ""
		if h < 10 {
			key = "0"
		}
		key += intToStr(h) + ":00"
		hourlyHits[key] = count
	}

	// Build duration buckets with labels
	bucketLabels := [6]string{"<100ms", "100-500ms", "500ms-1s", "1-3s", "3-10s", ">10s"}
	durationBuckets := make(map[string]int, 6)
	for i, label := range bucketLabels {
		durationBuckets[label] = metricsStore.durationBuckets[i]
	}

	// Copy daily hits, date hits, and filter usage
	dailyHits := make(map[string]int, len(metricsStore.dailyHits))
	for k, v := range metricsStore.dailyHits {
		dailyHits[k] = v
	}
	dateHits := make(map[string]int, len(metricsStore.dateHits))
	for k, v := range metricsStore.dateHits {
		dateHits[k] = v
	}
	// Calculate error rate as percentage
	var errorRate float64
	if totalRequests > 0 {
		errorRate = float64(totalErrors) / float64(totalRequests) * 100
	}

	byDesc := append([]ReportStats(nil), reportStats...)
	sort.Slice(byDesc, func(i, j int) bool {
		if byDesc[i].RequestCount != byDesc[j].RequestCount {
			return byDesc[i].RequestCount > byDesc[j].RequestCount
		}
		return byDesc[i].ReportID < byDesc[j].ReportID
	})
	topReports := byDesc
	if len(topReports) > 10 {
		topReports = topReports[:10]
	}

	byAsc := append([]ReportStats(nil), reportStats...)
	sort.Slice(byAsc, func(i, j int) bool {
		if byAsc[i].RequestCount != byAsc[j].RequestCount {
			return byAsc[i].RequestCount < byAsc[j].RequestCount
		}
		return byAsc[i].ReportID < byAsc[j].ReportID
	})
	leastRequested := byAsc
	if len(leastRequested) > 10 {
		leastRequested = leastRequested[:10]
	}

	// Total available reports count (already fetched above)
	totalReportsCount := len(allReports.Reports)

	// Build sorted user stats (by request count desc)
	userStatsList := make([]UserStats, 0, len(metricsStore.userStats))
	for _, us := range metricsStore.userStats {
		topReports := make(map[string]int, len(us.TopReports))
		for k, v := range us.TopReports {
			topReports[k] = v
		}
		userStatsList = append(userStatsList, UserStats{
			Username:     us.Username,
			Email:        us.Email,
			RequestCount: us.RequestCount,
			LastSeen:     us.LastSeen,
			TopReports:   topReports,
		})
	}
	sort.Slice(userStatsList, func(i, j int) bool {
		return userStatsList[i].RequestCount > userStatsList[j].RequestCount
	})

	// Compute PDF stats
	pdfStats := PDFStats{
		TotalDownloads: metricsStore.pdfDownloads,
		TotalErrors:    metricsStore.pdfErrors,
		MaxDurationMs:  metricsStore.pdfDurMax,
		ByReport:       maps.Clone(metricsStore.pdfByReport),
	}
	if metricsStore.pdfDownloads > 0 {
		successCount := metricsStore.pdfDownloads - metricsStore.pdfErrors
		if successCount > 0 {
			pdfStats.AvgDurationMs = float64(metricsStore.pdfDurSum) / float64(successCount)
		}
		pdfStats.ErrorRate = float64(metricsStore.pdfErrors) / float64(metricsStore.pdfDownloads) * 100
	}

	// Compute CSV stats
	csvStats := CSVStats{
		TotalDownloads: metricsStore.csvDownloads,
		TotalRows:      metricsStore.csvTotalRows,
		ByReport:       maps.Clone(metricsStore.csvByReport),
	}

	// Compute Chat stats
	chatStats := ChatStats{
		TotalQuestions:  metricsStore.chatQuestions,
		TotalExecutions: metricsStore.chatExecutions,
		TotalErrors:     metricsStore.chatErrors,
		ByTable:         maps.Clone(metricsStore.chatByTable),
	}
	if metricsStore.chatAskDurCount > 0 {
		chatStats.AvgAskDurationMs = float64(metricsStore.chatAskDurSum) / float64(metricsStore.chatAskDurCount)
	}
	if metricsStore.chatExecDurCount > 0 {
		chatStats.AvgExecDurationMs = float64(metricsStore.chatExecDurSum) / float64(metricsStore.chatExecDurCount)
	}
	// Copy recent interactions (last 50 for the response)
	recentChat := make([]ChatInteraction, 0, 50)
	startChat := len(metricsStore.chatInteractions) - 50
	if startChat < 0 {
		startChat = 0
	}
	for i := len(metricsStore.chatInteractions) - 1; i >= startChat; i-- {
		recentChat = append(recentChat, metricsStore.chatInteractions[i])
	}
	chatStats.RecentInteractions = recentChat

	// Compute Component failure stats
	componentStats := ComponentFailureStats{
		TotalExecuted:    metricsStore.componentExecTotal,
		TotalFailed:      metricsStore.componentFailTotal,
		FailuresByReport: maps.Clone(metricsStore.componentFailByReport),
	}
	if metricsStore.componentExecTotal > 0 {
		componentStats.FailureRate = float64(metricsStore.componentFailTotal) / float64(metricsStore.componentExecTotal) * 100
	}
	// Copy recent failures (newest first)
	recentFail := make([]ComponentFailure, 0, 50)
	startFail := len(metricsStore.componentFailures) - 50
	if startFail < 0 {
		startFail = 0
	}
	for i := len(metricsStore.componentFailures) - 1; i >= startFail; i-- {
		recentFail = append(recentFail, metricsStore.componentFailures[i])
	}
	componentStats.RecentFailures = recentFail

	// Compute Query performance stats
	queryStats := QueryPerformanceStats{
		TotalExecuted: metricsStore.queryExecTotal,
		TotalErrors:   metricsStore.queryErrorTotal,
		ByTable:       make([]QueryTableStats, 0, len(metricsStore.queryByTable)),
		SlowQueries:   make([]QueryMetric, 0, len(metricsStore.slowQueries)),
		LargeResults:  make([]QueryMetric, 0, len(metricsStore.largeResults)),
	}
	if metricsStore.queryExecTotal > 0 {
		queryStats.ErrorRate = float64(metricsStore.queryErrorTotal) / float64(metricsStore.queryExecTotal) * 100
	}
	for table, accum := range metricsStore.queryByTable {
		ts := QueryTableStats{
			Table:          table,
			ExecutionCount: accum.Count,
			ErrorCount:     accum.Errors,
			MaxDurationMs:  accum.DurMax,
			MaxRows:        accum.RowMax,
		}
		if accum.Count > 0 {
			ts.AvgDurationMs = accum.DurSum / float64(accum.Count)
			ts.AvgRows = int(accum.RowSum / int64(accum.Count))
		}
		queryStats.ByTable = append(queryStats.ByTable, ts)
	}
	sort.Slice(queryStats.ByTable, func(i, j int) bool {
		return queryStats.ByTable[i].ExecutionCount > queryStats.ByTable[j].ExecutionCount
	})
	// Copy slow queries (newest first)
	for i := len(metricsStore.slowQueries) - 1; i >= 0; i-- {
		queryStats.SlowQueries = append(queryStats.SlowQueries, metricsStore.slowQueries[i])
	}
	// Copy large results (newest first)
	for i := len(metricsStore.largeResults) - 1; i >= 0; i-- {
		queryStats.LargeResults = append(queryStats.LargeResults, metricsStore.largeResults[i])
	}

	return MetricsResponse{
		ServerStartTime:       metricsStore.serverStartTime,
		TotalRequests:         totalRequests,
		TotalErrors:           totalErrors,
		ErrorRate:             errorRate,
		ActiveRequests:        metricsStore.activeRequests,
		PeakConcurrent:        metricsStore.peakConcurrent,
		UniqueReports:         len(reportStats),
		TotalReports:          totalReportsCount,
		ReportStats:           reportStats,
		TopReports:            topReports,
		LeastRequestedReports: leastRequested,
		RecentRequests:        recentRequests,
		Cache:                 componentCache.GetStats(false),
		DBPool:                getDBPoolStats(),
		HourlyHits:            hourlyHits,
		DailyHits:             dailyHits,
		DateHits:              dateHits,
		DurationBuckets:       durationBuckets,
		PDFStats:              pdfStats,
		CSVStats:              csvStats,
		ChatStats:             chatStats,
		ComponentStats:        componentStats,
		QueryStats:            queryStats,
		UserStats:             userStatsList,
		UIStats:               GetUIStats(),
	}
}

// intToStr converts an int to string without importing strconv
func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return digits
}

// SaveMetrics writes current metrics to data/metrics.json atomically
func SaveMetrics() error {
	if metricsStore == nil {
		return nil
	}

	if err := os.MkdirAll("data", 0755); err != nil {
		return err
	}

	metricsStore.RLock()
	persisted := PersistedMetrics{
		SavedAt:               time.Now(),
		ReportCounts:          maps.Clone(metricsStore.reportCounts),
		ReportErrors:          maps.Clone(metricsStore.reportErrors),
		ReportDurSum:          maps.Clone(metricsStore.reportDurSum),
		ReportDurCount:        maps.Clone(metricsStore.reportDurCount),
		ReportDurMax:          maps.Clone(metricsStore.reportDurMax),
		LastAccessed:          maps.Clone(metricsStore.lastAccessed),
		HourlyHits:            maps.Clone(metricsStore.hourlyHits),
		DailyHits:             maps.Clone(metricsStore.dailyHits),
		DateHits:              maps.Clone(metricsStore.dateHits),
		DurationBuckets:       metricsStore.durationBuckets,
		PeakConcurrent:        metricsStore.peakConcurrent,
		PDFDownloads:          metricsStore.pdfDownloads,
		PDFErrors:             metricsStore.pdfErrors,
		PDFDurSum:             metricsStore.pdfDurSum,
		PDFDurMax:             metricsStore.pdfDurMax,
		PDFByReport:           maps.Clone(metricsStore.pdfByReport),
		CSVDownloads:          metricsStore.csvDownloads,
		CSVTotalRows:          metricsStore.csvTotalRows,
		CSVByReport:           maps.Clone(metricsStore.csvByReport),
		ChatQuestions:         metricsStore.chatQuestions,
		ChatExecutions:        metricsStore.chatExecutions,
		ChatErrors:            metricsStore.chatErrors,
		ChatAskDurSum:         metricsStore.chatAskDurSum,
		ChatAskDurCount:       metricsStore.chatAskDurCount,
		ChatExecDurSum:        metricsStore.chatExecDurSum,
		ChatExecDurCount:      metricsStore.chatExecDurCount,
		ChatByTable:           maps.Clone(metricsStore.chatByTable),
		ComponentExecTotal:    metricsStore.componentExecTotal,
		ComponentFailTotal:    metricsStore.componentFailTotal,
		ComponentFailByReport: maps.Clone(metricsStore.componentFailByReport),
		QueryExecTotal:        metricsStore.queryExecTotal,
		QueryErrorTotal:       metricsStore.queryErrorTotal,
	}
	// Deep-clone query by table accumulators
	if len(metricsStore.queryByTable) > 0 {
		persisted.QueryByTable = make(map[string]*queryTableAccum, len(metricsStore.queryByTable))
		for k, v := range metricsStore.queryByTable {
			clone := *v
			persisted.QueryByTable[k] = &clone
		}
	}
	// Deep-clone user stats (each entry has a TopReports map)
	if len(metricsStore.userStats) > 0 {
		persisted.UserStats = make(map[string]*UserStats, len(metricsStore.userStats))
		for k, us := range metricsStore.userStats {
			persisted.UserStats[k] = &UserStats{
				Username:     us.Username,
				Email:        us.Email,
				RequestCount: us.RequestCount,
				LastSeen:     us.LastSeen,
				TopReports:   maps.Clone(us.TopReports),
			}
		}
	}
	metricsStore.RUnlock()

	data, err := json.MarshalIndent(persisted, "", "  ")
	if err != nil {
		return err
	}

	tmpPath := metricsFilePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmpPath, metricsFilePath)
}

// LoadMetrics reads persisted metrics from disk and merges into the store
func LoadMetrics() {
	if metricsStore == nil {
		return
	}

	data, err := os.ReadFile(metricsFilePath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Warning: Could not read metrics file: %v", err)
		}
		return
	}

	var persisted PersistedMetrics
	if err := json.Unmarshal(data, &persisted); err != nil {
		log.Printf("Warning: Could not parse metrics file: %v", err)
		return
	}

	metricsStore.Lock()
	defer metricsStore.Unlock()

	// Merge persisted data (nil-safe)
	if persisted.ReportCounts != nil {
		for k, v := range persisted.ReportCounts {
			metricsStore.reportCounts[k] += v
		}
	}
	if persisted.ReportErrors != nil {
		for k, v := range persisted.ReportErrors {
			metricsStore.reportErrors[k] += v
		}
	}
	if persisted.ReportDurSum != nil {
		for k, v := range persisted.ReportDurSum {
			metricsStore.reportDurSum[k] += v
		}
	}
	if persisted.ReportDurCount != nil {
		for k, v := range persisted.ReportDurCount {
			metricsStore.reportDurCount[k] += v
		}
	}
	if persisted.ReportDurMax != nil {
		for k, v := range persisted.ReportDurMax {
			if v > metricsStore.reportDurMax[k] {
				metricsStore.reportDurMax[k] = v
			}
		}
	}
	if persisted.LastAccessed != nil {
		for k, v := range persisted.LastAccessed {
			if v.After(metricsStore.lastAccessed[k]) {
				metricsStore.lastAccessed[k] = v
			}
		}
	}
	if persisted.HourlyHits != nil {
		for k, v := range persisted.HourlyHits {
			metricsStore.hourlyHits[k] += v
		}
	}
	if persisted.DailyHits != nil {
		for k, v := range persisted.DailyHits {
			metricsStore.dailyHits[k] += v
		}
	}
	if persisted.DateHits != nil {
		for k, v := range persisted.DateHits {
			metricsStore.dateHits[k] += v
		}
	}
	for i := range persisted.DurationBuckets {
		metricsStore.durationBuckets[i] += persisted.DurationBuckets[i]
	}
	if persisted.PeakConcurrent > metricsStore.peakConcurrent {
		metricsStore.peakConcurrent = persisted.PeakConcurrent
	}

	// Merge PDF metrics
	metricsStore.pdfDownloads += persisted.PDFDownloads
	metricsStore.pdfErrors += persisted.PDFErrors
	metricsStore.pdfDurSum += persisted.PDFDurSum
	if persisted.PDFDurMax > metricsStore.pdfDurMax {
		metricsStore.pdfDurMax = persisted.PDFDurMax
	}
	if persisted.PDFByReport != nil {
		for k, v := range persisted.PDFByReport {
			metricsStore.pdfByReport[k] += v
		}
	}

	// Merge CSV metrics
	metricsStore.csvDownloads += persisted.CSVDownloads
	metricsStore.csvTotalRows += persisted.CSVTotalRows
	if persisted.CSVByReport != nil {
		for k, v := range persisted.CSVByReport {
			metricsStore.csvByReport[k] += v
		}
	}

	// Merge chat metrics
	metricsStore.chatQuestions += persisted.ChatQuestions
	metricsStore.chatExecutions += persisted.ChatExecutions
	metricsStore.chatErrors += persisted.ChatErrors
	metricsStore.chatAskDurSum += persisted.ChatAskDurSum
	metricsStore.chatAskDurCount += persisted.ChatAskDurCount
	metricsStore.chatExecDurSum += persisted.ChatExecDurSum
	metricsStore.chatExecDurCount += persisted.ChatExecDurCount
	if persisted.ChatByTable != nil {
		for k, v := range persisted.ChatByTable {
			metricsStore.chatByTable[k] += v
		}
	}

	// Merge component failure metrics
	metricsStore.componentExecTotal += persisted.ComponentExecTotal
	metricsStore.componentFailTotal += persisted.ComponentFailTotal
	if persisted.ComponentFailByReport != nil {
		for k, v := range persisted.ComponentFailByReport {
			metricsStore.componentFailByReport[k] += v
		}
	}

	// Merge query performance metrics
	metricsStore.queryExecTotal += persisted.QueryExecTotal
	metricsStore.queryErrorTotal += persisted.QueryErrorTotal
	if persisted.QueryByTable != nil {
		for k, v := range persisted.QueryByTable {
			existing, ok := metricsStore.queryByTable[k]
			if !ok {
				clone := *v
				metricsStore.queryByTable[k] = &clone
			} else {
				existing.Count += v.Count
				existing.Errors += v.Errors
				existing.DurSum += v.DurSum
				if v.DurMax > existing.DurMax {
					existing.DurMax = v.DurMax
				}
				existing.RowSum += v.RowSum
				if v.RowMax > existing.RowMax {
					existing.RowMax = v.RowMax
				}
			}
		}
	}

	// Merge user stats
	if persisted.UserStats != nil {
		for k, pus := range persisted.UserStats {
			existing, ok := metricsStore.userStats[k]
			if !ok {
				metricsStore.userStats[k] = &UserStats{
					Username:     pus.Username,
					Email:        pus.Email,
					RequestCount: pus.RequestCount,
					LastSeen:     pus.LastSeen,
					TopReports:   maps.Clone(pus.TopReports),
				}
			} else {
				existing.RequestCount += pus.RequestCount
				if pus.LastSeen.After(existing.LastSeen) {
					existing.LastSeen = pus.LastSeen
				}
				if pus.Email != "" {
					existing.Email = pus.Email
				}
				for reportID, count := range pus.TopReports {
					existing.TopReports[reportID] += count
				}
			}
		}
	}

	log.Printf("Loaded persisted metrics (saved at %s)", persisted.SavedAt.Format(time.RFC3339))
}

// StartPeriodicSave starts a goroutine that saves metrics every 30 minutes.
// Close the returned channel to stop it.
func StartPeriodicSave() chan struct{} {
	stop := make(chan struct{})
	ticker := time.NewTicker(30 * time.Minute)

	go func() {
		for {
			select {
			case <-ticker.C:
				if err := SaveMetrics(); err != nil {
					log.Printf("Periodic metrics save failed: %v", err)
				} else {
					log.Println("Periodic metrics save completed")
				}
			case <-stop:
				ticker.Stop()
				return
			}
		}
	}()

	return stop
}

// GetMetricsHandler returns the metrics API endpoint handler
func GetMetricsHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, r)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	metrics := GetMetrics()

	// Strip PII and verbose audit fields from public response
	for i := range metrics.ChatStats.RecentInteractions {
		metrics.ChatStats.RecentInteractions[i].Username = ""
		metrics.ChatStats.RecentInteractions[i].Question = ""
		metrics.ChatStats.RecentInteractions[i].Correction = ""
		metrics.ChatStats.RecentInteractions[i].AIResponse = ""
		metrics.ChatStats.RecentInteractions[i].SQL = ""
		metrics.ChatStats.RecentInteractions[i].Summary = ""
	}
	for i := range metrics.UIStats.RecentEvents {
		metrics.UIStats.RecentEvents[i].Username = ""
		metrics.UIStats.RecentEvents[i].SessionID = ""
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(metrics)
}

// RecordCSVDownloadHandler handles POST /api/metrics/csv
func RecordCSVDownloadHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, r)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		ReportID       string `json:"reportId"`
		ComponentTitle string `json:"componentTitle"`
		RowCount       int    `json:"rowCount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	RecordCSVDownload(body.ReportID, body.RowCount)
	w.WriteHeader(http.StatusNoContent)
}

// responseRecorder wraps http.ResponseWriter to capture status code
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *responseRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

// MetricsMiddleware wraps a handler to record request metrics
func MetricsMiddleware(next http.HandlerFunc, extractReportID func(*http.Request) string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Track concurrent requests
		IncrementActiveRequests()
		defer DecrementActiveRequests()

		// Create response recorder to capture status code
		recorder := &responseRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Call the actual handler
		next(recorder, r)

		// Record the metric
		duration := time.Since(start)
		reportID := extractReportID(r)

		// Internet scanners probe with arbitrary URLs (SSRF payloads, path
		// traversal). Never let an unvalidated path segment become a persisted
		// metrics key — collapse invalid IDs to a single sentinel so the request
		// still counts but attacker-controlled strings never reach data/metrics.json.
		if reportID != "" && ValidateReportID(reportID) != nil {
			reportID = "_invalid"
		}

		// Extract filters from query params
		filters := make(map[string]string)
		for key, values := range r.URL.Query() {
			if len(values) > 0 {
				filters[key] = values[0]
			}
		}

		metric := RequestMetric{
			ReportID:   reportID,
			Timestamp:  start,
			DurationMs: duration.Milliseconds(),
			Filters:    filters,
			StatusCode: recorder.statusCode,
			IsError:    recorder.statusCode >= 400,
		}

		// Extract user identity from auth context (nil when AUTH_MODE=off)
		if claims := GetUserClaims(r); claims != nil {
			metric.Username = claims.Username
			metric.Email = claims.Email
		}

		RecordRequest(metric)
	}
}
