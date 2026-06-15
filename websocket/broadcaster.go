package websocket

import (
	"encoding/json"
	"log"
	"time"

	"Realify/database"
	"Realify/models"
)

// NotificationBroadcaster broadcasts task notifications to WebSocket clients
type NotificationBroadcaster struct {
	Manager   *Manager
	Publisher *NotificationPublisher
}

// NewNotificationBroadcaster creates a new notification broadcaster
func NewNotificationBroadcaster(manager *Manager) *NotificationBroadcaster {
	return &NotificationBroadcaster{
		Manager:   manager,
		Publisher: NewNotificationPublisher(),
	}
}

// BroadcastTaskNotification broadcasts a task notification
func (nb *NotificationBroadcaster) BroadcastTaskNotification(
	taskID string,
	taskType string,
	platform models.TaskPlatform,
	userID string,
	status models.TaskStatus,
	message string,
	progress int,
	details map[string]interface{},
	errMsg string,
) {
	// Create notification message
	notification := models.NotificationMessage{
		Type:      "notification",
		TaskID:    taskID,
		TaskType:  taskType,
		Platform:  string(platform),
		Status:    status,
		Message:   message,
		Progress:  progress,
		Timestamp: time.Now(),
		ErrorMsg:  errMsg,
		Details:   details,
	}

	// Publish to Redis
	if nb.Publisher != nil {
		if err := nb.Publisher.PublishNotification(notification); err != nil {
			log.Printf("Failed to publish notification to Redis: %v", err)
		}
	}

	// Save to database
	go func() {
		taskNotification := models.TaskNotification{
			TaskID:   taskID,
			TaskType: taskType,
			Platform: platform,
			UserID:   userID,
			Status:   status,
			Message:  message,
			Progress: progress,
			ErrorMsg: errMsg,
		}

		// Serialize details to JSON
		if len(details) > 0 {
			if detailsJSON, err := json.Marshal(details); err == nil {
				taskNotification.Details = string(detailsJSON)
			}
		}

		// Save to database
		if err := database.DB.Create(&taskNotification).Error; err != nil {
			log.Printf("Failed to save task notification to database: %v", err)
		}
	}()

	// Broadcast to user via WebSocket (if manager is available)
	if nb.Manager != nil {
		nb.Manager.BroadcastToUser(userID, notification)
	}

	log.Printf("Broadcasted notification for task %s (type: %s) to user %s: %s", taskID, taskType, userID, message)
}

// BroadcastTaskInitiated broadcasts a task initiated notification
func (nb *NotificationBroadcaster) BroadcastTaskInitiated(
	taskID string,
	taskType string,
	platform models.TaskPlatform,
	userID string,
) {
	nb.BroadcastTaskNotification(
		taskID,
		taskType,
		platform,
		userID,
		models.TaskStatusInitiated,
		"Task initiated",
		0,
		nil,
		"",
	)
}

// BroadcastTaskProcessing broadcasts a task processing notification
func (nb *NotificationBroadcaster) BroadcastTaskProcessing(
	taskID string,
	taskType string,
	platform models.TaskPlatform,
	userID string,
	message string,
	progress int,
	details map[string]interface{},
) {
	nb.BroadcastTaskNotification(
		taskID,
		taskType,
		platform,
		userID,
		models.TaskStatusProcessing,
		message,
		progress,
		details,
		"",
	)
}

// BroadcastTaskCompleted broadcasts a task completed notification
func (nb *NotificationBroadcaster) BroadcastTaskCompleted(
	taskID string,
	taskType string,
	platform models.TaskPlatform,
	userID string,
	message string,
	details map[string]interface{},
) {
	nb.BroadcastTaskNotification(
		taskID,
		taskType,
		platform,
		userID,
		models.TaskStatusCompleted,
		message,
		100,
		details,
		"",
	)
}

// BroadcastTaskFailed broadcasts a task failed notification
func (nb *NotificationBroadcaster) BroadcastTaskFailed(
	taskID string,
	taskType string,
	platform models.TaskPlatform,
	userID string,
	errorMsg string,
	details map[string]interface{},
) {
	nb.BroadcastTaskNotification(
		taskID,
		taskType,
		platform,
		userID,
		models.TaskStatusFailed,
		"Task failed",
		0,
		details,
		errorMsg,
	)
}

// GetTaskNotifications retrieves task notifications for a user
func (nb *NotificationBroadcaster) GetTaskNotifications(userID string, limit int) ([]models.TaskNotification, error) {
	var notifications []models.TaskNotification
	result := database.DB.
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&notifications)
	return notifications, result.Error
}

// GetTaskNotificationByID retrieves a specific task notification
func (nb *NotificationBroadcaster) GetTaskNotificationByID(taskID string) (*models.TaskNotification, error) {
	var notification models.TaskNotification
	result := database.DB.Where("task_id = ?", taskID).First(&notification)
	if result.Error != nil {
		return nil, result.Error
	}
	return &notification, nil
}
