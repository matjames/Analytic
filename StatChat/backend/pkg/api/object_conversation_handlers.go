package api

import (
	"crypto/hmac"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"statchat/pkg/store"
)

type objectConversationRequest struct {
	ObjectRef    string         `json:"objectRef"`
	Name         string         `json:"name"`
	Title        string         `json:"title"`
	MemberIDs    []string       `json:"memberIds"`
	Participants []string       `json:"participants"`
	Metadata     map[string]any `json:"metadata"`
}

func objectConversationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		objectRef := strings.TrimSpace(r.URL.Query().Get("objectRef"))
		if !validObjectRef(objectRef) {
			writeError(w, http.StatusBadRequest, "objectRef must use obj:<module>:<entity>:<id>")
			return
		}
		conversation, err := store.GetObjectConversation(requestTenantID(r), objectRef)
		if err != nil {
			writeError(w, http.StatusNotFound, "object conversation not found")
			return
		}
		if !requireConversationAccess(w, r, conversation.ID) {
			return
		}
		writeJSON(w, conversation)
		return
	}

	var req objectConversationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	req.ObjectRef = strings.TrimSpace(req.ObjectRef)
	if !validObjectRef(req.ObjectRef) {
		writeError(w, http.StatusBadRequest, "objectRef must use obj:<module>:<entity>:<id>")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = strings.TrimSpace(req.Title)
	}
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	members := append([]string{}, req.MemberIDs...)
	members = append(members, req.Participants...)
	conversation, err := store.CreateOrGetObjectConversation(
		requestTenantID(r), req.ObjectRef, name, requestUserID(r), members, req.Metadata,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create object conversation")
		return
	}
	if !containsString(conversation.MemberIDs, requestUserID(r)) && trustedObjectConversationRequest(r) {
		if err := store.AddConversationMember(conversation.ID, requestUserID(r)); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to join object conversation")
			return
		}
		conversation, err = store.GetObjectConversation(requestTenantID(r), req.ObjectRef)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to reload object conversation")
			return
		}
	}
	if !containsString(conversation.MemberIDs, requestUserID(r)) {
		writeError(w, http.StatusForbidden, "you do not have access to this object conversation")
		return
	}
	writeJSON(w, conversation)
}

func trustedObjectConversationRequest(r *http.Request) bool {
	expected := strings.TrimSpace(os.Getenv("STATGATE_INTERNAL_API_KEY"))
	supplied := strings.TrimSpace(r.Header.Get("X-Internal-API-Key"))
	return expected != "" && supplied != "" && hmac.Equal([]byte(expected), []byte(supplied))
}

func validObjectRef(value string) bool {
	if len(value) < 9 || len(value) > 512 || !strings.HasPrefix(value, "obj:") {
		return false
	}
	parts := strings.Split(value, ":")
	if len(parts) < 4 {
		return false
	}
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			return false
		}
	}
	return true
}
