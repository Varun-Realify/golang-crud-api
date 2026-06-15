package main

import (
	"Realify/database"
	"Realify/models"

	"Realify/services"
	"Realify/websocket"
	"Realify/workers"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

// Global publisher for notifications
var publisher *websocket.NotificationPublisher

func main() {
	database.InitDB()

	// Initialize notification publisher
	publisher = websocket.NewNotificationPublisher()
	defer publisher.Close()

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "127.0.0.1:6379"
	}

	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr},
		asynq.Config{Concurrency: 10},
	)

	mux := asynq.NewServeMux()
	mux.HandleFunc(workers.TypeGoogleAdsIngest, handleGoogleAdsIngest)
	mux.HandleFunc(workers.TypeMetaAdsIngest, handleMetaAdsIngest)
	mux.HandleFunc(workers.TypeMetaCampaignCreate, handleMetaCampaignCreate)
	mux.HandleFunc(workers.TypeMetaAdSetCreate, handleMetaAdSetCreate)
	mux.HandleFunc(workers.TypeMetaAdCreate, handleMetaAdCreate)
	mux.HandleFunc(workers.TypeGoogleCampaignCreate, handleGoogleCampaignCreate)
	mux.HandleFunc(workers.TypeGoogleAdGroupCreate, handleGoogleAdGroupCreate)
	mux.HandleFunc(workers.TypeGoogleAdCreate, handleGoogleAdCreate)

	fmt.Println("Worker starting on Redis", redisAddr)
	if err := srv.Run(mux); err != nil {
		log.Fatal(err)
	}
}

func broadcastNotification(
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
	if publisher != nil {
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
		if err := publisher.PublishNotification(notification); err != nil {
			log.Printf("Failed to publish notification: %v", err)
		}

		// Save to database
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

		if len(details) > 0 {
			if detailsJSON, err := json.Marshal(details); err == nil {
				taskNotification.Details = string(detailsJSON)
			}
		}

		if err := database.DB.Create(&taskNotification).Error; err != nil {
			log.Printf("Failed to save notification to database: %v", err)
		}
	}
}

func handleGoogleAdsIngest(ctx context.Context, t *asynq.Task) error {
	var p workers.GoogleAdsIngestPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}

	taskID := uuid.New().String()

	// Broadcast initiated notification
	broadcastNotification(
		taskID,
		workers.TypeGoogleAdsIngest,
		models.PlatformGoogle,
		p.UserID,
		models.TaskStatusInitiated,
		"Google Ads ingestion initiated",
		0,
		nil,
		"",
	)

	fmt.Printf("Starting Google Ads Ingestion: %s\n", p.Type)

	// Broadcast processing notification
	broadcastNotification(
		taskID,
		workers.TypeGoogleAdsIngest,
		models.PlatformGoogle,
		p.UserID,
		models.TaskStatusProcessing,
		"Syncing Google campaigns...",
		25,
		nil,
		"",
	)

	// Step 1: Ingest
	if err := services.SyncGoogleCampaigns(); err != nil {
		broadcastNotification(
			taskID,
			workers.TypeGoogleAdsIngest,
			models.PlatformGoogle,
			p.UserID,
			models.TaskStatusFailed,
			"Google Ads ingestion failed",
			0,
			nil,
			err.Error(),
		)
		return fmt.Errorf("google sync failed: %w", err)
	}

	// Step 2: Query from DB to verify
	var count int64
	database.DB.Model(&models.GoogleCampaign{}).Count(&count)
	fmt.Printf("[google:ads_ingest] verified — total campaigns in DB: %d\n", count)

	// Broadcast completed notification
	broadcastNotification(
		taskID,
		workers.TypeGoogleAdsIngest,
		models.PlatformGoogle,
		p.UserID,
		models.TaskStatusCompleted,
		fmt.Sprintf("Successfully synced %d Google campaigns", count),
		100,
		map[string]interface{}{
			"total_campaigns": count,
		},
		"",
	)

	return nil
}

