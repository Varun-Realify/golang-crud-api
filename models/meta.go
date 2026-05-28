package models

import "time"

// CampaignCreate represents the payload for creating a Meta campaign
type CampaignCreate struct {
	Name                string   `json:"name"`
	Objective           string   `json:"objective"`
	Status              string   `json:"status"`
	SpecialAdCategories []string `json:"special_ad_categories"`
}

// CampaignUpdate represents the payload for updating a Meta campaign
type CampaignUpdate struct {
	Name      string `json:"name,omitempty"`
	Status    string `json:"status,omitempty"`
	Objective string `json:"objective,omitempty"`
}

// AdSetCreate represents the payload for creating a Meta ad set
type AdSetCreate struct {
	Name             string                 `json:"name"`
	CampaignID       string                 `json:"campaign_id"`
	DailyBudget      int64                  `json:"daily_budget"` // In paise/cents
	BillingEvent     string                 `json:"billing_event"`
	OptimizationGoal string                 `json:"optimization_goal"`
	Targeting        map[string]interface{} `json:"targeting"`
	Status           string                 `json:"status"`
}

// AdCreate represents the payload for creating a Meta ad
type AdCreate struct {
	Name     string `json:"name"`
	AdSetID  string `json:"adset_id"`
	ImageURL string `json:"image_url"`
	BodyText string `json:"body_text"`
	Headline string `json:"headline"`
	LinkURL  string `json:"link_url"`
	Status   string `json:"status"`
}

// MetaAd represents an ad returned from Meta API
type MetaAd struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	AdSetID    string `json:"adset_id"`
	CampaignID string `json:"campaign_id"`
}

// MetaCampaign represents a campaign returned from Meta API
type MetaCampaign struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Status         string  `json:"status"`
	Objective      string  `json:"objective"`
	StartTime      string  `json:"start_time"`
	StopTime       string  `json:"stop_time"`
	DailyBudget    float64 `json:"daily_budget"`
	LifetimeBudget float64 `json:"lifetime_budget"`
	TotalBudget    float64 `json:"total_budget"`
}

// MetaInsights represents performance data from Meta API
type MetaInsights struct {
	Spend       float64 `json:"spend"`
	Impressions int     `json:"impressions"`
	Clicks      int     `json:"clicks"`
	Conversions int     `json:"conversions"`
	Revenue     float64 `json:"revenue"`
	CTR         float64 `json:"ctr"`
	CPM         float64 `json:"cpm"`
	Reach       int     `json:"reach"`
}

// MetaTimeSeries represents daily progress data
type MetaTimeSeries struct {
	Date        string  `json:"date"`
	Spend       float64 `json:"spend"`
	Impressions int     `json:"impressions"`
	Clicks      int     `json:"clicks"`
	Conversions int     `json:"conversions"`
}

// MetaTokenResponse represents the response for token exchange
type MetaTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

// MetaConnectRequest is the payload for onboarding a Meta user.
// The short-lived token is exchanged for a long-lived token and the user
// row is created (or updated) in the local database.
type MetaConnectRequest struct {
	ShortToken  string `json:"short_token"`
	Email       string `json:"email"`
	AdAccountID string `json:"ad_account_id"` // e.g. "act_123456789"
	PageID      string `json:"page_id"`
	PixelID     string `json:"pixel_id,omitempty"`
	CatalogID   string `json:"catalog_id,omitempty"`
}

// SyncResult represents the result of a catalog sync
type SyncResult struct {
	TotalSynced int                      `json:"total_synced"`
	Details     []map[string]interface{} `json:"details"`
}

// Date parameters for insights
type InsightsParams struct {
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
}

// --- DATABASE MODELS ---

type MetaCampaignRecord struct {
	ID             string  `gorm:"primaryKey" json:"id"`
	Name           string  `json:"name"`
	Status         string  `json:"status"`
	Objective      string  `json:"objective"`
	StartTime      string  `json:"start_time"`
	StopTime       string  `json:"stop_time"`
	DailyBudget    float64 `json:"daily_budget"`
	LifetimeBudget float64 `json:"lifetime_budget"`
	TotalBudget    float64 `json:"total_budget"`
	AdAccountID    string  `json:"ad_account_id"`
}

type MetaAdSetRecord struct {
	ID          string  `gorm:"primaryKey" json:"id"`
	Name        string  `json:"name"`
	CampaignID  string  `json:"campaign_id"`
	DailyBudget float64 `json:"daily_budget"`
	Status      string  `json:"status"`
	AdAccountID string  `json:"ad_account_id"`
}

type MetaAdRecord struct {
	ID          string `gorm:"primaryKey" json:"id"`
	Name        string `json:"name"`
	AdSetID     string `json:"adset_id"`
	Status      string `json:"status"`
	AdAccountID string `json:"ad_account_id"`
}
