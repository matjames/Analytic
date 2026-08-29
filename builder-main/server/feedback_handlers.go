package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	feedbackMaxCommentLen = 4000
	feedbackMaxBodyBytes  = 1 << 15 // 32 KiB
	feedbackDefaultLimit  = 50
)

// feedbackPublicRead, when true, lets GET /api/feedback{,/{reportID}} bypass
// AuthMiddleware. Identities are redacted in the response so PII does not leak.
// POST always requires auth regardless of this flag.
var feedbackPublicRead bool

// InitFeedbackConfig reads feedback-related env vars once at startup.
// Call after InitFeedbackDB in main().
func InitFeedbackConfig() {
	feedbackPublicRead = strings.EqualFold(os.Getenv("FEEDBACK_PUBLIC_READ"), "true")
	if feedbackPublicRead {
		logInfo("FEEDBACK_PUBLIC_READ=true: GET /api/feedback endpoints are public; identities redacted")
	}
}

// feedbackConditionalAuth allows unauthenticated GETs (for the public,
// redacted feedback list) while keeping POST behind AuthMiddleware so the
// stored audit trail always has a real identity.
func feedbackConditionalAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			next(w, r)
			return
		}
		AuthMiddleware(next)(w, r)
	}
}

// redactFeedbackEntries strips identifying fields before serialising to
// unauthenticated callers. Mutates entries in place and returns the slice.
func redactFeedbackEntries(entries []FeedbackEntry) []FeedbackEntry {
	for i := range entries {
		entries[i].Username = "anonymous"
		entries[i].Email = ""
	}
	return entries
}

// FeedbackHandler routes /api/feedback/{reportID} to POST or GET.
// Must be wrapped by AuthMiddleware so GetUserClaims(r) is populated when auth is on.
func FeedbackHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	prefix := basePath + "/api/feedback/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		writeJSONError(w, http.StatusBadRequest, "invalid path")
		return
	}
	reportID := strings.TrimPrefix(r.URL.Path, prefix)
	if err := ValidateReportID(reportID); err != nil {
		logWarnCtx(ctx, "Feedback: invalid report ID %q: %v", reportID, err)
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid report ID: %v", err))
		return
	}

	switch r.Method {
	case http.MethodPost:
		postFeedback(w, r, reportID)
	case http.MethodGet:
		getFeedback(w, r, reportID)
	default:
		w.Header().Set("Allow", "GET, POST")
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

type feedbackPostBody struct {
	Comment   string `json:"comment"`
	SectionID string `json:"sectionId"`
}

type feedbackPostResponse struct {
	ID        int64  `json:"id"`
	CreatedAt int64  `json:"createdAt"`
	Username  string `json:"username"`
}

func postFeedback(w http.ResponseWriter, r *http.Request, reportID string) {
	ctx := r.Context()

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, feedbackMaxBodyBytes))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "request body too large or unreadable")
		return
	}
	var payload feedbackPostBody
	if err := json.Unmarshal(body, &payload); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	comment := strings.TrimSpace(payload.Comment)
	if comment == "" {
		writeJSONError(w, http.StatusBadRequest, "comment is required")
		return
	}
	// Count by runes so multi-byte characters aren't penalised by length.
	if runeLen := len([]rune(comment)); runeLen > feedbackMaxCommentLen {
		writeJSONError(w, http.StatusBadRequest,
			fmt.Sprintf("comment too long (max %d characters, got %d)", feedbackMaxCommentLen, runeLen))
		return
	}

	claims := GetUserClaims(r)
	userSub, username, email := identityFromClaims(claims, r)

	sectionID := strings.TrimSpace(payload.SectionID)

	id, err := InsertFeedback(ctx, reportID, userSub, username, email, sectionID, comment)
	if err != nil {
		logErrorCtx(ctx, "Feedback insert failed (report=%s user=%s): %v", reportID, username, err)
		writeJSONError(w, http.StatusInternalServerError, "failed to save feedback")
		return
	}
	if sectionID != "" {
		logInfoCtx(ctx, "Feedback saved: id=%d report=%s section=%s user=%s", id, reportID, sectionID, username)
	} else {
		logInfoCtx(ctx, "Feedback saved: id=%d report=%s user=%s", id, reportID, username)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(feedbackPostResponse{
		ID:        id,
		CreatedAt: time.Now().Unix(),
		Username:  username,
	})
}

type feedbackListResponse struct {
	Entries  []FeedbackEntry `json:"entries"`
	HasMore  bool            `json:"hasMore"`
	NextCur  int64           `json:"nextCursor,omitempty"` // pass as ?before= to paginate
	ReportID string          `json:"reportId,omitempty"`   // empty when listing across all reports
}

// FeedbackListAllHandler returns feedback entries across all reports,
// newest first, with the same keyset pagination shape as the per-report GET.
// Mounted at GET /api/feedback (no trailing slash) behind AuthMiddleware.
func FeedbackListAllHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	ctx := r.Context()

	limit := feedbackDefaultLimit
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	var before int64
	if v := r.URL.Query().Get("before"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			before = n
		}
	}

	entries, err := ListAllFeedback(ctx, limit+1, before)
	if err != nil {
		logErrorCtx(ctx, "Feedback list-all failed: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to load feedback")
		return
	}

	resp := feedbackListResponse{Entries: entries}
	if len(entries) > limit {
		resp.Entries = entries[:limit]
		resp.HasMore = true
		resp.NextCur = resp.Entries[len(resp.Entries)-1].CreatedAt
	}
	if resp.Entries == nil {
		resp.Entries = []FeedbackEntry{}
	}
	if feedbackPublicRead {
		resp.Entries = redactFeedbackEntries(resp.Entries)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func getFeedback(w http.ResponseWriter, r *http.Request, reportID string) {
	ctx := r.Context()

	limit := feedbackDefaultLimit
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	var before int64
	if v := r.URL.Query().Get("before"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			before = n
		}
	}
	sectionID := strings.TrimSpace(r.URL.Query().Get("sectionId"))

	// Request one extra to determine hasMore without a second query.
	entries, err := ListFeedback(ctx, reportID, limit+1, before, sectionID)
	if err != nil {
		logErrorCtx(ctx, "Feedback list failed (report=%s): %v", reportID, err)
		writeJSONError(w, http.StatusInternalServerError, "failed to load feedback")
		return
	}

	resp := feedbackListResponse{
		ReportID: reportID,
		Entries:  entries,
	}
	if len(entries) > limit {
		resp.Entries = entries[:limit]
		resp.HasMore = true
		resp.NextCur = resp.Entries[len(resp.Entries)-1].CreatedAt
	}
	if resp.Entries == nil {
		resp.Entries = []FeedbackEntry{}
	}
	if feedbackPublicRead {
		resp.Entries = redactFeedbackEntries(resp.Entries)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// identityFromClaims extracts user identity from JWT claims.
// When AUTH_MODE=off, claims are nil; we fall back to a stable "anonymous" marker
// so local development still works. Production (AUTH_MODE=on) always has claims.
func identityFromClaims(claims *UserClaims, r *http.Request) (userSub, username, email string) {
	if claims != nil && claims.Subject != "" {
		username = claims.Username
		if username == "" {
			username = claims.Name
		}
		if username == "" {
			username = claims.Email
		}
		if username == "" {
			username = "user"
		}
		return claims.Subject, username, claims.Email
	}
	// AUTH_MODE=off: no identity available. Tag by IP so comments aren't all
	// indistinguishable, but make it explicit this is anonymous.
	ip := getClientIP(r)
	return "anonymous", "anonymous@" + ip, ""
}
