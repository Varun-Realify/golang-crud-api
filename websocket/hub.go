package websocket

import (
	"sync"

	"github.com/gorilla/websocket"
)

// Client represents a WebSocket client connection
type Client struct {
	ID     string
	UserID string
	Conn   *websocket.Conn
	Send   chan interface{}
	Hub    *Hub
}

// Hub maintains active client connections and broadcasts messages
type Hub struct {
	// Map of user ID to set of connected clients
	clients    map[string]map[*Client]bool
	broadcast  chan interface{}
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]map[*Client]bool),
		broadcast:  make(chan interface{}, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub's event loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case message := <-h.broadcast:
			h.broadcastToAll(message)
		}
	}
}

// registerClient adds a client to the hub
func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.clients[client.UserID]; !exists {
		h.clients[client.UserID] = make(map[*Client]bool)
	}
	h.clients[client.UserID][client] = true
}

// unregisterClient removes a client from the hub
func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if userClients, exists := h.clients[client.UserID]; exists {
		if _, ok := userClients[client]; ok {
			delete(userClients, client)
			close(client.Send)
			if len(userClients) == 0 {
				delete(h.clients, client.UserID)
			}
		}
	}
}

// BroadcastToUser sends a message to all connections of a specific user
func (h *Hub) BroadcastToUser(userID string, message interface{}) {
	h.mu.RLock()
	clients, exists := h.clients[userID]
	h.mu.RUnlock()

	if !exists {
		return
	}

	for client := range clients {
		select {
		case client.Send <- message:
		default:
			// Client's send channel is full, close it
			go h.unregisterClient(client)
		}
	}
}

// broadcastToAll sends a message to all connected clients
func (h *Hub) broadcastToAll(message interface{}) {
	h.mu.RLock()
	clientsMap := make(map[string]map[*Client]bool)
	for userID, clients := range h.clients {
		for client := range clients {
			if _, exists := clientsMap[userID]; !exists {
				clientsMap[userID] = make(map[*Client]bool)
			}
			clientsMap[userID][client] = true
		}
	}
	h.mu.RUnlock()

	for _, clients := range clientsMap {
		for client := range clients {
			select {
			case client.Send <- message:
			default:
				// Client's send channel is full, close it
				go h.unregisterClient(client)
			}
		}
	}
}

// GetClientCount returns the number of connected clients
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	count := 0
	for _, clients := range h.clients {
		count += len(clients)
	}
	return count
}

// GetUserClientCount returns the number of connected clients for a specific user
func (h *Hub) GetUserClientCount(userID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, exists := h.clients[userID]; exists {
		return len(clients)
	}
	return 0
}
