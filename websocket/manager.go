package websocket

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"Realify/models"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

// Manager handles WebSocket connections
type Manager struct {
	Hub          *Hub
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	PingInterval time.Duration
	PongWait     time.Duration
}

// NewManager creates a new WebSocket manager
func NewManager() *Manager {
	return &Manager{
		Hub:          NewHub(),
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		PingInterval: 30 * time.Second,
		PongWait:     60 * time.Second,
	}
}

// Start starts the manager and hub
func (m *Manager) Start() {
	go m.Hub.Run()
	go m.startRedisSubscriber()
}

func (m *Manager) startRedisSubscriber() {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "127.0.0.1:6379"
	}

	client := redis.NewClient(&redis.Options{Addr: redisAddr})
	ctx := context.Background()

	pubsub := client.PSubscribe(ctx, "notifications:*")
	if _, err := pubsub.Receive(ctx); err != nil {
		log.Printf("WebSocket Redis subscribe failed: %v", err)
		return
	}

	ch := pubsub.Channel()
	log.Printf("WebSocket manager subscribed to Redis notification channels: notifications:*")

	for msg := range ch {
		var notification models.NotificationMessage
		if err := json.Unmarshal([]byte(msg.Payload), &notification); err != nil {
			log.Printf("Failed to unmarshal notification payload: %v", err)
			continue
		}

		if notification.UserID != "" {
			m.BroadcastToUser(notification.UserID, notification)
		} else {
			m.BroadcastToAll(notification)
		}
	}
}

// HandleConnection handles a new WebSocket connection
func (m *Manager) HandleConnection(conn *websocket.Conn, userID string) {
	clientID := uuid.New().String()
	client := &Client{
		ID:     clientID,
		UserID: userID,
		Conn:   conn,
		Send:   make(chan interface{}, 256),
		Hub:    m.Hub,
	}

	m.Hub.register <- client

	// Set up connection options
	conn.SetReadDeadline(time.Now().Add(m.ReadTimeout))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(m.PongWait))
		return nil
	})

	// Start goroutines for reading and writing
	go m.readPump(client)
	go m.writePump(client)

	log.Printf("Client %s (UserID: %s) connected. Total clients: %d\n", clientID, userID, m.Hub.GetClientCount())
}

// readPump reads messages from the WebSocket connection
func (m *Manager) readPump(client *Client) {
	defer func() {
		client.Hub.unregister <- client
		client.Conn.Close()
		log.Printf("Client %s disconnected. Total clients: %d\n", client.ID, client.Hub.GetClientCount())
	}()

	for {
		_, message, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			return
		}

		// Parse incoming message
		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		// Handle ping/pong
		if msgType, ok := msg["type"].(string); ok && msgType == "ping" {
			pongMsg := map[string]interface{}{
				"type":      "pong",
				"timestamp": time.Now(),
			}
			client.Send <- pongMsg
		}
	}
}

// writePump writes messages to the WebSocket connection
func (m *Manager) writePump(client *Client) {
	ticker := time.NewTicker(m.PingInterval)
	defer func() {
		ticker.Stop()
		client.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			client.Conn.SetWriteDeadline(time.Now().Add(m.WriteTimeout))
			if !ok {
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := client.Conn.WriteJSON(message); err != nil {
				return
			}

		case <-ticker.C:
			client.Conn.SetWriteDeadline(time.Now().Add(m.WriteTimeout))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// BroadcastToUser sends a message to all connections of a specific user
func (m *Manager) BroadcastToUser(userID string, message interface{}) {
	m.Hub.BroadcastToUser(userID, message)
}

// BroadcastToAll sends a message to all connected clients
func (m *Manager) BroadcastToAll(message interface{}) {
	m.Hub.broadcast <- message
}

// GetStats returns connection statistics
func (m *Manager) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"total_clients": m.Hub.GetClientCount(),
	}
}
