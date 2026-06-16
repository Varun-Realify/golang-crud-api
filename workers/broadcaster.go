package workers

import "Realify/websocket"

var Broadcaster *websocket.NotificationBroadcaster

func InitBroadcaster(manager *websocket.Manager) {
	Broadcaster = websocket.NewNotificationBroadcaster(manager)
}

func SetBroadcaster(b *websocket.NotificationBroadcaster) {
	Broadcaster = b
}
