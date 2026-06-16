package websocket

import (
	"encoding/json"
	"log"
	"time"

	"Realify/database"
	"Realify/models"
)

type NotificationBroadcaster struct {
	Manager   *Manager
	Publisher *NotificationPublisher
}

func NewNotificationBroadcaster(manager *Manager) *NotificationBroadcaster {
	return &NotificationBroadcaster{
		Manager:   manager,
		Publisher: NewNotificationPublisher(),
	}
}

func (nb *NotificationBroadcaster) BroadcastTaskNotification(
	taskID string, taskType string, platform models.TaskPlatform,
	userID string, status models.TaskStatus, message string,
	progress int, details map[string]interface{}, errMsg string,
) {
	notification := models.NotificationMessage{
		Type: "notification", TaskID: taskID, TaskType: taskType,
		Platform: string(platform), Status: status, Message: message,
		Progress: progress, Timestamp: time.Now(), ErrorMsg: errMsg, Details: details,
	}
	if nb.Publisher != nil {
		if err := nb.Publisher.PublishNotification(notification); err != nil {
			log.Printf("Failed to publish notification to Redis: %v", err)
		}
	}
	go func() {
		taskNotification := models.TaskNotification{
			TaskID: taskID, TaskType: taskType, Platform: platform,
			UserID: userID, Status: status, Message: message,
			Progress: progress, ErrorMsg: errMsg,
		}
		if len(details) > 0 {
			if detailsJSON, err := json.Marshal(details); err == nil {
				taskNotification.Details = string(detailsJSON)
			}
		}
		if err := database.DB.Create(&taskNotification).Error; err != nil {
			log.Printf("Failed to save task notification to database: %v", err)
		}
	}()
	if nb.Manager != nil {
		nb.Manager.BroadcastToUser(userID, notification)
	}
	log.Printf("Broadcasted notification for task %s (type: %s) to user %s: %s", taskID, taskType, userID, message)
}

// func (nb *NotificationBroadcaster) BroadcastTaskInitiated(...) { ... }
// func (nb *NotificationBroadcaster) BroadcastTaskProcessing(...) { ... }
// func (nb *NotificationBroadcaster) BroadcastTaskCompleted(...) { ... }
// func (nb *NotificationBroadcaster) BroadcastTaskFailed(...) { ... }
// func (nb *NotificationBroadcaster) GetTaskNotifications(...) { ... }
// func (nb *NotificationBroadcaster) GetTaskNotificationByID(...) { ... }
