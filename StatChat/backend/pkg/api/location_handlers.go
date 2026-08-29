package api

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"statchat/pkg/model"
	"statchat/pkg/store"

	"github.com/google/uuid"
)

type sendLocationRequest struct {
	ConversationID string  `json:"conversationId"`
	Latitude       float64 `json:"latitude"`
	Longitude      float64 `json:"longitude"`
	AccuracyMeters float64 `json:"accuracyMeters,omitempty"`
	Label          string  `json:"label,omitempty"`
}

func validateLocationRequest(req sendLocationRequest) error {
	if strings.TrimSpace(req.ConversationID) == "" {
		return fmt.Errorf("conversationId is required")
	}
	if math.IsNaN(req.Latitude) || math.IsInf(req.Latitude, 0) || req.Latitude < -90 || req.Latitude > 90 {
		return fmt.Errorf("latitude must be between -90 and 90")
	}
	if math.IsNaN(req.Longitude) || math.IsInf(req.Longitude, 0) || req.Longitude < -180 || req.Longitude > 180 {
		return fmt.Errorf("longitude must be between -180 and 180")
	}
	if math.IsNaN(req.AccuracyMeters) || math.IsInf(req.AccuracyMeters, 0) || req.AccuracyMeters < 0 || req.AccuracyMeters > 100000 {
		return fmt.Errorf("accuracyMeters must be between 0 and 100000")
	}
	if len([]rune(strings.TrimSpace(req.Label))) > 120 {
		return fmt.Errorf("label must be at most 120 characters")
	}
	return nil
}

func sendLocationHandler(w http.ResponseWriter, r *http.Request) {
	var req sendLocationRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	req.ConversationID = strings.TrimSpace(req.ConversationID)
	req.Label = strings.TrimSpace(req.Label)
	if err := validateLocationRequest(req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if !requireConversationAccess(w, r, req.ConversationID) {
		return
	}
	now := time.Now().UTC()
	text := "Shared location"
	if req.Label != "" {
		text = req.Label
	}
	message := model.Message{ID: uuid.NewString(), TenantID: requestTenantID(r), ConversationID: req.ConversationID, SenderID: requestUserID(r), Sender: requestUserName(r), Text: text, CreatedAt: now, Status: "active", DeliveryStatus: "sent"}
	location := model.MessageLocation{MessageID: message.ID, Latitude: req.Latitude, Longitude: req.Longitude, AccuracyMeters: req.AccuracyMeters, Label: req.Label, CreatedAt: now}
	message.Location = &location
	if err := store.StoreLocationMessage(message, location); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to share location")
		return
	}
	store.BroadcastMessage(message)
	_ = store.NotifyConversationMembers(message.ConversationID, requestUserID(r), requestUserName(r), "Shared a location", fmt.Sprintf("/chat/%s", message.ConversationID), nil)
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, message)
}
