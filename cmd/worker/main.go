package main

import (
	"Realify/database"
	"Realify/models"
	"Realify/services"
	"Realify/workers"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/hibiken/asynq"
)

func main() {
	database.InitDB()

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

func handleGoogleAdsIngest(ctx context.Context, t *asynq.Task) error {
	var p workers.GoogleAdsIngestPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}
	fmt.Printf("Starting Google Ads Ingestion: %s\n", p.Type)

	// Step 1: Ingest
	if err := services.SyncGoogleCampaigns(); err != nil {
		return fmt.Errorf("google sync failed: %w", err)
	}

	// Step 2: Query from DB to verify
	var count int64
	database.DB.Model(&models.GoogleCampaign{}).Count(&count)
	fmt.Printf("[google:ads_ingest] verified — total campaigns in DB: %d\n", count)
	return nil
}

func handleMetaAdsIngest(ctx context.Context, t *asynq.Task) error {
	var p workers.MetaAdsIngestPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}
	fmt.Printf("Starting Meta Ads Ingestion: %s\n", p.Type)

	// Step 1: Ingest
	if err := services.SyncMetaCampaigns(p.AdAccountID, p.AccessToken); err != nil {
		return fmt.Errorf("meta sync failed: %w", err)
	}

	// Step 2: Query from DB to verify
	var count int64
	database.DB.Model(&models.MetaCampaignRecord{}).Where("ad_account_id = ?", p.AdAccountID).Count(&count)
	fmt.Printf("[meta:ads_ingest] verified — account=%s total campaigns in DB: %d\n", p.AdAccountID, count)
	return nil
}

func handleMetaCampaignCreate(ctx context.Context, t *asynq.Task) error {
	var p workers.MetaCampaignCreatePayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}

	// Step 1: ingest — call Meta API and save to DB
	result, err := services.CreateCampaign(p.AdAccountID, p.AccessToken, p.Req)
	if err != nil {
		return fmt.Errorf("meta campaign create failed: %w", err)
	}

	// Step 2: query from DB to confirm
	id, _ := result["id"].(string)
	var record models.MetaCampaignRecord
	if err := database.DB.First(&record, "id = ?", id).Error; err != nil {
		return fmt.Errorf("meta campaign ingested but DB query failed: %w", err)
	}
	fmt.Printf("[meta:campaign_create] verified — id=%s name=%s status=%s account=%s\n",
		record.ID, record.Name, record.Status, record.AdAccountID)
	return nil
}

func handleMetaAdSetCreate(ctx context.Context, t *asynq.Task) error {
	var p workers.MetaAdSetCreatePayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}

	// Step 1: ingest
	result, err := services.CreateAdSet(p.AdAccountID, p.AccessToken, p.PixelID, p.Req)
	if err != nil {
		return fmt.Errorf("meta adset create failed: %w", err)
	}

	// Step 2: query from DB to confirm
	id, _ := result["id"].(string)
	var record models.MetaAdSetRecord
	if err := database.DB.First(&record, "id = ?", id).Error; err != nil {
		return fmt.Errorf("meta adset ingested but DB query failed: %w", err)
	}
	fmt.Printf("[meta:adset_create] verified — id=%s name=%s status=%s campaign=%s\n",
		record.ID, record.Name, record.Status, record.CampaignID)
	return nil
}

func handleMetaAdCreate(ctx context.Context, t *asynq.Task) error {
	var p workers.MetaAdCreatePayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}

	// Step 1: ingest
	result, err := services.CreateAd(p.AdAccountID, p.AccessToken, p.PageID, p.Req)
	if err != nil {
		return fmt.Errorf("meta ad create failed: %w", err)
	}

	// Step 2: query from DB to confirm
	id, _ := result["id"].(string)
	var record models.MetaAdRecord
	if err := database.DB.First(&record, "id = ?", id).Error; err != nil {
		return fmt.Errorf("meta ad ingested but DB query failed: %w", err)
	}
	fmt.Printf("[meta:ad_create] verified — id=%s name=%s status=%s adset=%s\n",
		record.ID, record.Name, record.Status, record.AdSetID)
	return nil
}

func handleGoogleCampaignCreate(ctx context.Context, t *asynq.Task) error {
	var p workers.GoogleCampaignCreatePayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}

	// Step 1: ingest
	result, err := services.CreateGoogleCampaign(p.Req)
	if err != nil {
		return fmt.Errorf("google campaign create failed: %w", err)
	}

	// Step 2: query from DB to confirm
	resultMap, _ := result.(map[string]interface{})
	googleID, _ := resultMap["id"].(string)
	var record models.GoogleCampaign
	if err := database.DB.Where("google_id = ?", googleID).First(&record).Error; err != nil {
		return fmt.Errorf("google campaign ingested but DB query failed: %w", err)
	}
	fmt.Printf("[google:campaign_create] verified — google_id=%s name=%s status=%s resource=%s\n",
		record.GoogleID, record.Name, record.Status, record.ResourceName)
	return nil
}

func handleGoogleAdGroupCreate(ctx context.Context, t *asynq.Task) error {
	var p workers.GoogleAdGroupCreatePayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}

	// Step 1: ingest
	result, err := services.CreateGoogleAdGroup(p.Req)
	if err != nil {
		return fmt.Errorf("google adgroup create failed: %w", err)
	}

	// Step 2: query from DB to confirm
	adGroup, _ := result.(models.GoogleAdGroup)
	var record models.GoogleAdGroup
	if err := database.DB.Where("google_id = ?", adGroup.GoogleID).First(&record).Error; err != nil {
		return fmt.Errorf("google adgroup ingested but DB query failed: %w", err)
	}
	fmt.Printf("[google:adgroup_create] verified — google_id=%s name=%s status=%s campaign=%s\n",
		record.GoogleID, record.Name, record.Status, record.CampaignResourceName)
	return nil
}

func handleGoogleAdCreate(ctx context.Context, t *asynq.Task) error {
	var p workers.GoogleAdCreatePayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}

	// Step 1: ingest
	result, err := services.CreateGoogleAd(p.Req)
	if err != nil {
		return fmt.Errorf("google ad create failed: %w", err)
	}

	// Step 2: query from DB to confirm
	ad, _ := result.(models.GoogleAd)
	var record models.GoogleAd
	if err := database.DB.Where("google_id = ?", ad.GoogleID).First(&record).Error; err != nil {
		return fmt.Errorf("google ad ingested but DB query failed: %w", err)
	}
	fmt.Printf("[google:ad_create] verified — google_id=%s status=%s adgroup=%s url=%s\n",
		record.GoogleID, record.Status, record.AdGroupResourceName, record.FinalURL)
	return nil
}
