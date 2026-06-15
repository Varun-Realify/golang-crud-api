package workers

import "Realify/models"

const (
	TypeGoogleAdsIngest = "google:ads_ingest"
	TypeMetaAdsIngest   = "meta:ads_ingest"

	TypeMetaCampaignCreate   = "meta:campaign_create"
	TypeMetaAdSetCreate      = "meta:adset_create"
	TypeMetaAdCreate         = "meta:ad_create"
	TypeGoogleCampaignCreate = "google:campaign_create"
	TypeGoogleAdGroupCreate  = "google:adgroup_create"
	TypeGoogleAdCreate       = "google:ad_create"
)

type GoogleAdsIngestPayload struct {
	UserID       string `json:"user_id"`
	ResourceName string `json:"resource_name"`
	Type         string `json:"type"` // e.g. campaign
}

type MetaAdsIngestPayload struct {
	UserID      string `json:"user_id"`
	AdAccountID string `json:"ad_account_id"`
	AccessToken string `json:"access_token"`
	Type        string `json:"type"` // e.g. campaign
}

type MetaCampaignCreatePayload struct {
	UserID      string                `json:"user_id"`
	AdAccountID string                `json:"ad_account_id"`
	AccessToken string                `json:"access_token"`
	Req         models.CampaignCreate `json:"req"`
}

type MetaAdSetCreatePayload struct {
	UserID      string             `json:"user_id"`
	AdAccountID string             `json:"ad_account_id"`
	AccessToken string             `json:"access_token"`
	PixelID     string             `json:"pixel_id"`
	Req         models.AdSetCreate `json:"req"`
}

type MetaAdCreatePayload struct {
	UserID      string          `json:"user_id"`
	AdAccountID string          `json:"ad_account_id"`
	AccessToken string          `json:"access_token"`
	PageID      string          `json:"page_id"`
	Req         models.AdCreate `json:"req"`
}

type GoogleCampaignCreatePayload struct {
	UserID string                       `json:"user_id"`
	Req    models.GoogleCampaignRequest `json:"req"`
}

type GoogleAdGroupCreatePayload struct {
	UserID string                      `json:"user_id"`
	Req    models.GoogleAdGroupRequest `json:"req"`
}

type GoogleAdCreatePayload struct {
	UserID string                 `json:"user_id"`
	Req    models.GoogleAdRequest `json:"req"`
}
