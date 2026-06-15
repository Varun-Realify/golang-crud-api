package websocket

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"Realify/models"

	"github.com/redis/go-redis/v9"
)

// NotificationPublisher publishes task notifications via Redis
type NotificationPublisher struct {
	client *redis.Client
}

// NewNotificationPublisher creates a new notification publisher
func NewNotificationPublisher() *NotificationPublisher {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "127.0.0.1:6379"
	}

	client := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	return &NotificationPublisher{
		client: client,
	}
}

// PublishNotification publishes a notification to Redis
func (np *NotificationPublisher) PublishNotification(notification models.NotificationMessage) error {
	data, err := json.Marshal(notification)
	if err != nil {
		return err
	}

	ctx := context.Background()
	channel := "notifications:" + notification.Platform

	// Publish to platform channel
	if err := np.client.Publish(ctx, channel, data).Err(); err != nil {
		log.Printf("Failed to publish to %s: %v", channel, err)
	}

	// Also publish to user-specific channel
	userChannel := "notifications:user:" + notification.TaskID
	if err := np.client.Publish(ctx, userChannel, data).Err(); err != nil {
		log.Printf("Failed to publish to %s: %v", userChannel, err)
	}

	return nil
}

// Close closes the Redis connection
func (np *NotificationPublisher) Close() error {
	return np.client.Close()
}