func handleMetaAdsIngest(ctx context.Context, t *asynq.Task) error {
	var p workers.MetaAdsIngestPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}

	taskID := uuid.New().String()

	// Broadcast initiated notification
	broadcastNotification(
		taskID,
		workers.TypeMetaAdsIngest,
		models.PlatformMeta,
		p.UserID,
		models.TaskStatusInitiated,
		"Meta Ads ingestion initiated",
		0,
		nil,
		"",
	)

	fmt.Printf("Starting Meta Ads Ingestion: %s\n", p.Type)

	// Broadcast processing notification
	broadcastNotification(
		taskID,
		workers.TypeMetaAdsIngest,
		models.PlatformMeta,
		p.UserID,
		models.TaskStatusProcessing,
		fmt.Sprintf("Syncing Meta campaigns for account %s...", p.AdAccountID),
		25,
		nil,
		"",
	)

	// Step 1: Ingest
	if err := services.SyncMetaCampaigns(p.AdAccountID, p.AccessToken); err != nil {
		broadcastNotification(
			taskID,
			workers.TypeMetaAdsIngest,
			models.PlatformMeta,
			p.UserID,
			models.TaskStatusFailed,
			"Meta Ads ingestion failed",
			0,
			nil,
			err.Error(),
		)
		return fmt.Errorf("meta sync failed: %w", err)
	}

	// Step 2: Query from DB to verify
	var count int64
	database.DB.Model(&models.MetaCampaignRecord{}).Where("ad_account_id = ?", p.AdAccountID).Count(&count)
	fmt.Printf("[meta:ads_ingest] verified — account=%s total campaigns in DB: %d\n", p.AdAccountID, count)

	// Broadcast completed notification
	broadcastNotification(
		taskID,
		workers.TypeMetaAdsIngest,
		models.PlatformMeta,
		p.UserID,
		models.TaskStatusCompleted,
		fmt.Sprintf("Successfully synced %d Meta campaigns", count),
		100,
		map[string]interface{}{
			"ad_account_id":   p.AdAccountID,
			"total_campaigns": count,
		},
		"",
	)

	return nil
}

func handleMetaCampaignCreate(ctx context.Context, t *asynq.Task) error {
	var p workers.MetaCampaignCreatePayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}

	taskID := uuid.New().String()

	// Broadcast initiated notification
	broadcastNotification(
		taskID,
		workers.TypeMetaCampaignCreate,
		models.PlatformMeta,
		p.UserID,
		models.TaskStatusInitiated,
		fmt.Sprintf("Creating Meta campaign: %s", p.Req.Name),
		0,
		nil,
		"",
	)

	// Broadcast processing notification
	broadcastNotification(
		taskID,
		workers.TypeMetaCampaignCreate,
		models.PlatformMeta,
		p.UserID,
		models.TaskStatusProcessing,
		"Sending campaign creation request to Meta API...",
		50,
		nil,
		"",
	)

	// Step 1: ingest — call Meta API and save to DB
	result, err := services.CreateCampaign(p.AdAccountID, p.AccessToken, p.Req)
	if err != nil {
		broadcastNotification(
			taskID,
			workers.TypeMetaCampaignCreate,
			models.PlatformMeta,
			p.UserID,
			models.TaskStatusFailed,
			fmt.Sprintf("Failed to create campaign: %s", p.Req.Name),
			0,
			nil,
			err.Error(),
		)
		return fmt.Errorf("meta campaign create failed: %w", err)
	}

	// Step 2: query from DB to confirm
	id, _ := result["id"].(string)
	var record models.MetaCampaignRecord
	if err := database.DB.First(&record, "id = ?", id).Error; err != nil {
		broadcastNotification(
			taskID,
			workers.TypeMetaCampaignCreate,
			models.PlatformMeta,
			p.UserID,
			models.TaskStatusFailed,
			"Campaign created but database verification failed",
			0,
			nil,
			err.Error(),
		)
		return fmt.Errorf("meta campaign ingested but DB query failed: %w", err)
	}

	fmt.Printf("[meta:campaign_create] verified — id=%s name=%s status=%s account=%s\n",
		record.ID, record.Name, record.Status, record.AdAccountID)

	// Broadcast completed notification
	broadcastNotification(
		taskID,
		workers.TypeMetaCampaignCreate,
		models.PlatformMeta,
		p.UserID,
		models.TaskStatusCompleted,
		fmt.Sprintf("Successfully created campaign: %s", record.Name),
		100,
		map[string]interface{}{
			"campaign_id":     record.ID,
			"campaign_name":   record.Name,
			"campaign_status": record.Status,
		},
		"",
	)

	return nil
}

