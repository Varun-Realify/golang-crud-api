package models

import (
	"time"
)

// TaskStatus represents the status of a task
type TaskStatus string

const (
	TaskStatusInitiated  TaskStatus = "initiated"
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusProcessing TaskStatus = "processing"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusFailed     TaskStatus = "failed"
	TaskStatusCancelled  TaskStatus = "cancelled"
)

// TaskPlatform represents the platform for a task
type TaskPlatform string

const (
	PlatformGoogle TaskPlatform = "google"
	PlatformMeta   TaskPlatform = "meta"
)

// TaskNotification represents a notification for task status updates
type TaskNotification struct {
	ID        string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Task metadata
	TaskID   string       `gorm:"index" json:"task_id"`
	TaskType string       `json:"task_type"` // e.g. "google:ads_ingest", "meta:campaign_create"
	Platform TaskPlatform `json:"platform"`

	// User and context
	UserID string `gorm:"index" json:"user_id"`

	// Status and details
	Status   TaskStatus `gorm:"index" json:"status"`
	Message  string     `json:"message"`
	Details  string     `json:"details"` // JSON string with additional details
	ErrorMsg string     `json:"error_msg,omitempty"`

	// Progress tracking (0-100)
	Progress int `json:"progress"`

	// Metadata
	Metadata string `json:"metadata,omitempty"` // JSON string with additional metadata
}

// NotificationMessage is the WebSocket message sent to clients
type NotificationMessage struct {
	Type      string                 `json:"type"` // "notification"
	TaskID    string                 `json:"task_id"`
	TaskType  string                 `json:"task_type"`
	Platform  string                 `json:"platform"`
	UserID    string                 `json:"user_id"`
	Status    TaskStatus             `json:"status"`
	Message   string                 `json:"message"`
	Progress  int                    `json:"progress"`
	Timestamp time.Time              `json:"timestamp"`
	ErrorMsg  string                 `json:"error_msg,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
}
