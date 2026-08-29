package api

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"statchat/pkg/model"
	"statchat/pkg/store"

	"github.com/gorilla/mux"
)

var complianceAdminRoles = map[string]bool{
	"admin": true, "superadmin": true, "tenant_admin": true, "platform_admin": true,
}

func isComplianceAdministrator(user model.User) bool {
	for _, role := range user.Roles {
		if complianceAdminRoles[strings.ToLower(strings.TrimSpace(role))] {
			return true
		}
	}
	return false
}

func requireComplianceAdministrator(w http.ResponseWriter, r *http.Request) bool {
	user, err := requestCurrentUser(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authenticated user is required")
		return false
	}
	if !isComplianceAdministrator(user) {
		writeError(w, http.StatusForbidden, "tenant administrator access is required")
		return false
	}
	return true
}

func retentionPolicyHandler(w http.ResponseWriter, r *http.Request) {
	if !requireComplianceAdministrator(w, r) {
		return
	}
	policy, err := store.GetRetentionPolicy(requestTenantID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load retention policy")
		return
	}
	writeJSON(w, policy)
}

func updateRetentionPolicyHandler(w http.ResponseWriter, r *http.Request) {
	if !requireComplianceAdministrator(w, r) {
		return
	}
	var request struct {
		RetentionDays int  `json:"retentionDays"`
		Enabled       bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	if request.RetentionDays < 1 || request.RetentionDays > 3650 {
		writeError(w, http.StatusBadRequest, "retentionDays must be between 1 and 3650")
		return
	}
	policy, err := store.UpsertRetentionPolicy(requestTenantID(r), requestUserID(r), request.RetentionDays, request.Enabled)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update retention policy")
		return
	}
	writeJSON(w, policy)
}

func enforceRetentionHandler(w http.ResponseWriter, r *http.Request) {
	if !requireComplianceAdministrator(w, r) {
		return
	}
	count, err := store.EnforceRetention(r.Context(), requestTenantID(r), requestUserID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to enforce retention policy")
		return
	}
	writeJSON(w, map[string]any{"deletedMessages": count})
}

func legalHoldsHandler(w http.ResponseWriter, r *http.Request) {
	if !requireComplianceAdministrator(w, r) {
		return
	}
	holds, err := store.ListLegalHolds(requestTenantID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load legal holds")
		return
	}
	writeJSON(w, holds)
}

func createLegalHoldHandler(w http.ResponseWriter, r *http.Request) {
	if !requireComplianceAdministrator(w, r) {
		return
	}
	var request struct {
		ConversationID string `json:"conversationId"`
		Name           string `json:"name"`
		Reason         string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	request.ConversationID = strings.TrimSpace(request.ConversationID)
	request.Name = strings.TrimSpace(request.Name)
	request.Reason = strings.TrimSpace(request.Reason)
	if request.Name == "" || len(request.Name) > 120 || request.Reason == "" || len(request.Reason) > 1000 {
		writeError(w, http.StatusBadRequest, "name and reason are required and must be within their limits")
		return
	}
	if request.ConversationID != "" {
		valid, err := store.ConversationBelongsToTenant(request.ConversationID, requestTenantID(r))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to validate conversation")
			return
		}
		if !valid {
			writeError(w, http.StatusBadRequest, "conversation does not belong to this tenant")
			return
		}
	}
	hold, err := store.CreateLegalHold(model.LegalHold{TenantID: requestTenantID(r), ConversationID: request.ConversationID, Name: request.Name, Reason: request.Reason, CreatedBy: requestUserID(r)})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create legal hold")
		return
	}
	writeJSON(w, hold)
}

func releaseLegalHoldHandler(w http.ResponseWriter, r *http.Request) {
	if !requireComplianceAdministrator(w, r) {
		return
	}
	released, err := store.ReleaseLegalHold(requestTenantID(r), mux.Vars(r)["id"], requestUserID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to release legal hold")
		return
	}
	if !released {
		writeError(w, http.StatusNotFound, "active legal hold not found")
		return
	}
	writeJSON(w, map[string]bool{"released": true})
}