func handleMetaAdSetCreate(ctx context.Context, t *asynq.Task) error {
	var p workers.MetaAdSetCreatePayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}

	taskID := uuid.New().String()

	// Broadcast initiated notification
	broadcastNotification(
		taskID,
		workers.TypeMetaAdSetCreate,
		models.PlatformMeta,
		p.UserID,
		models.TaskStatusInitiated,
		fmt.Sprintf("Creating Meta ad set: %s", p.Req.Name),
		0,
		nil,
		"",
	)

	// Broadcast processing notification
	broadcastNotification(
		taskID,
		workers.TypeMetaAdSetCreate,
		models.PlatformMeta,
		p.UserID,
		models.TaskStatusProcessing,
		"Sending ad set creation request to Meta API...",
		50,
		nil,
		"",
	)

	// Step 1: ingest
	result, err := services.CreateAdSet(p.AdAccountID, p.AccessToken, p.PixelID, p.Req)
	if err != nil {
		broadcastNotification(
			taskID,
			workers.TypeMetaAdSetCreate,
			models.PlatformMeta,
			p.UserID,
			models.TaskStatusFailed,
			fmt.Sprintf("Failed to create ad set: %s", p.Req.Name),
			0,
			nil,
			err.Error(),
		)
		return fmt.Errorf("meta adset create failed: %w", err)
	}

	// Step 2: query from DB to confirm
	id, _ := result["id"].(string)
	var record models.MetaAdSetRecord
	if err := database.DB.First(&record, "id = ?", id).Error; err != nil {
		broadcastNotification(
			taskID,
			workers.TypeMetaAdSetCreate,
			models.PlatformMeta,
			p.UserID,
			models.TaskStatusFailed,
			"Ad set created but database verification failed",
			0,
			nil,
			err.Error(),
		)
		return fmt.Errorf("meta adset ingested but DB query failed: %w", err)
	}

	fmt.Printf("[meta:adset_create] verified — id=%s name=%s status=%s campaign=%s\n",
		record.ID, record.Name, record.Status, record.CampaignID)

	// Broadcast completed notification
	broadcastNotification(
		taskID,
		workers.TypeMetaAdSetCreate,
		models.PlatformMeta,
		p.UserID,
		models.TaskStatusCompleted,
		fmt.Sprintf("Successfully created ad set: %s", record.Name),
		100,
		map[string]interface{}{
			"adset_id":     record.ID,
			"adset_name":   record.Name,
			"campaign_id":  record.CampaignID,
			"adset_status": record.Status,
		},
		"",
	)

	return nil
}

func handleMetaAdCreate(ctx context.Context, t *asynq.Task) error {
	var p workers.MetaAdCreatePayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}

	taskID := uuid.New().String()

	// Broadcast initiated notification
	broadcastNotification(
		taskID,
		workers.TypeMetaAdCreate,
		models.PlatformMeta,
		p.UserID,
		models.TaskStatusInitiated,
		fmt.Sprintf("Creating Meta ad: %s", p.Req.Name),
		0,
		nil,
		"",
	)

	// Broadcast processing notification
	broadcastNotification(
		taskID,
		workers.TypeMetaAdCreate,
		models.PlatformMeta,
		p.UserID,
		models.TaskStatusProcessing,
		"Sending ad creation request to Meta API...",
		50,
		nil,
		"",
	)

	// Step 1: ingest
	result, err := services.CreateAd(p.AdAccountID, p.AccessToken, p.PageID, p.Req)
	if err != nil {
		broadcastNotification(
			taskID,
			workers.TypeMetaAdCreate,
			models.PlatformMeta,
			p.UserID,
			models.TaskStatusFailed,
			fmt.Sprintf("Failed to create ad: %s", p.Req.Name),
			0,
			nil,
			err.Error(),
		)
		return fmt.Errorf("meta ad create failed: %w", err)
	}

	// Step 2: query from DB to confirm
	id, _ := result["id"].(string)
	var record models.MetaAdRecord
	if err := database.DB.First(&record, "id = ?", id).Error; err != nil {
		broadcastNotification(
			taskID,
			workers.TypeMetaAdCreate,
			models.PlatformMeta,
			p.UserID,
			models.TaskStatusFailed,
			"Ad created but database verification failed",
			0,
			nil,
			err.Error(),
		)
		return fmt.Errorf("meta ad ingested but DB query failed: %w", err)
	}

	fmt.Printf("[meta:ad_create] verified — id=%s name=%s status=%s adset=%s\n",
		record.ID, record.Name, record.Status, record.AdSetID)

	// Broadcast completed notification
	broadcastNotification(
		taskID,
		workers.TypeMetaAdCreate,
		models.PlatformMeta,
		p.UserID,
		models.TaskStatusCompleted,
		fmt.Sprintf("Successfully created ad: %s", record.Name),
		100,
		map[string]interface{}{
			"ad_id":     record.ID,
			"ad_name":   record.Name,
			"adset_id":  record.AdSetID,
			"ad_status": record.Status,
		},
		"",
	)

	return nil
}

