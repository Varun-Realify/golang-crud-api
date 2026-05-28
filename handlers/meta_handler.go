package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"Realify/models"
	"Realify/services"
	"Realify/workers"

	"github.com/gorilla/mux"
)

// GetMetaStatus godoc
// @Summary Check Meta API connection
// @Description Check connection and list available pages/catalogs. Pass X-User-Email header to use a specific user's credentials; omits header to fall back to .env defaults.
// @Tags meta
// @Produce json
// @Param X-User-Email header string false "User email (multi-user mode)"
// @Success 200 {object} map[string]interface{}
// @Router /meta/status [get]
func GetMetaStatus(w http.ResponseWriter, r *http.Request) {
	user := GetUserContext(r)
	status, err := services.TestMetaConnection(user.MetaAccessToken)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch pages and catalogs to help the user configure the app
	pages, _ := services.GetAvailablePages(user.MetaAccessToken)
	catalogs, _ := services.GetAvailableCatalogs(user.MetaAccessToken)

	// Clean up the output to match the requested format
	cleanPages := make([]map[string]interface{}, 0)
	for _, p := range pages {
		cleanPages = append(cleanPages, map[string]interface{}{
			"id":   p["id"],
			"name": p["name"],
		})
	}

	cleanCatalogs := make([]map[string]interface{}, 0)
	for _, c := range catalogs {
		cleanCatalogs = append(cleanCatalogs, map[string]interface{}{
			"id":   c["id"],
			"name": c["name"],
		})
	}

	response := map[string]interface{}{
		"connected":          status["connected"],
		"user":               status["user"],
		"ad_account":         user.MetaAdAccountID,
		"available_pages":    cleanPages,
		"available_catalogs": cleanCatalogs,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetAllCampaigns godoc
// @Summary Get all Meta campaigns
// @Description Fetch all ad campaigns from the local database (results are synced from Meta). Pass X-User-Email header to query a specific user's campaigns.
// @Tags meta
// @Produce json
// @Param X-User-Email header string false "User email (multi-user mode)"
// @Success 200 {object} map[string]interface{}
// @Router /meta/campaigns [get]
func GetAllCampaigns(w http.ResponseWriter, r *http.Request) {
	user := GetUserContext(r)
	campaigns, err := services.GetAllCampaigns(user.MetaAdAccountID, user.MetaAccessToken)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"count":     len(campaigns),
		"campaigns": campaigns,
	})
}

// SyncCampaigns godoc
// @Summary Sync Meta campaigns
// @Description Queues a background task to fetch all campaigns from Meta and ingest them into the DB.
// @Tags meta
// @Produce json
// @Param X-User-Email header string false "User email (multi-user mode)"
// @Success 202 {object} map[string]string
// @Router /meta/sync [post]
func SyncCampaigns(w http.ResponseWriter, r *http.Request) {
	user := GetUserContext(r)
	if err := workers.EnqueueTask(workers.TypeMetaAdsIngest, workers.MetaAdsIngestPayload{
		AdAccountID: user.MetaAdAccountID,
		AccessToken: user.MetaAccessToken,
		Type:        "campaign",
	}); err != nil {
		http.Error(w, "Failed to queue sync task: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "queued",
		"message": "Meta campaigns sync queued",
	})
}

