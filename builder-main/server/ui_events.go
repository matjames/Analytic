package main

import (
	"encoding/json"
	"maps"
	"net/http"
	"strings"
	"sync"
	"time"
)

// UI interaction telemetry. Self-contained module — plugs into the existing
// MetricsStore via a single field (see metrics.go). Server-side whitelist
// keeps the event taxonomy bounded and props structurally PII-free.

// UIEvent is a single UI interaction reported by the frontend.
type UIEvent struct {
	Name      string            `json:"name"`
	ReportID  string            `json:"reportId,omitempty"`
	Props     map[string]string `json:"props,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
	SessionID string            `json:"sessionId,omitempty"`
	Username  string            `json:"username,omitempty"`
}

// UIEventStats is the aggregated view returned by /api/metrics.
type UIEventStats struct {
	TotalEvents   int                       `json:"totalEvents"`
	CountsByName  map[string]int            `json:"countsByName"`
	CountsByRoute map[string]map[string]int `json:"countsByRoute"`
	RecentEvents  []UIEvent                 `json:"recentEvents"`
}

// uiEventWhitelist is the authoritative set of accepted event names. Adding
// a new event = adding a line here plus the call site. Unknown names are
// rejected with 400.
var uiEventWhitelist = map[string]bool{
	"report.open":              true,
	"filter.apply":             true,
	"filter.reset":             true,
	"share.menu_open":          true,
	"share.copy_link":          true,
	"ask.submit":               true,
	"ask.cancel":               true,
	"builder.save":             true,
	"builder.validation_block": true,
}

const (
	uiMaxPropKeys     = 8
	uiMaxPropValueLen = 64
	uiMaxPropKeyLen   = 32
	uiMaxRecentEvents = 500
)

// UIStore holds the UI event ring buffer and counters. Held under its own
// lock rather than piggybacking on MetricsStore's lock to keep contention
// isolated (UI events are higher-volume than report requests).
type UIStore struct {
	sync.RWMutex
	total         int
	countsByName  map[string]int
	countsByRoute map[string]map[string]int
	recent        []UIEvent
}

var uiStore *UIStore

// InitUIEvents initializes the UI event store. Called from InitMetrics().
func InitUIEvents() {
	uiStore = &UIStore{
		countsByName:  make(map[string]int),
		countsByRoute: make(map[string]map[string]int),
		recent:        make([]UIEvent, 0, uiMaxRecentEvents),
	}
}

// sanitizeProps enforces the bounded-props contract: drops unknown keys over
// the limit, trims whitespace, and caps value length.
func sanitizeProps(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		if len(out) >= uiMaxPropKeys {
			break
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if k == "" || len(k) > uiMaxPropKeyLen {
			continue
		}
		if len(v) > uiMaxPropValueLen {
			v = v[:uiMaxPropValueLen]
		}
		out[k] = v
	}
	return out
}

// RecordUIEvent stores a UI event. Unknown event names are dropped silently
// here — the handler rejects them earlier with 400.
func RecordUIEvent(ev UIEvent) {
	if uiStore == nil || !uiEventWhitelist[ev.Name] {
		return
	}

	uiStore.Lock()
	defer uiStore.Unlock()

	uiStore.total++
	uiStore.countsByName[ev.Name]++

	if ev.ReportID != "" {
		byReport, ok := uiStore.countsByRoute[ev.Name]
		if !ok {
			byReport = make(map[string]int)
			uiStore.countsByRoute[ev.Name] = byReport
		}
		byReport[ev.ReportID]++
	}

	if len(uiStore.recent) >= uiMaxRecentEvents {
		uiStore.recent = uiStore.recent[1:]
	}
	uiStore.recent = append(uiStore.recent, ev)
}

// GetUIStats returns a snapshot of UI event aggregates. PII (username,
// sessionId) is NOT stripped here — callers that expose this publicly
// (see GetMetricsHandler) must strip before returning to clients.
func GetUIStats() UIEventStats {
	if uiStore == nil {
		return UIEventStats{
			CountsByName:  map[string]int{},
			CountsByRoute: map[string]map[string]int{},
			RecentEvents:  []UIEvent{},
		}
	}

	uiStore.RLock()
	defer uiStore.RUnlock()

	countsByRoute := make(map[string]map[string]int, len(uiStore.countsByRoute))
	for name, byReport := range uiStore.countsByRoute {
		countsByRoute[name] = maps.Clone(byReport)
	}

	recent := make([]UIEvent, 0, len(uiStore.recent))
	for i := len(uiStore.recent) - 1; i >= 0; i-- {
		recent = append(recent, uiStore.recent[i])
	}

	return UIEventStats{
		TotalEvents:   uiStore.total,
		CountsByName:  maps.Clone(uiStore.countsByName),
		CountsByRoute: countsByRoute,
		RecentEvents:  recent,
	}
}

// uiEventRequest is the accepted POST body for /api/metrics/ui-event.
// Batch endpoint: clients POST a flush of queued events at once.
type uiEventRequest struct {
	SessionID string              `json:"sessionId"`
	Events    []uiEventRequestRow `json:"events"`
}

type uiEventRequestRow struct {
	Name     string            `json:"name"`
	ReportID string            `json:"reportId,omitempty"`
	Props    map[string]string `json:"props,omitempty"`
}

const uiMaxBatchSize = 50

// PostUIEventHandler handles POST /api/metrics/ui-event. Unauthed (matches
// the /api/metrics/csv pattern); user identity is pulled from JWT claims
// when present.
func PostUIEventHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, r)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Cap request body to prevent abuse; a 50-event batch with bounded
	// props fits well under 16KB.
	r.Body = http.MaxBytesReader(w, r.Body, 32*1024)

	var body uiEventRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	if len(body.Events) == 0 || len(body.Events) > uiMaxBatchSize {
		http.Error(w, "Invalid batch size", http.StatusBadRequest)
		return
	}

	var username, email string
	if claims := GetUserClaims(r); claims != nil {
		username = claims.Username
		email = claims.Email
	}
	_ = email // reserved; not stored on events to keep rows small

	sessionID := strings.TrimSpace(body.SessionID)
	if len(sessionID) > 64 {
		sessionID = sessionID[:64]
	}

	now := time.Now()
	for _, row := range body.Events {
		if !uiEventWhitelist[row.Name] {
			continue
		}
		RecordUIEvent(UIEvent{
			Name:      row.Name,
			ReportID:  row.ReportID,
			Props:     sanitizeProps(row.Props),
			Timestamp: now,
			SessionID: sessionID,
			Username:  username,
		})
	}

	w.WriteHeader(http.StatusNoContent)
}