func handleGoogleCampaignCreate(ctx context.Context, t *asynq.Task) error {
	var p workers.GoogleCampaignCreatePayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}

	taskID := uuid.New().String()

	// Broadcast initiated notification
	broadcastNotification(
		taskID,
		workers.TypeGoogleCampaignCreate,
		models.PlatformGoogle,
		p.UserID,
		models.TaskStatusInitiated,
		fmt.Sprintf("Creating Google campaign: %s", p.Req.Name),
		0,
		nil,
		"",
	)

	// Broadcast processing notification
	broadcastNotification(
		taskID,
		workers.TypeGoogleCampaignCreate,
		models.PlatformGoogle,
		p.UserID,
		models.TaskStatusProcessing,
		"Sending campaign creation request to Google Ads API...",
		50,
		nil,
		"",
	)

	// Step 1: ingest
	result, err := services.CreateGoogleCampaign(p.Req)
	if err != nil {
		broadcastNotification(
			taskID,
			workers.TypeGoogleCampaignCreate,
			models.PlatformGoogle,
			p.UserID,
			models.TaskStatusFailed,
			fmt.Sprintf("Failed to create campaign: %s", p.Req.Name),
			0,
			nil,
			err.Error(),
		)
		return fmt.Errorf("google campaign create failed: %w", err)
	}

	// Step 2: query from DB to confirm
	resultMap, _ := result.(map[string]interface{})
	googleID, _ := resultMap["id"].(string)
	var record models.GoogleCampaign
	if err := database.DB.Where("google_id = ?", googleID).First(&record).Error; err != nil {
		broadcastNotification(
			taskID,
			workers.TypeGoogleCampaignCreate,
			models.PlatformGoogle,
			p.UserID,
			models.TaskStatusFailed,
			"Campaign created but database verification failed",
			0,
			nil,
			err.Error(),
		)
		return fmt.Errorf("google campaign ingested but DB query failed: %w", err)
	}

	fmt.Printf("[google:campaign_create] verified — google_id=%s name=%s status=%s resource=%s\n",
		record.GoogleID, record.Name, record.Status, record.ResourceName)

	// Broadcast completed notification
	broadcastNotification(
		taskID,
		workers.TypeGoogleCampaignCreate,
		models.PlatformGoogle,
		p.UserID,
		models.TaskStatusCompleted,
		fmt.Sprintf("Successfully created campaign: %s", record.Name),
		100,
		map[string]interface{}{
			"campaign_id":     record.GoogleID,
			"campaign_name":   record.Name,
			"campaign_status": record.Status,
			"resource_name":   record.ResourceName,
		},
		"",
	)

	return nil
}

