package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// NotificationMessage represents a WebSocket notification
type NotificationMessage struct {
	Type      string                 `json:"type"`
	TaskID    string                 `json:"task_id"`
	TaskType  string                 `json:"task_type"`
	Platform  string                 `json:"platform"`
	Status    string                 `json:"status"`
	Message   string                 `json:"message"`
	Progress  int                    `json:"progress"`
	Timestamp time.Time              `json:"timestamp"`
	ErrorMsg  string                 `json:"error_msg,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// WebSocketClient is an example WebSocket client for receiving notifications
type WebSocketClient struct {
	url    string
	userID string
	conn   *websocket.Conn
	done   chan struct{}
	ticker *time.Ticker
}

// NewWebSocketClient creates a new WebSocket client
func NewWebSocketClient(host string, userID string) *WebSocketClient {
	return &WebSocketClient{
		url:    fmt.Sprintf("ws://%s/ws", host),
		userID: userID,
		done:   make(chan struct{}),
	}
}

// Connect establishes a WebSocket connection
func (c *WebSocketClient) Connect() error {
	u := url.URL{
		Scheme:   "ws",
		Host:     c.url[5:], // Remove "ws://" prefix
		Path:     "/ws",
		RawQuery: fmt.Sprintf("user_id=%s", c.url_encode(c.userID)),
	}

	log.Printf("Connecting to %s", u.String())

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return err
	}

	c.conn = conn

	// Start reading messages in a goroutine
	go c.readMessages()

	// Start ping ticker
	c.ticker = time.NewTicker(30 * time.Second)
	go c.pingTicker()

	return nil
}

// readMessages reads messages from the WebSocket connection
func (c *WebSocketClient) readMessages() {
	defer close(c.done)

	for {
		var msg NotificationMessage
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			return
		}

		c.handleNotification(msg)
	}
}

// pingTicker sends periodic pings
func (c *WebSocketClient) pingTicker() {
	defer c.ticker.Stop()

	for {
		select {
		case <-c.done:
			return
		case <-c.ticker.C:
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Error sending ping: %v", err)
				return
			}
		}
	}
}

// handleNotification processes a received notification
func (c *WebSocketClient) handleNotification(msg NotificationMessage) {
	fmt.Printf("\n=== NOTIFICATION ===\n")
	fmt.Printf("Task ID: %s\n", msg.TaskID)
	fmt.Printf("Task Type: %s\n", msg.TaskType)
	fmt.Printf("Platform: %s\n", msg.Platform)
	fmt.Printf("Status: %s\n", msg.Status)
	fmt.Printf("Message: %s\n", msg.Message)
	fmt.Printf("Progress: %d%%\n", msg.Progress)
	fmt.Printf("Timestamp: %s\n", msg.Timestamp.Format(time.RFC3339))

	if msg.ErrorMsg != "" {
		fmt.Printf("Error: %s\n", msg.ErrorMsg)
	}

	if len(msg.Details) > 0 {
		detailsJSON, _ := json.MarshalIndent(msg.Details, "", "  ")
		fmt.Printf("Details: %s\n", string(detailsJSON))
	}

	fmt.Printf("==================\n\n")

	// Handle different status types
	switch msg.Status {
	case "initiated":
		fmt.Printf("✓ Task initiated\n")
	case "processing":
		fmt.Printf("⟳ Task processing... (%d%%)\n", msg.Progress)
	case "completed":
		fmt.Printf("✓ Task completed!\n")
	case "failed":
		fmt.Printf("✗ Task failed: %s\n", msg.ErrorMsg)
	}
}

// Close closes the WebSocket connection
func (c *WebSocketClient) Close() error {
	return c.conn.WriteMessage(websocket.CloseMessage, []byte{})
}

// url_encode encodes a string for URL
func (c *WebSocketClient) url_encode(s string) string {
	return url.QueryEscape(s)
}

// Example usage
func main() {
	// Generate a user ID for testing
	userID := uuid.New().String()
	fmt.Printf("User ID: %s\n", userID)

	// Create client
	client := NewWebSocketClient("localhost:8080", userID)

	// Connect to WebSocket
	if err := client.Connect(); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	fmt.Println("Connected to notification service. Waiting for messages...")
	fmt.Println("Press Ctrl+C to exit.")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	fmt.Println("\nDisconnecting...")
}
