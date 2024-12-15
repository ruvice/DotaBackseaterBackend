package handler

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
)

type EventHandler struct {
	clients     map[string][]chan SSEMessage // Map of ChannelID to client channels
	clientsLock sync.RWMutex                 // Protect concurrent access to clients
}

// Register a new client for a ChannelID
func (h *EventHandler) registerClient(channelID string, clientChan chan SSEMessage) {
	h.clientsLock.Lock()
	defer h.clientsLock.Unlock()

	if h.clients == nil {
		h.clients = make(map[string][]chan SSEMessage)
	}

	h.clients[channelID] = append(h.clients[channelID], clientChan)
}

// Unregister a client for a ChannelID
func (h *EventHandler) unregisterClient(channelID string, clientChan chan SSEMessage) {
	h.clientsLock.Lock()
	defer h.clientsLock.Unlock()

	clients := h.clients[channelID]
	for i, ch := range clients {
		if ch == clientChan {
			// Remove the client from the list
			h.clients[channelID] = append(clients[:i], clients[i+1:]...)
			break
		}
	}

	// Clean up the map entry if no clients remain
	if len(h.clients[channelID]) == 0 {
		delete(h.clients, channelID)
	}
}

func (h *EventHandler) EstablishSSEConnection(w http.ResponseWriter, r *http.Request) {
	channelID := chi.URLParam(r, "channelID")
	if channelID == "" {
		http.Error(w, "Missing channelID", http.StatusBadRequest)
		return
	}

	log.Printf("Received SSE connection request for ChannelID: %s\n", channelID)

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Ensure the connection supports streaming
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	// Create a new channel for this client
	clientChan := make(chan SSEMessage, 100)
	defer close(clientChan) // Ensure the channel is cleaned up

	// Register the client for this ChannelID
	h.registerClient(channelID, clientChan)
	defer h.unregisterClient(channelID, clientChan)

	// Listen for messages or client disconnect
	ctx := r.Context()
	for {
		select {
		case msg := <-clientChan: // Send message to client
			log.Println("DEBUG: ", msg)
			fmt.Fprintf(w, "event: %s\ndata: %v\n\n", msg.EventType, msg.Data)
			flusher.Flush()
		case <-ctx.Done(): // Client disconnected
			log.Printf("Client disconnected from ChannelID: %s\n", channelID)
			return
		}
	}
}

func (h *EventHandler) broadcastToChannel(channelID string, message SSEMessage) {
	h.clientsLock.RLock()
	defer h.clientsLock.RUnlock()

	clients, exists := h.clients[channelID]
	if !exists {
		log.Printf("No active clients for ChannelID: %s\n", channelID)
		return
	}

	for _, clientChan := range clients {
		// Non-blocking send to avoid slow or disconnected clients
		select {
		case clientChan <- message:
		default:
			log.Printf("Client channel for ChannelID %s is blocked, skipping...\n", channelID)
		}
	}
}

func (h *EventHandler) StartSSEPushWorker() {
	go func() {
		for pushRequest := range SSEPushChannel {
			log.Printf("Processing SSEPushRequest: ChannelID=%s, EventType=%s\n, Data=%v\n", pushRequest.ChannelID, pushRequest.SSEMessage.EventType, pushRequest.SSEMessage.Data)
			h.broadcastToChannel(pushRequest.ChannelID, pushRequest.SSEMessage)
		}
	}()
}