func handleGoogleAdGroupCreate(ctx context.Context, t *asynq.Task) error {
	var p workers.GoogleAdGroupCreatePayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}

	taskID := uuid.New().String()

	// Broadcast initiated notification
	broadcastNotification(
		taskID,
		workers.TypeGoogleAdGroupCreate,
		models.PlatformGoogle,
		p.UserID,
		models.TaskStatusInitiated,
		fmt.Sprintf("Creating Google ad group: %s", p.Req.Name),
		0,
		nil,
		"",
	)

	// Broadcast processing notification
	broadcastNotification(
		taskID,
		workers.TypeGoogleAdGroupCreate,
		models.PlatformGoogle,
		p.UserID,
		models.TaskStatusProcessing,
		"Sending ad group creation request to Google Ads API...",
		50,
		nil,
		"",
	)

	// Step 1: ingest
	result, err := services.CreateGoogleAdGroup(p.Req)
	if err != nil {
		broadcastNotification(
			taskID,
			workers.TypeGoogleAdGroupCreate,
			models.PlatformGoogle,
			p.UserID,
			models.TaskStatusFailed,
			fmt.Sprintf("Failed to create ad group: %s", p.Req.Name),
			0,
			nil,
			err.Error(),
		)
		return fmt.Errorf("google adgroup create failed: %w", err)
	}

	// Step 2: query from DB to confirm
	adGroup, _ := result.(models.GoogleAdGroup)
	var record models.GoogleAdGroup
	if err := database.DB.Where("google_id = ?", adGroup.GoogleID).First(&record).Error; err != nil {
		broadcastNotification(
			taskID,
			workers.TypeGoogleAdGroupCreate,
			models.PlatformGoogle,
			p.UserID,
			models.TaskStatusFailed,
			"Ad group created but database verification failed",
			0,
			nil,
			err.Error(),
		)
		return fmt.Errorf("google adgroup ingested but DB query failed: %w", err)
	}

	fmt.Printf("[google:adgroup_create] verified — google_id=%s name=%s status=%s campaign=%s\n",
		record.GoogleID, record.Name, record.Status, record.CampaignResourceName)

	// Broadcast completed notification
	broadcastNotification(
		taskID,
		workers.TypeGoogleAdGroupCreate,
		models.PlatformGoogle,
		p.UserID,
		models.TaskStatusCompleted,
		fmt.Sprintf("Successfully created ad group: %s", record.Name),
		100,
		map[string]interface{}{
			"adgroup_id":        record.GoogleID,
			"adgroup_name":      record.Name,
			"adgroup_status":    record.Status,
			"campaign_resource": record.CampaignResourceName,
		},
		"",
	)

	return nil
}

func handleGoogleAdCreate(ctx context.Context, t *asynq.Task) error {
	var p workers.GoogleAdCreatePayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}

	taskID := uuid.New().String()

	// Broadcast initiated notification
	broadcastNotification(
		taskID,
		workers.TypeGoogleAdCreate,
		models.PlatformGoogle,
		p.UserID,
		models.TaskStatusInitiated,
		fmt.Sprintf("Creating Google ad: %s", p.Req.Headline),
		0,
		nil,
		"",
	)

	// Broadcast processing notification
	broadcastNotification(
		taskID,
		workers.TypeGoogleAdCreate,
		models.PlatformGoogle,
		p.UserID,
		models.TaskStatusProcessing,
		"Sending ad creation request to Google Ads API...",
		50,
		nil,
		"",
	)

	// Step 1: ingest
	result, err := services.CreateGoogleAd(p.Req)
	if err != nil {
		broadcastNotification(
			taskID,
			workers.TypeGoogleAdCreate,
			models.PlatformGoogle,
			p.UserID,
			models.TaskStatusFailed,
			"Failed to create ad",
			0,
			nil,
			err.Error(),
		)
		return fmt.Errorf("google ad create failed: %w", err)
	}

	// Step 2: query from DB to confirm
	ad, _ := result.(models.GoogleAd)
	var record models.GoogleAd
	if err := database.DB.Where("google_id = ?", ad.GoogleID).First(&record).Error; err != nil {
		broadcastNotification(
			taskID,
			workers.TypeGoogleAdCreate,
			models.PlatformGoogle,
			p.UserID,
			models.TaskStatusFailed,
			"Ad created but database verification failed",
			0,
			nil,
			err.Error(),
		)
		return fmt.Errorf("google ad ingested but DB query failed: %w", err)
	}

	fmt.Printf("[google:ad_create] verified — google_id=%s status=%s adgroup=%s url=%s\n",
		record.GoogleID, record.Status, record.AdGroupResourceName, record.FinalURL)

	// Broadcast completed notification
	broadcastNotification(
		taskID,
		workers.TypeGoogleAdCreate,
		models.PlatformGoogle,
		p.UserID,
		models.TaskStatusCompleted,
		fmt.Sprintf("Successfully created ad: %s", record.GoogleID),
		100,
		map[string]interface{}{
			"ad_id":            record.GoogleID,
			"ad_status":        record.Status,
			"adgroup_resource": record.AdGroupResourceName,
			"final_url":        record.FinalURL,
		},
		"",
	)

	return nil
}
