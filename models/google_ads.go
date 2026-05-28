package models

import "gorm.io/gorm"

// Google Ads Models (API Requests)

type GoogleCampaignRequest struct {
	Name              string `json:"name"`
	DailyBudgetMicros int64  `json:"daily_budget_micros"`
}

type GoogleAdGroupRequest struct {
	Name                 string `json:"name"`
	CampaignResourceName string `json:"campaign_resource_name"`
}

type GoogleAdRequest struct {
	Name                string `json:"name"`
	AdGroupResourceName string `json:"ad_group_resource_name"`
	FinalUrl            string `json:"final_url"`
	Headline            string `json:"headline"`
	Description         string `json:"description"`
}

type GoogleKeywordRequest struct {
	AdGroupResourceName string   `json:"ad_group_resource_name"`
	Keywords            []string `json:"keywords"`
	MatchType           string   `json:"match_type"` // EXACT, PHRASE, BROAD
	CpcBidMicros        int64    `json:"cpc_bid_micros"`
}

type GoogleKeywordDeleteRequest struct {
	ResourceName string `json:"resource_name"`
}

// --- DATABASE MODELS ---

type GoogleCampaign struct {
	gorm.Model
	GoogleID     string `gorm:"uniqueIndex" json:"google_id"`
	ResourceName string `gorm:"uniqueIndex" json:"resource_name"`
	Name         string `json:"name"`
	Status       string `json:"status"`
}

type GoogleAdGroup struct {
	gorm.Model
	GoogleID             string `gorm:"uniqueIndex" json:"google_id"`
	ResourceName         string `gorm:"uniqueIndex" json:"resource_name"`
	Name                 string `json:"name"`
	Status               string `json:"status"`
	CampaignResourceName string `json:"campaign_resource_name"`
}

type GoogleAd struct {
	gorm.Model
	GoogleID            string `gorm:"uniqueIndex" json:"google_id"`
	ResourceName        string `gorm:"uniqueIndex" json:"resource_name"`
	Status              string `json:"status"`
	AdGroupResourceName string `json:"ad_group_resource_name"`
	FinalURL            string `json:"final_url"`
}
