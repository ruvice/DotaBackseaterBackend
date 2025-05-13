package handler

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
)

const (
	pongWait   = 60 * time.Second
	pingPeriod = 50 * time.Second
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

	client := &WSClient{
		conn:      conn,
		channelID: channelID,
		sendChan:  make(chan WSMessage, 100),
	}

	h.registerClient(client)
	defer h.unregisterClient(client)

	// Graceful shutdown helpers
	done := make(chan struct{})
	var once sync.Once
	cleanup := func() {
		once.Do(func() {
			log.Println("Cleaning up connection")
			close(done)            // stop ping loop
			conn.Close()           // close conn
			close(client.sendChan) // end writer
		})
	}
	defer cleanup()

	// Set read deadline and pong handler
	conn.SetReadLimit(512)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	// Start ping loop
	go func() {
		ticker := time.NewTicker(pingPeriod)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					log.Println("Ping failed:", err)
					cleanup()
					return
				}
			case <-done:
				return
			}
		}
	}()

	// Start write loop
	go func() {
		for msg := range client.sendChan {
			if err := conn.WriteJSON(msg); err != nil {
				log.Println("WebSocket write error:", err)
				cleanup()
				return
			}
		}
	}()

	// Read loop — blocks until connection closes or times out
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Client disconnected from channel %s: %v", channelID, err)
			break
		}
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
