package ws

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // In production, restrict to allowed origins
	},
}

// Hub manages WebSocket connections grouped by user ID.
type Hub struct {
	// clients maps user IDs to their active WebSocket connections
	clients map[string]map[*websocket.Conn]bool
	mu      sync.RWMutex
}

// NewHub creates a new WebSocket hub.
func NewHub() *Hub {
	return &Hub{
		clients: make(map[string]map[*websocket.Conn]bool),
	}
}

// HandleWebSocket upgrades an HTTP connection to WebSocket.
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "missing user_id", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	h.addClient(userID, conn)
	log.Printf("🔌 WebSocket connected: user=%s", userID)

	// Keep connection alive and handle disconnects
	go h.readPump(userID, conn)
}

func (h *Hub) addClient(userID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.clients[userID] == nil {
		h.clients[userID] = make(map[*websocket.Conn]bool)
	}
	h.clients[userID][conn] = true
}

func (h *Hub) removeClient(userID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if conns, ok := h.clients[userID]; ok {
		delete(conns, conn)
		if len(conns) == 0 {
			delete(h.clients, userID)
		}
	}
	conn.Close()
}

// readPump reads messages from the client (mainly to detect disconnects).
func (h *Hub) readPump(userID string, conn *websocket.Conn) {
	defer h.removeClient(userID, conn)
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			log.Printf("🔌 WebSocket disconnected: user=%s", userID)
			return
		}
	}
}

// SendToUser sends a JSON message to all WebSocket connections for a user.
func (h *Hub) SendToUser(userID string, message interface{}) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	conns, ok := h.clients[userID]
	if !ok {
		return
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal WS message: %v", err)
		return
	}

	for conn := range conns {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("Failed to send WS message: %v", err)
			go h.removeClient(userID, conn)
		}
	}
}

// ListenToRedis subscribes to Redis pub/sub and forwards status updates to WebSocket clients.
func (h *Hub) ListenToRedis(ctx context.Context, rdb *redis.Client, channel string) {
	pubsub := rdb.Subscribe(ctx, channel)
	defer pubsub.Close()

	log.Printf("📡 Listening for status updates on Redis channel: %s", channel)

	for msg := range pubsub.Channel() {
		var update map[string]string
		if err := json.Unmarshal([]byte(msg.Payload), &update); err != nil {
			log.Printf("Failed to parse Redis message: %v", err)
			continue
		}

		// The update contains file_id, status, message
		// We need to find which user owns this file and notify them
		// For MVP, the worker includes user_id in the status update
		userID := update["user_id"]
		if userID != "" {
			h.SendToUser(userID, update)
		}
	}
}
