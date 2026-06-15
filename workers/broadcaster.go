package workers

import (
	"Realify/websocket"
)

// Global broadcaster instance
var Broadcaster *websocket.NotificationBroadcaster

// InitBroadcaster initializes the global notification broadcaster
// This should be called from the worker after the WebSocket manager is created
func InitBroadcaster(manager *websocket.Manager) {
	Broadcaster = websocket.NewNotificationBroadcaster(manager)
}

// SetBroadcaster sets the global broadcaster instance
// This is used to inject the broadcaster from the API server
func SetBroadcaster(b *websocket.NotificationBroadcaster) {
	Broadcaster = b
}
