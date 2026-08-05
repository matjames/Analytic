package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"statchat/pkg/model"
	"statchat/pkg/store"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// ── Call Session Lifecycle ──

type createCallRequest struct {
	Kind           model.CallKind `json:"kind"`
	RoomName       string         `json:"roomName"`
	ConversationID string         `json:"conversationId,omitempty"`
	HostID         string         `json:"hostId,omitempty"`
	HostName       string         `json:"hostName,omitempty"`
}

func createCallSessionHandler(w http.ResponseWriter, r *http.Request) {
	currentUser, err := requestCurrentUser(r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load current user")
		return
	}

	var req createCallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	if req.Kind == "" {
		req.Kind = model.CallKindVideo
	}
	if req.RoomName == "" {
		req.RoomName = fmt.Sprintf("%s's %s call", currentUser.Name, req.Kind)
	}
	if req.HostID == "" || req.HostID != currentUser.ID {
		req.HostID = currentUser.ID
	}
	if req.HostName == "" || req.HostName != currentUser.Name {
		req.HostName = currentUser.Name
	}

	session, err := store.CreateCallSession(model.CallSession{
		RoomName:     req.RoomName,
		Kind:         req.Kind,
		HostID:       req.HostID,
		HostName:     req.HostName,
		Status:       model.CallStatusLive,
		Conversation: req.ConversationID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create call session")
		return
	}

	// Auto-join host as first participant.
	if _, err := store.JoinCallSession(session.ID, req.HostID, req.HostName); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to join host to session")
		return
	}

	writeJSON(w, session)
}

func listCallSessionsHandler(w http.ResponseWriter, r *http.Request) {
	sessions, err := store.ListActiveCallSessions()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list call sessions")
		return
	}
	writeJSON(w, sessions)
}

func getCallSessionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	session, err := store.GetCallSession(vars["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "call session not found")
		return
	}
	participants, err := store.GetActiveParticipants(session.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load participants")
		return
	}
	writeJSON(w, map[string]interface{}{
		"session":      session,
		"participants": participants,
	})
}

type joinCallRequest struct {
	UserID   string `json:"userId,omitempty"`
	UserName string `json:"userName,omitempty"`
}

func joinCallSessionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	session, err := store.GetCallSession(sessionID)
	if err != nil {
		writeError(w, http.StatusNotFound, "call session not found")
		return
	}
	if session.Status != model.CallStatusLive {
		writeError(w, http.StatusConflict, "call session is not live")
		return
	}

	var req joinCallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	currentUser, err := requestCurrentUser(r)
	if err != nil {
		currentUser = model.User{ID: "user-001", Name: "StatChat User"}
	}
	if req.UserID == "" || req.UserID != currentUser.ID {
		req.UserID = currentUser.ID
	}
	if req.UserName == "" {
		req.UserName = currentUser.Name
	}

	participant, err := store.JoinCallSession(sessionID, req.UserID, req.UserName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to join call session")
		return
	}

	// Broadcast participant joined to the room.
	store.BroadcastCallState(sessionID, "call-participant-joined", participant)
	writeJSON(w, participant)
}

func leaveCallSessionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	var req struct {
		UserID string `json:"userId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	if req.UserID == "" {
		req.UserID = requestUserID(r)
	}

	if err := store.LeaveCallSession(sessionID, req.UserID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to leave call session")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func endCallSessionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	session, err := store.EndCallSession(sessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to end call session")
		return
	}
	store.BroadcastCallState(sessionID, "call-ended", map[string]string{"sessionId": sessionID})
	writeJSON(w, session)
}

func getCallParticipantsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	participants, err := store.GetActiveParticipants(vars["id"])
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load participants")
		return
	}
	writeJSON(w, participants)
}

// ── Call Recordings ──

type callRecordingUploadRequest struct {
	Title string
}

func uploadCallRecordingHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	if _, err := store.GetCallSession(sessionID); err != nil {
		writeError(w, http.StatusNotFound, "call session not found")
		return
	}

	if err := r.ParseMultipartForm(256 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart upload")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	// Save recordings into the uploads directory so the existing
	// /uploads/ file server can serve them (matches the stored URL).
	dir := strings.TrimSpace(os.Getenv("STATCHAT_UPLOAD_DIR"))
	if dir == "" {
		dir = filepath.Join(".", "uploads")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create recording dir")
		return
	}

	extension := strings.ToLower(filepath.Ext(header.Filename))
	safeName := fmt.Sprintf("%s%s", uuid.NewString(), extension)
	destinationPath := filepath.Join(dir, safeName)
	destinationFile, err := os.Create(destinationPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create recording file")
		return
	}
	defer destinationFile.Close()

	size, err := io.Copy(destinationFile, file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save recording")
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		title = "Meeting recording " + time.Now().UTC().Format("2006-01-02 15:04")
	}

	recording := model.CallRecording{
		SessionID: sessionID,
		Title:     title,
		FileName:  header.Filename,
		URL:       fmt.Sprintf("/uploads/%s", safeName),
		Size:      size,
		Duration:  strings.TrimSpace(r.FormValue("duration")),
		CreatedAt: time.Now().UTC(),
	}
	saved, err := store.SaveCallRecording(recording)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save recording metadata")
		return
	}

	writeJSON(w, saved)
}

func listCallRecordingsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	recordings, err := store.GetCallRecordings(vars["id"])
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load recordings")
		return
	}
	writeJSON(w, recordings)
}

// ── Meeting → Call session linking ──

func getMeetingSessionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	meetingID := vars["id"]

	// Find an active/live call session associated with this meeting room.
	// The meeting room url encodes a room token; try to find by room name matching title.
	var meetings, err = store.GetMeetings()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load meetings")
		return
	}
	for _, m := range meetings {
		if m.ID != meetingID {
			continue
		}
		session, err := store.FindCallSessionByRoom(m.Room)
		if err == nil && session.ID != "" {
			writeJSON(w, session)
			return
		}
		// No live session yet — create one and link to the meeting.
		currentUser, err := requestCurrentUser(r)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to load current user")
			return
		}
		session, err = store.CreateCallSession(model.CallSession{
			RoomName: m.Title,
			RoomID:   m.Room,
			Kind:     model.CallKindVideo,
			HostID:   currentUser.ID,
			HostName: currentUser.Name,
			Status:   model.CallStatusLive,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to create meeting session")
			return
		}
		if _, err := store.JoinCallSession(session.ID, currentUser.ID, currentUser.Name); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to join meeting host to session")
			return
		}
		writeJSON(w, session)
		return
	}

	writeError(w, http.StatusNotFound, "meeting not found")
}
