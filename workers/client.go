package workers

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/hibiken/asynq"
)

var Client *asynq.Client

func InitClient() {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "127.0.0.1:6379"
	}

	Client = asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr})
	fmt.Println("Asynq Client connected to Redis at", redisAddr)
}

func EnqueueTask(taskType string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	task := asynq.NewTask(taskType, data)
	info, err := Client.Enqueue(task)
	if err != nil {
		return err
	}

	fmt.Printf("Enqueued task: id=%s queue=%s\n", info.ID, info.Queue)
	return nil
}

func CloseClient() {
	if Client != nil {
		Client.Close()
	}
}
