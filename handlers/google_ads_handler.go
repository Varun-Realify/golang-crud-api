package handlers

import (
	"Realify/config"
	"Realify/models"
	"Realify/services"
	"Realify/workers"
	"encoding/json"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

// GoogleLogin redirects to Google OAuth consent screen
// @Summary Google Ads OAuth Login
// @Description Redirects user to Google OAuth2 consent screen
// @Tags google_auth
// @Success 200 {object} map[string]string
// @Router /google/auth [get]
func GoogleLogin(w http.ResponseWriter, r *http.Request) {
	clientID := os.Getenv("GOOGLE_ADS_CLIENT_ID")
	redirectURI := os.Getenv("GOOGLE_ADS_REDIRECT_URL")
	scope := "https://www.googleapis.com/auth/adwords"
	authURL := "https://accounts.google.com/o/oauth2/v2/auth?client_id=" + clientID +
		"&redirect_uri=" + redirectURI +
		"&response_type=code&scope=" + scope +
		"&access_type=offline&prompt=consent"

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"login_url": authURL,
		"message":   "Copy and paste the login_url into your browser address bar to authenticate",
	})
}

// GoogleCallback handles Google OAuth callback
// @Summary Google Ads OAuth Callback
// @Description Receives the auth code from Google and exchanges it for tokens
// @Tags google_auth
// @Param code query string true "Auth Code from Google"
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /callback [get]
func GoogleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Code not found", http.StatusBadRequest)
		return
	}
	tokens, err := config.ExchangeGoogleCodeForToken(code)
	if err != nil {
		http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Auth successful",
		"tokens":  tokens,
	})
}