// CreateCampaign godoc
// @Summary Create a new Meta campaign
// @Description Queues campaign creation via the Meta API and DB ingestion as a background task. Returns 202 immediately.
// @Tags meta
// @Accept json
// @Produce json
// @Param X-User-Email header string false "User email (multi-user mode)"
// @Param campaign body models.CampaignCreate true "Create campaign"
// @Success 202 {object} map[string]string
// @Router /meta/campaigns [post]
func CreateCampaign(w http.ResponseWriter, r *http.Request) {
	var req models.CampaignCreate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	user := GetUserContext(r)
	if err := workers.EnqueueTask(workers.TypeMetaCampaignCreate, workers.MetaCampaignCreatePayload{
		AdAccountID: user.MetaAdAccountID,
		AccessToken: user.MetaAccessToken,
		Req:         req,
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

// CreateAdSet godoc
// @Summary Create a new ad set
// @Description Queues ad set creation via the Meta API and DB ingestion as a background task. Returns 202 immediately.
// @Tags meta
// @Accept json
// @Produce json
// @Param X-User-Email header string false "User email (multi-user mode)"
// @Param adset body models.AdSetCreate true "Create ad set"
// @Success 202 {object} map[string]string
// @Router /meta/adsets [post]
func CreateAdSet(w http.ResponseWriter, r *http.Request) {
	var req models.AdSetCreate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	user := GetUserContext(r)
	if err := workers.EnqueueTask(workers.TypeMetaAdSetCreate, workers.MetaAdSetCreatePayload{
		AdAccountID: user.MetaAdAccountID,
		AccessToken: user.MetaAccessToken,
		PixelID:     user.MetaPixelID,
		Req:         req,
	}); err != nil {
		http.Error(w, "Failed to queue task: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "queued",
		"message": "Ad set creation queued for background processing",
	})
}

// CreateAd godoc
// @Summary Create a new ad
// @Description Queues ad creation (with creative) via the Meta API and DB ingestion as a background task. Returns 202 immediately. Requires the user's Meta Page ID.
// @Tags meta
// @Accept json
// @Produce json
// @Param X-User-Email header string false "User email (multi-user mode)"
// @Param ad body models.AdCreate true "Create ad"
// @Success 202 {object} map[string]string
// @Router /meta/ads [post]
func CreateAd(w http.ResponseWriter, r *http.Request) {
	var req models.AdCreate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	user := GetUserContext(r)
	if user.MetaPageID == "" {
		http.Error(w, "Meta Page ID is required for this user", http.StatusBadRequest)
		return
	}

	if err := workers.EnqueueTask(workers.TypeMetaAdCreate, workers.MetaAdCreatePayload{
		AdAccountID: user.MetaAdAccountID,
		AccessToken: user.MetaAccessToken,
		PageID:      user.MetaPageID,
		Req:         req,
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

// GetCampaignInsights godoc
// @Summary Get campaign insights
// @Description Fetches live performance metrics for a campaign directly from the Meta API (not local DB). Defaults to last 30 days if dates are omitted.
// @Tags meta
// @Produce json
// @Param X-User-Email header string false "User email (multi-user mode)"
// @Param id path string true "Campaign ID"
// @Param start_date query string false "Start Date (YYYY-MM-DD)"
// @Param end_date query string false "End Date (YYYY-MM-DD)"
// @Success 200 {object} map[string]interface{}
// @Router /meta/campaigns/{id}/insights [get]
func GetCampaignInsights(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	campaignID := vars["id"]

	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	if startDate == "" {
		startDate = time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	}
	if endDate == "" {
		endDate = time.Now().Format("2006-01-02")
	}

	user := GetUserContext(r)
	insights, err := services.GetCampaignInsights(campaignID, user.MetaAccessToken, startDate, endDate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"campaign_id": campaignID,
		"period":      startDate + " -> " + endDate,
		"insights":    insights,
	})
}

// UpdateCampaign godoc
// @Summary Update a Meta campaign
// @Description Updates a campaign's status or name via the Meta API. Pass X-User-Email header to act as a specific user.
// @Tags meta
// @Accept json
// @Produce json
// @Param X-User-Email header string false "User email (multi-user mode)"
// @Param id path string true "Campaign ID"
// @Param campaign body models.CampaignUpdate true "Update campaign"
// @Success 200 {object} map[string]interface{}
// @Router /meta/campaigns/{id} [patch]
func UpdateCampaign(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	campaignID := vars["id"]

	var req models.CampaignUpdate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	user := GetUserContext(r)
	result, err := services.UpdateCampaign(campaignID, user.MetaAccessToken, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GetAdSetAds godoc
// @Summary Get ads in an ad set
// @Description Fetch all ads belonging to a specific ad set via the Meta API. Pass X-User-Email header to act as a specific user.
// @Tags meta
// @Produce json
// @Param X-User-Email header string false "User email (multi-user mode)"
// @Param id path string true "Ad Set ID"
// @Success 200 {object} map[string]interface{}
// @Router /meta/adsets/{id}/ads [get]
func GetAdSetAds(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	adSetID := vars["id"]

	user := GetUserContext(r)
	ads, err := services.GetAdSetAds(adSetID, user.MetaAccessToken)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"count": len(ads),
		"ads":   ads,
	})
}

// GetAdInsights godoc
// @Summary Get ad insights
// @Description Fetches live performance metrics for an ad directly from the Meta API (not local DB). Defaults to last 30 days if dates are omitted.
// @Tags meta
// @Produce json
// @Param X-User-Email header string false "User email (multi-user mode)"
// @Param id path string true "Ad ID"
// @Param start_date query string false "Start Date (YYYY-MM-DD)"
// @Param end_date query string false "End Date (YYYY-MM-DD)"
// @Success 200 {object} map[string]interface{}
// @Router /meta/ads/{id}/insights [get]
func GetAdInsights(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	adID := vars["id"]

	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	if startDate == "" {
		startDate = time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	}
	if endDate == "" {
		endDate = time.Now().Format("2006-01-02")
	}

	user := GetUserContext(r)
	insights, err := services.GetAdInsights(adID, user.MetaAccessToken, startDate, endDate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ad_id":    adID,
		"period":   startDate + " -> " + endDate,
		"insights": insights,
	})
}

// ExchangeToken godoc
// @Summary Connect a Meta user (exchange token + save credentials)
// @Description Exchanges a short-lived Meta token for a long-lived one (60 days),
// fetches the user's Meta ID, then creates or updates the user row in the local
// database. Use the X-User-Email header on all subsequent requests.
// @Tags auth
// @Accept json
// @Produce json
// @Param body body models.MetaConnectRequest true "Connection payload"
// @Success 200 {object} map[string]interface{}
// @Router /meta/auth/token [post]
func ExchangeToken(w http.ResponseWriter, r *http.Request) {
	var req models.MetaConnectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if req.ShortToken == "" || req.Email == "" || req.AdAccountID == "" {
		http.Error(w, "short_token, email, and ad_account_id are required", http.StatusBadRequest)
		return
	}

	// Step 1: exchange for long-lived token
	tokenResp, err := services.GetLongLivedToken(req.ShortToken)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Step 2: upsert the user row in the local DB
	user, err := services.UpsertMetaUser(
		req.Email,
		tokenResp.AccessToken,
		req.AdAccountID,
		req.PageID,
		req.PixelID,
		req.CatalogID,
	)
	if err != nil {
		http.Error(w, "Token exchanged but failed to save user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":          "User connected successfully. Use X-User-Email header for subsequent requests.",
		"email":            user.Email,
		"meta_user_id":     user.MetaUserID,
		"ad_account_id":    user.MetaAdAccountID,
		"long_lived_token": tokenResp.AccessToken,
		"expires_in":       tokenResp.ExpiresIn,
	})
}
