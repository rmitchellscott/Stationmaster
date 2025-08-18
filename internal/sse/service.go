package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Event represents a server-sent event
type Event struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// Client represents a connected SSE client
type Client struct {
	ID       string
	DeviceID uuid.UUID
	UserID   uuid.UUID
	Writer   http.ResponseWriter
	Flusher  http.Flusher
	Done     chan bool
}

// Service manages SSE connections and broadcasts
type Service struct {
	mu      sync.RWMutex
	clients map[string]*Client
}

// NewService creates a new SSE service
func NewService() *Service {
	return &Service{
		clients: make(map[string]*Client),
	}
}

// AddClient adds a new SSE client connection
func (s *Service) AddClient(deviceID, userID uuid.UUID, w http.ResponseWriter) *Client {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

	clientID := fmt.Sprintf("%s-%s-%d", deviceID.String(), userID.String(), time.Now().UnixNano())

	client := &Client{
		ID:       clientID,
		DeviceID: deviceID,
		UserID:   userID,
		Writer:   w,
		Flusher:  flusher,
		Done:     make(chan bool),
	}

	s.mu.Lock()
	s.clients[clientID] = client
	s.mu.Unlock()

	log.Printf("[SSE] Client connected: %s for device %s", clientID, deviceID.String())

	// Send initial connection event
	s.sendToClient(client, Event{
		Type: "connected",
		Data: map[string]interface{}{
			"device_id": deviceID.String(),
			"timestamp": time.Now().UTC(),
		},
	})

	return client
}

// RemoveClient removes a client connection
func (s *Service) RemoveClient(clientID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if client, exists := s.clients[clientID]; exists {
		close(client.Done)
		delete(s.clients, clientID)
		log.Printf("[SSE] Client disconnected: %s", clientID)
	}
}

// BroadcastToDevice sends an event to all clients connected to a specific device
func (s *Service) BroadcastToDevice(deviceID uuid.UUID, event Event) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, client := range s.clients {
		if client.DeviceID == deviceID {
			s.sendToClient(client, event)
		}
	}
}

// BroadcastToUser sends an event to all clients connected by a specific user
func (s *Service) BroadcastToUser(userID uuid.UUID, event Event) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, client := range s.clients {
		if client.UserID == userID {
			s.sendToClient(client, event)
		}
	}
}

// sendToClient sends an event to a specific client
func (s *Service) sendToClient(client *Client, event Event) {
	eventData, err := json.Marshal(event)
	if err != nil {
		log.Printf("[SSE] Failed to marshal event: %v", err)
		return
	}

	// Send event in SSE format
	fmt.Fprintf(client.Writer, "data: %s\n\n", eventData)
	client.Flusher.Flush()
}

// KeepAlive sends periodic keep-alive events to maintain connections
func (s *Service) KeepAlive(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.mu.RLock()
			for _, client := range s.clients {
				s.sendToClient(client, Event{
					Type: "ping",
					Data: map[string]interface{}{
						"timestamp": time.Now().UTC(),
					},
				})
			}
			s.mu.RUnlock()
		}
	}
}

// GetClientCount returns the number of connected clients
func (s *Service) GetClientCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.clients)
}

// GetDeviceClientCount returns the number of clients connected to a specific device
func (s *Service) GetDeviceClientCount(deviceID uuid.UUID) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, client := range s.clients {
		if client.DeviceID == deviceID {
			count++
		}
	}
	return count
}

// Global SSE service instance
var globalSSEService *Service

// InitializeSSEService initializes the global SSE service
func InitializeSSEService() {
	globalSSEService = NewService()
}

// GetSSEService returns the global SSE service instance
func GetSSEService() *Service {
	if globalSSEService == nil {
		InitializeSSEService()
	}
	return globalSSEService
}
