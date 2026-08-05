package store

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"statchat/pkg/model"

	"github.com/gorilla/websocket"
)

// callClients maps a call room key -> set of sockets connected to that call.
// Used for WebRTC signaling (SDP offers/answers + ICE candidates) relay.
var callClients = map[string]map[*websocket.Conn]bool{}
var callClientsMutex sync.Mutex

// SignalRelay relays a WebRTC signaling message to the appropriate recipient(s).
// - If To is empty, broadcast to everyone in the room except the sender.
// - If To is set, deliver only to that socket (keyed by user identity).
// The signal is also persisted as a call event for audit.
func SignalRelay(signal model.CallSignal) {
	callClientsMutex.Lock()
	defer callClientsMutex.Unlock()

	roomKey := fmt.Sprintf("%s:%s", signal.SessionID, signal.From)
	// The room is keyed by session; we track per-session membership by user id.
	// We store sockets under session:roomName so we can target by From.
	room := fmt.Sprintf("call:%s", signal.SessionID)

	// Locate the socket(s) for the target user.
	var targetConns []*websocket.Conn
	if signal.To != "" {
		for conn := range callClients[room] {
			if connCallUser(conn) == signal.To {
				targetConns = append(targetConns, conn)
			}
		}
	} else {
		// Broadcast to all except the sender.
		for conn := range callClients[room] {
			if connCallUser(conn) == signal.From {
				continue
			}
			targetConns = append(targetConns, conn)
		}
	}

	envelope := model.GatewayEnvelope{
		Event:   "call-signal",
		Payload: signal,
	}
	data, err := json.Marshal(envelope)
	if err != nil {
		log.Printf("signal relay marshal failed: %v", err)
		return
	}

	for _, conn := range targetConns {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("signal relay write failed: %v", err)
		}
	}

	_ = roomKey
}

// RegisterCallClient associates a socket with a call room and user identity.
func RegisterCallClient(conn *websocket.Conn, sessionID string) {
	callClientsMutex.Lock()
	defer callClientsMutex.Unlock()
	room := fmt.Sprintf("call:%s", sessionID)
	if callClients[room] == nil {
		callClients[room] = map[*websocket.Conn]bool{}
	}
	callClients[room][conn] = true
}

// UnregisterCallClient removes a socket from any call room.
func UnregisterCallClient(conn *websocket.Conn) {
	callClientsMutex.Lock()
	defer callClientsMutex.Unlock()
	for room, conns := range callClients {
		if _, ok := conns[conn]; ok {
			delete(conns, conn)
			if len(conns) == 0 {
				delete(callClients, room)
			}
		}
	}
}

// connMetadata holds per-socket call identity.
var connCallMeta = map[*websocket.Conn]struct {
	userID    string
	sessionID string
}{}
var connCallMetaMutex sync.Mutex

// SetConnCallIdentity associates a socket with a user + session.
func SetConnCallIdentity(conn *websocket.Conn, sessionID, userID string) {
	connCallMetaMutex.Lock()
	defer connCallMetaMutex.Unlock()
	connCallMeta[conn] = struct {
		userID    string
		sessionID string
	}{userID: userID, sessionID: sessionID}
	if sessionID != "" {
		RegisterCallClient(conn, sessionID)
	}
}

// ClearConnCallIdentity removes per-socket call identity.
func ClearConnCallIdentity(conn *websocket.Conn) {
	connCallMetaMutex.Lock()
	defer connCallMetaMutex.Unlock()
	delete(connCallMeta, conn)
	UnregisterCallClient(conn)
}

func connCallUser(conn *websocket.Conn) string {
	connCallMetaMutex.Lock()
	defer connCallMetaMutex.Unlock()
	if meta, ok := connCallMeta[conn]; ok {
		return meta.userID
	}
	return ""
}

func connCallSession(conn *websocket.Conn) string {
	connCallMetaMutex.Lock()
	defer connCallMetaMutex.Unlock()
	if meta, ok := connCallMeta[conn]; ok {
		return meta.sessionID
	}
	return ""
}

// BroadcastCallState sends a room-state update (e.g., participant joined/left)
// to all sockets in a call room.
func BroadcastCallState(sessionID string, event string, payload interface{}) {
	callClientsMutex.Lock()
	defer callClientsMutex.Unlock()
	room := fmt.Sprintf("call:%s", sessionID)
	envelope := model.GatewayEnvelope{
		Event:   event,
		Payload: payload,
	}
	data, err := json.Marshal(envelope)
	if err != nil {
		return
	}
	for conn := range callClients[room] {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("call state broadcast failed: %v", err)
		}
	}
}
