package handlers

import (
	"encoding/json"
	"net/http"

	"Realify/database"
	"Realify/models"
	wsmanager "Realify/websocket"

	"github.com/gorilla/websocket"
)

// Global WebSocket manager instance
var WSManager *wsmanager.Manager

// InitWebSocket initializes the WebSocket manager
func InitWebSocket() {
	WSManager = wsmanager.NewManager()
	WSManager.Start()
}

// WebSocketUpgrade upgrades HTTP connection to WebSocket
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, implement proper CORS checks
		return true
	},
}

// HandleWebSocket handles WebSocket connections
// @Summary WebSocket Notifications
// @Description WebSocket endpoint for real-time task notifications
// @Tags notifications
// @Param user_id query string true "User ID"
// @Success 101 "Switching Protocols"
// @Failure 400 {object} map[string]string
// @Router /ws [get]
func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id query parameter is required", http.StatusBadRequest)
		return
	}

	// Verify user exists
	var user models.User
	if err := database.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Failed to upgrade connection", http.StatusInternalServerError)
		return
	}

	// Handle the connection
	WSManager.HandleConnection(conn, userID)
}

// GetNotificationHistory retrieves task notification history
// @Summary Get Notification History
// @Description Retrieve past task notifications for the authenticated user
// @Tags notifications
// @Param limit query int false "Limit number of notifications (default: 50)" default(50)
// @Produce json
// @Success 200 {array} models.TaskNotification
// @Failure 400 {object} map[string]string
// @Router /notifications/history [get]
func GetNotificationHistory(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get limit from query
	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if err := json.Unmarshal([]byte(limitStr), &limit); err == nil {
			if limit > 100 {
				limit = 100 // Max 100
			}
			if limit < 1 {
				limit = 1
			}
		}
	}

	// Fetch notifications from database
	var notifications []models.TaskNotification
	if err := database.DB.
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&notifications).Error; err != nil {
		http.Error(w, "Failed to fetch notifications", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notifications)
}

// GetNotificationByTaskID retrieves a specific task notification
// @Summary Get Notification by Task ID
// @Description Retrieve a specific task notification
// @Tags notifications
// @Param task_id query string true "Task ID"
// @Produce json
// @Success 200 {object} models.TaskNotification
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /notifications [get]
func GetNotificationByTaskID(w http.ResponseWriter, r *http.Request) {
	taskID := r.URL.Query().Get("task_id")
	if taskID == "" {
		http.Error(w, "task_id query parameter is required", http.StatusBadRequest)
		return
	}

	// Extract user ID from context
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Fetch notification from database
	var notification models.TaskNotification
	if err := database.DB.
		Where("task_id = ? AND user_id = ?", taskID, userID).
		First(&notification).Error; err != nil {
		http.Error(w, "Notification not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notification)
}

// BroadcastNotificationToUser sends a notification to a specific user
// This is a helper function for internal use
func BroadcastNotificationToUser(userID string, message interface{}) {
	if WSManager != nil {
		WSManager.BroadcastToUser(userID, message)
	}
}

// GetOrCreateDefaultUser returns the first user in the DB, creating a demo user if none exists.
// @Summary Get or Create Default User
// @Description Returns the first user record, or creates a demo user if the table is empty
// @Tags users
// @Produce json
// @Success 200 {object} models.User
// @Failure 500 {object} map[string]string
// @Router /users/default [get]
func GetOrCreateDefaultUser(w http.ResponseWriter, r *http.Request) {
	var user models.User
	if err := database.DB.First(&user).Error; err != nil {
		user = models.User{Name: "Demo User", Email: "demo@realify.local"}
		if err := database.DB.Create(&user).Error; err != nil {
			http.Error(w, "Failed to create user", http.StatusInternalServerError)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// GetWSStats returns WebSocket connection statistics
// @Summary Get WebSocket Stats
// @Description Get WebSocket connection statistics
// @Tags notifications
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /notifications/stats [get]
func GetWSStats(w http.ResponseWriter, r *http.Request) {
	stats := map[string]interface{}{
		"total_clients": 0,
	}

	if WSManager != nil {
		stats = WSManager.GetStats()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