// GetGoogleCustomers lists accessible Google Ads accounts
// @Summary Get Google Ads Customers
// @Description Fetch a list of accessible Google Ads customer accounts
// @Tags google
// @Produce json
// @Success 200 {object} interface{}
// @Router /google/customers [get]
func GetGoogleCustomers(w http.ResponseWriter, r *http.Request) {
	data, err := services.GetGoogleCustomers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// --- CAMPAIGNS ---

// ListGoogleCampaigns lists all campaigns
// @Summary List Google Campaigns
// @Description Fetch all campaigns for the configured customer ID
// @Tags google_campaigns
// @Produce json
// @Success 200 {object} interface{}
// @Router /google/campaigns [get]
func ListGoogleCampaigns(w http.ResponseWriter, r *http.Request) {
	data, err := services.ListGoogleCampaigns()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// SyncGoogleCampaigns godoc
// @Summary Sync Google campaigns
// @Description Queues a background task to fetch all campaigns from Google and ingest them into the DB.
// @Tags google_campaigns
// @Produce json
// @Success 202 {object} map[string]string
// @Router /google/sync [post]
func SyncGoogleCampaigns(w http.ResponseWriter, r *http.Request) {
	if err := workers.EnqueueTask(workers.TypeGoogleAdsIngest, workers.GoogleAdsIngestPayload{
		Type: "campaign",
	}); err != nil {
		http.Error(w, "Failed to queue sync task: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "queued",
		"message": "Google campaigns sync queued",
	})
}

// CreateGoogleCampaign creates a new campaign
// @Summary Create Google Campaign
// @Description Queues Google Ads campaign creation as a background task. Returns 202 immediately.
// @Tags google_campaigns
// @Accept json
// @Produce json
// @Param campaign body models.GoogleCampaignRequest true "Campaign Details"
// @Success 202 {object} map[string]string
// @Router /google/campaigns [post]
func CreateGoogleCampaign(w http.ResponseWriter, r *http.Request) {
	var req models.GoogleCampaignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if err := workers.EnqueueTask(workers.TypeGoogleCampaignCreate, workers.GoogleCampaignCreatePayload{
		Req: req,
	}); err != nil {
		http.Error(w, "Failed to queue task: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "queued",
		"message": "Campaign creation queued for background processing",
	})
}

// DeleteGoogleCampaign deletes a campaign
// @Summary Delete Google Campaign
// @Description Remove a campaign by its resource name
// @Tags google_campaigns
// @Param resource_name query string true "Resource Name"
// @Produce json
// @Success 200 {object} interface{}
// @Router /google/campaigns [delete]
func DeleteGoogleCampaign(w http.ResponseWriter, r *http.Request) {
	resourceName := r.URL.Query().Get("resource_name")
	if resourceName == "" {
		http.Error(w, "Resource name is required", http.StatusBadRequest)
		return
	}
	data, err := services.DeleteGoogleCampaign(resourceName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// GetGoogleCampaignPerformanceInsights gets performance metrics
// @Summary Get Google Campaign Performance Insights
// @Description Fetch real-time performance metrics for a specific campaign ID
// @Tags google_campaigns
// @Produce json
// @Param id path string true "Campaign ID"
// @Success 200 {object} interface{}
// @Router /google/campaigns/{id}/insights [get]
func GetGoogleCampaignPerformanceInsights(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	data, err := services.GetGoogleCampaignPerformanceInsights(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// --- AD GROUPS ---

// ListGoogleAdGroups lists ad groups
// @Summary List Google Ad Groups
// @Description Fetch all ad groups for a specific campaign
// @Tags google_adgroups
// @Param campaign_resource_name query string true "Campaign Resource Name"
// @Produce json
// @Success 200 {object} interface{}
// @Router /google/adgroups [get]
func ListGoogleAdGroups(w http.ResponseWriter, r *http.Request) {
	campaignResourceName := r.URL.Query().Get("campaign_resource_name")
	data, err := services.ListGoogleAdGroups(campaignResourceName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// CreateGoogleAdGroup creates an ad group
// @Summary Create Google Ad Group
// @Description Queues Google Ads ad group creation as a background task. Returns 202 immediately.
// @Tags google_adgroups
// @Accept json
// @Produce json
// @Param adgroup body models.GoogleAdGroupRequest true "Ad Group Details"
// @Success 202 {object} map[string]string
// @Router /google/adgroups [post]
func CreateGoogleAdGroup(w http.ResponseWriter, r *http.Request) {
	var req models.GoogleAdGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if err := workers.EnqueueTask(workers.TypeGoogleAdGroupCreate, workers.GoogleAdGroupCreatePayload{
		Req: req,
	}); err != nil {
		http.Error(w, "Failed to queue task: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "queued",
		"message": "Ad group creation queued for background processing",
	})
}

// GetGoogleAdGroupPerformanceInsights gets performance metrics
// @Summary Get Google Ad Group Performance Insights
// @Description Fetch real-time performance metrics for a specific ad group ID
// @Tags google_adgroups
// @Produce json
// @Param id path string true "Ad Group ID"
// @Success 200 {object} interface{}
// @Router /google/adgroups/{id}/insights [get]
func GetGoogleAdGroupPerformanceInsights(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	data, err := services.GetGoogleAdGroupPerformanceInsights(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// --- ADS ---

// ListGoogleAds lists ads
// @Summary List Google Ads
// @Description Fetch all ads for a specific ad group
// @Tags google_ads
// @Param ad_group_resource_name query string true "Ad Group Resource Name"
// @Produce json
// @Success 200 {object} interface{}
// @Router /google/ads [get]
func ListGoogleAds(w http.ResponseWriter, r *http.Request) {
	adGroupResourceName := r.URL.Query().Get("ad_group_resource_name")
	data, err := services.ListGoogleAds(adGroupResourceName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// CreateGoogleAd creates an ad
// @Summary Create Google Ad
// @Description Queues Google responsive search ad creation as a background task. Returns 202 immediately.
// @Tags google_ads
// @Accept json
// @Produce json
// @Param ad body models.GoogleAdRequest true "Ad Details"
// @Success 202 {object} map[string]string
// @Router /google/ads [post]
func CreateGoogleAd(w http.ResponseWriter, r *http.Request) {
	var req models.GoogleAdRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if err := workers.EnqueueTask(workers.TypeGoogleAdCreate, workers.GoogleAdCreatePayload{
		Req: req,
	}); err != nil {
		http.Error(w, "Failed to queue task: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "queued",
		"message": "Ad creation queued for background processing",
	})
}

// --- KEYWORDS ---

// CreateGoogleKeywords adds keywords
// @Summary Create Google Keywords
// @Description Add keywords to an ad group
// @Tags google_keywords
// @Accept json
// @Produce json
// @Param keywords body models.GoogleKeywordRequest true "Keyword Details"
// @Success 200 {object} interface{}
// @Router /google/keywords [post]
func CreateGoogleKeywords(w http.ResponseWriter, r *http.Request) {
	var req models.GoogleKeywordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	data, err := services.CreateGoogleKeywords(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// ListGoogleKeywords lists keywords
// @Summary List Google Keywords
// @Description Fetch all keywords for an ad group
// @Tags google_keywords
// @Param ad_group_resource_name query string true "Ad Group Resource Name"
// @Produce json
// @Success 200 {object} interface{}
// @Router /google/keywords [get]
func ListGoogleKeywords(w http.ResponseWriter, r *http.Request) {
	adGroupResourceName := r.URL.Query().Get("ad_group_resource_name")
	data, err := services.ListGoogleKeywords(adGroupResourceName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// DeleteGoogleKeyword removes a keyword
// @Summary Delete Google Keyword
// @Description Remove a specific keyword by resource name
// @Tags google_keywords
// @Accept json
// @Produce json
// @Param keyword body models.GoogleKeywordDeleteRequest true "Delete Details"
// @Success 200 {object} interface{}
// @Router /google/keywords [delete]
func DeleteGoogleKeyword(w http.ResponseWriter, r *http.Request) {
	var req models.GoogleKeywordDeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	data, err := services.DeleteGoogleKeyword(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