func complianceAuditHandler(w http.ResponseWriter, r *http.Request) {
	if !requireComplianceAdministrator(w, r) {
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	events, err := store.ListComplianceAuditEvents(requestTenantID(r), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load compliance audit")
		return
	}
	writeJSON(w, events)
}

type conversationExport struct {
	GeneratedAt    time.Time       `json:"generatedAt"`
	TenantID       string          `json:"tenantId"`
	ConversationID string          `json:"conversationId"`
	RequestedBy    string          `json:"requestedBy"`
	IncludeDeleted bool            `json:"includeDeleted"`
	MessageCount   int             `json:"messageCount"`
	Messages       []model.Message `json:"messages"`
}

func exportConversationHandler(w http.ResponseWriter, r *http.Request) {
	conversationID := mux.Vars(r)["id"]
	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if format == "" {
		format = "json"
	}
	if format != "json" && format != "csv" {
		writeError(w, http.StatusBadRequest, "format must be json or csv")
		return
	}
	includeDeleted, err := strconv.ParseBool(r.URL.Query().Get("includeDeleted"))
	if err != nil && r.URL.Query().Get("includeDeleted") != "" {
		writeError(w, http.StatusBadRequest, "includeDeleted must be a boolean")
		return
	}
	if includeDeleted {
		user, userErr := requestCurrentUser(r)
		if userErr != nil || !isComplianceAdministrator(user) {
			writeError(w, http.StatusForbidden, "tenant administrator access is required to export deleted messages")
			return
		}
		valid, validationErr := store.ConversationBelongsToTenant(conversationID, requestTenantID(r))
		if validationErr != nil {
			writeError(w, http.StatusInternalServerError, "failed to validate conversation")
			return
		}
		if !valid {
			writeError(w, http.StatusForbidden, "conversation does not belong to this tenant")
			return
		}
	} else if !requireConversationAccess(w, r, conversationID) {
		return
	}
	messages, err := store.ExportConversationMessages(conversationID, requestTenantID(r), includeDeleted)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to build conversation export")
		return
	}
	export := conversationExport{GeneratedAt: time.Now().UTC(), TenantID: requestTenantID(r), ConversationID: conversationID, RequestedBy: requestUserID(r), IncludeDeleted: includeDeleted, MessageCount: len(messages), Messages: messages}
	var body []byte
	contentType := "application/json; charset=utf-8"
	if format == "json" {
		body, err = json.MarshalIndent(export, "", "  ")
	} else {
		body, err = encodeConversationCSV(messages)
		contentType = "text/csv; charset=utf-8"
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to encode conversation export")
		return
	}
	if err := store.CreateComplianceAuditEvent(requestTenantID(r), requestUserID(r), "conversation.exported", "conversation", conversationID, map[string]any{"format": format, "includeDeleted": includeDeleted, "messageCount": len(messages)}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to audit conversation export")
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="statchat-%s.%s"`, safeExportName(conversationID), format))
	w.Write(body)
}

func encodeConversationCSV(messages []model.Message) ([]byte, error) {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	if err := writer.Write([]string{"message_id", "created_at", "sender_id", "sender", "text", "status", "attachment_urls", "latitude", "longitude"}); err != nil {
		return nil, err
	}
	for _, message := range messages {
		urls := make([]string, 0, len(message.Attachments))
		for _, attachment := range message.Attachments {
			urls = append(urls, attachment.URL)
		}
		latitude, longitude := "", ""
		if message.Location != nil {
			latitude = strconv.FormatFloat(message.Location.Latitude, 'f', -1, 64)
			longitude = strconv.FormatFloat(message.Location.Longitude, 'f', -1, 64)
		}
		if err := writer.Write([]string{message.ID, message.CreatedAt.UTC().Format(time.RFC3339Nano), csvSafe(message.SenderID), csvSafe(message.Sender), csvSafe(message.Text), message.Status, strings.Join(urls, " "), latitude, longitude}); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	return buffer.Bytes(), writer.Error()
}

func csvSafe(value string) string {
	trimmed := strings.TrimLeft(value, " \t\r\n")
	if trimmed != "" && strings.ContainsRune("=+-@", rune(trimmed[0])) {
		return "'" + value
	}
	return value
}

func safeExportName(value string) string {
	value = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, value)
	return strings.Trim(value, "-")
}
