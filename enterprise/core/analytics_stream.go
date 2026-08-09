package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ─── Real-Time Analytics Stream ─────────────────────────────────────
// Server-Sent Events (SSE) broker for live dashboard updates.
// Dashboards update when meaningful events occur without requiring
// manual refresh.

type StreamClient struct {
	ch chan []byte
}

var (
	streamClients    = make(map[*StreamClient]bool)
	streamMu         sync.RWMutex
	streamRegister   = make(chan *StreamClient, 64)
	streamUnregister = make(chan *StreamClient, 64)
)

// startAnalyticsStreamBroker runs the SSE fan-out broker.
func startAnalyticsStreamBroker() {
	go func() {
		for {
			select {
			case c := <-streamRegister:
				streamMu.Lock()
				streamClients[c] = true
				streamMu.Unlock()
			case c := <-streamUnregister:
				streamMu.Lock()
				if _, ok := streamClients[c]; ok {
					delete(streamClients, c)
					close(c.ch)
				}
				streamMu.Unlock()
			}
		}
	}()
}

// broadcastToStreamClients sends a message to all connected SSE clients.
func broadcastToStreamClients(eventType string, payload map[string]interface{}) {
	data, _ := json.Marshal(map[string]interface{}{
		"type": eventType,
		"data": payload,
		"ts":   time.Now().Unix(),
	})
	streamMu.RLock()
	for c := range streamClients {
		select {
		case c.ch <- data:
		default:
		}
	}
	streamMu.RUnlock()
}

// handleAnalyticsStream is the SSE endpoint.
func handleAnalyticsStream(c *gin.Context) {
	client := &StreamClient{ch: make(chan []byte, 64)}
	streamRegister <- client
	defer func() { streamUnregister <- client }()

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// Send initial welcome
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(500, gin.H{"error": "streaming not supported"})
		return
	}

	welcome := map[string]interface{}{
		"type": "connected",
		"data": map[string]interface{}{
			"message":   "Connected to StatGate Analytics Stream",
			"timestamp": nowUTC(),
			"version":   version,
		},
		"ts": time.Now().Unix(),
	}
	w, _ := json.Marshal(welcome)
	fmt.Fprintf(c.Writer, "data: %s\n\n", w)
	flusher.Flush()

	// Heartbeat ticker
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case msg, ok := <-client.ch:
			if !ok {
				return
			}
			fmt.Fprintf(c.Writer, "data: %s\n\n", msg)
			flusher.Flush()
		case <-ticker.C:
			// Heartbeat
			fmt.Fprintf(c.Writer, ": ping\n\n")
			flusher.Flush()
		}
	}
}

// startAnalyticsEventSubscriber subscribes to the Redis analytics channel
// and forwards events to SSE clients.
func startAnalyticsEventSubscriber() {
	go func() {
		if redisClient == nil {
			log.Println("analytics: Redis unavailable; event subscriber disabled")
			return
		}
		ctx := context.Background()
		pubsub := redisClient.Subscribe(ctx, "statgate:analytics")
		defer pubsub.Close()
		log.Println("analytics: subscribing to statgate:analytics for real-time updates")

		for msg := range pubsub.Channel() {
			// Parse and forward to SSE clients
			var ev map[string]interface{}
			if err := json.Unmarshal([]byte(msg.Payload), &ev); err != nil {
				continue
			}
			eventType := "analytics.update"
			if t, ok := ev["type"].(string); ok {
				eventType = t
			}
			payload := map[string]interface{}{}
			if p, ok := ev["payload"].(map[string]interface{}); ok {
				payload = p
			}
			broadcastToStreamClients(eventType, payload)
		}
	}()
}

// publishAnalyticsBroadcast publishes a broadcast message to stream clients
// via Redis (for cross-instance fan-out) and directly to local clients.
func publishAnalyticsBroadcast(eventType string, payload map[string]interface{}) {
	publishAnalyticsEvent(eventType, payload)
	broadcastToStreamClients(eventType, payload)
}
