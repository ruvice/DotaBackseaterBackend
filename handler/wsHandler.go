package handler

import (
	"log"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
)

// Struct representing a message sent to a WebSocket client
type WSMessage struct {
	EventType string      `json:"event"`
	Data      interface{} `json:"data"`
}

// Internal WebSocket client representation
type WSClient struct {
	conn      *websocket.Conn
	channelID string
	sendChan  chan WSMessage
}

// Handler managing all WebSocket clients
type WSHandler struct {
	clients     map[string][]*WSClient
	clientsLock sync.RWMutex
}

// WebSocket upgrader with permissive origin checks (secure this in production)
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Handle incoming WebSocket connections
func (h *WSHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	log.Println("Handling new connection")
	channelID := chi.URLParam(r, "channelID")
	if channelID == "" {
		http.Error(w, "Missing channelID", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade failed:", err)
		return
	}
	defer conn.Close()

	client := &WSClient{
		conn:      conn,
		channelID: channelID,
		sendChan:  make(chan WSMessage, 100),
	}

	h.registerClient(client)
	defer h.unregisterClient(client)

	// Goroutine for writing messages to the client
	go func() {
		for msg := range client.sendChan {
			if err := client.conn.WriteJSON(msg); err != nil {
				log.Println("WebSocket write error:", err)
				break
			}
		}
	}()

	// Optional read loop — if you expect messages from client
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Client disconnected from channel %s: %v", channelID, err)
			break
		}
		// No-op for now
	}
}

// Register a new WebSocket client
func (h *WSHandler) registerClient(client *WSClient) {
	h.clientsLock.Lock()
	defer h.clientsLock.Unlock()

	if h.clients == nil {
		h.clients = make(map[string][]*WSClient)
	}
	h.clients[client.channelID] = append(h.clients[client.channelID], client)
}

// Unregister a WebSocket client
func (h *WSHandler) unregisterClient(client *WSClient) {
	h.clientsLock.Lock()
	defer h.clientsLock.Unlock()

	clients := h.clients[client.channelID]
	for i, c := range clients {
		if c == client {
			h.clients[client.channelID] = append(clients[:i], clients[i+1:]...)
			break
		}
	}
	if len(h.clients[client.channelID]) == 0 {
		delete(h.clients, client.channelID)
	}
}

// Broadcast a message to all clients on a specific channel
func (h *WSHandler) BroadcastToChannel(channelID string, message WSMessage) {
	h.clientsLock.RLock()
	defer h.clientsLock.RUnlock()

	clients, exists := h.clients[channelID]
	if !exists {
		log.Printf("No WebSocket clients for channel %s", channelID)
		return
	}

	for _, client := range clients {
		select {
		case client.sendChan <- message:
		default:
			log.Printf("WebSocket sendChan for channel %s is blocked — skipping client", channelID)
		}
	}
}

// Background worker to consume push channel and fan out messages
func (h *WSHandler) StartWSPushWorker() {
	log.Println("Starting WSPushWorker")
	go func() {
		for push := range WSPushChannel {
			log.Printf("Broadcasting to channel %s: %+v", push.ChannelID, push.WSMessage)
			h.BroadcastToChannel(push.ChannelID, push.WSMessage)
		}
	}()
}
