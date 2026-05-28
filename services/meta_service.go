package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"Realify/config"
	"Realify/database"
	"Realify/models"
)

const MetaBaseURL = "https://graph.facebook.com/v18.0"

func metaRequest(method, path string, params url.Values, payload interface{}, accessToken string) ([]byte, error) {
	if params == nil {
		params = url.Values{}
	}
	params.Set("access_token", accessToken)

	fullURL := fmt.Sprintf("%s/%s?%s", MetaBaseURL, path, params.Encode())

	var body io.Reader
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, fullURL, body)
	if err != nil {
		return nil, err
	}

	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var metaErr struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    int    `json:"code"`
		} `json:"error"`
	}

	if json.Unmarshal(respBody, &metaErr) == nil && metaErr.Error.Message != "" {
		return nil, fmt.Errorf("Meta API Error %d: %s", metaErr.Error.Code, metaErr.Error.Message)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("Meta API HTTP Error %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func TestMetaConnection(accessToken string) (map[string]interface{}, error) {
	body, err := metaRequest("GET", "me", url.Values{"fields": {"id,name"}}, nil, accessToken)
	if err != nil {
		return map[string]interface{}{"connected": false, "error": err.Error()}, nil
	}

	var data map[string]interface{}
	json.Unmarshal(body, &data)

	return map[string]interface{}{
		"connected": true,
		"user":      data["name"],
		"user_id":   data["id"],
	}, nil
}

func GetAvailablePages(accessToken string) ([]map[string]interface{}, error) {
	body, err := metaRequest("GET", "me/accounts", url.Values{"fields": {"id,name,access_token,tasks"}}, nil, accessToken)
	if err != nil {
		return nil, err
	}

	var response struct {
		Data []map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	return response.Data, nil
}

func GetAvailableCatalogs(accessToken string) ([]map[string]interface{}, error) {
	body, err := metaRequest("GET", "me/catalogs", url.Values{"fields": {"id,name,vertical"}}, nil, accessToken)
	if err != nil {
		return nil, err
	}

	var response struct {
		Data []map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	return response.Data, nil
}

func GetAllCampaigns(adAccountID string, accessToken string) ([]models.MetaCampaignRecord, error) {
	var campaigns []models.MetaCampaignRecord
	result := database.DB.Where("ad_account_id = ?", adAccountID).Find(&campaigns)
	if result.Error != nil {
		return nil, result.Error
	}
	return campaigns, nil
}

func SyncMetaCampaigns(adAccountID string, accessToken string) error {
	params := url.Values{
		"fields": {"id,name,status,objective,start_time,stop_time,daily_budget,lifetime_budget"},
	}
	body, err := metaRequest("GET", adAccountID+"/campaigns", params, nil, accessToken)
	if err != nil {
		return err
	}

	var response struct {
		Data []struct {
			ID             string `json:"id"`
			Name           string `json:"name"`
			Status         string `json:"status"`
			Objective      string `json:"objective"`
			StartTime      string `json:"start_time"`
			StopTime       string `json:"stop_time"`
			DailyBudget    string `json:"daily_budget"`
			LifetimeBudget string `json:"lifetime_budget"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return err
	}

	for _, c := range response.Data {
		daily := parseAmount(c.DailyBudget)
		lifetime := parseAmount(c.LifetimeBudget)
		total := daily
		if lifetime > 0 {
			total = lifetime
		}

		dbRecord := models.MetaCampaignRecord{
			ID:             c.ID,
			Name:           c.Name,
			Status:         c.Status,
			Objective:      c.Objective,
			StartTime:      formatTime(c.StartTime),
			StopTime:       formatTime(c.StopTime),
			DailyBudget:    daily,
			LifetimeBudget: lifetime,
			TotalBudget:    total,
			AdAccountID:    adAccountID,
		}
		// Upsert
		if err := database.DB.Save(&dbRecord).Error; err != nil {
			fmt.Printf("Error saving campaign %s: %v\n", c.ID, err)
		}
	}
	return nil
}

func CreateCampaign(adAccountID string, accessToken string, req models.CampaignCreate) (map[string]interface{}, error) {
	body, err := metaRequest("POST", adAccountID+"/campaigns", nil, req, accessToken)
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	json.Unmarshal(body, &data)

	if id, ok := data["id"].(string); ok {
		params := url.Values{
			"fields": {"id,name,status,objective,start_time,stop_time,daily_budget,lifetime_budget"},
		}
		fullBody, _ := metaRequest("GET", id, params, nil, accessToken)
		var c struct {
			ID             string `json:"id"`
			Name           string `json:"name"`
			Status         string `json:"status"`
			Objective      string `json:"objective"`
			StartTime      string `json:"start_time"`
			StopTime       string `json:"stop_time"`
			DailyBudget    string `json:"daily_budget"`
			LifetimeBudget string `json:"lifetime_budget"`
		}
		json.Unmarshal(fullBody, &c)

		daily := parseAmount(c.DailyBudget)
		lifetime := parseAmount(c.LifetimeBudget)
		total := daily
		if lifetime > 0 {
			total = lifetime
		}

		dbRecord := models.MetaCampaignRecord{
			ID:             c.ID,
			Name:           c.Name,
			Status:         c.Status,
			Objective:      c.Objective,
			StartTime:      formatTime(c.StartTime),
			StopTime:       formatTime(c.StopTime),
			DailyBudget:    daily,
			LifetimeBudget: lifetime,
			TotalBudget:    total,
			AdAccountID:    adAccountID,
		}
		if err := database.DB.Create(&dbRecord).Error; err != nil {
			return nil, fmt.Errorf("campaign created on Meta but DB save failed: %w", err)
		}
		data["db_record"] = dbRecord
	}

	return data, nil
}

func CreateAdSet(adAccountID string, accessToken string, pixelID string, req models.AdSetCreate) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"name":              req.Name,
		"campaign_id":       req.CampaignID,
		"daily_budget":      req.DailyBudget,
		"billing_event":     "IMPRESSIONS",
		"optimization_goal": "OFFSITE_CONVERSIONS",
		"targeting":         req.Targeting,
		"status":            req.Status,
	}

	if pixelID != "" {
		payload["promoted_object"] = map[string]string{
			"pixel_id":          pixelID,
			"custom_event_type": "PURCHASE",
		}
	}

	body, err := metaRequest("POST", adAccountID+"/adsets", nil, payload, accessToken)
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	json.Unmarshal(body, &data)

	if id, ok := data["id"].(string); ok {
		params := url.Values{
			"fields": {"id,name,status,campaign_id,daily_budget"},
		}
		fullBody, _ := metaRequest("GET", id, params, nil, accessToken)
		var s struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Status      string `json:"status"`
			CampaignID  string `json:"campaign_id"`
			DailyBudget string `json:"daily_budget"`
		}
		json.Unmarshal(fullBody, &s)

		dbRecord := models.MetaAdSetRecord{
			ID:          s.ID,
			Name:        s.Name,
			CampaignID:  s.CampaignID,
			DailyBudget: parseAmount(s.DailyBudget),
			Status:      s.Status,
			AdAccountID: adAccountID,
		}
		if err := database.DB.Create(&dbRecord).Error; err != nil {
			return nil, fmt.Errorf("adset created on Meta but DB save failed: %w", err)
		}
		data["db_record"] = dbRecord
	}

	return data, nil
}

func CreateAd(adAccountID string, accessToken string, pageID string, req models.AdCreate) (map[string]interface{}, error) {
	creativePayload := map[string]interface{}{
		"name": req.Name + "_creative",
		"object_story_spec": map[string]interface{}{
			"page_id": pageID,
			"link_data": map[string]string{
				"image_url": req.ImageURL,
				"link":      req.LinkURL,
				"message":   req.BodyText,
				"caption":   req.Headline,
			},
		},
	}

	creativeBody, err := metaRequest("POST", adAccountID+"/adcreatives", nil, creativePayload, accessToken)
	if err != nil {
		return nil, err
	}

	var creativeData struct {
		ID string `json:"id"`
	}
	json.Unmarshal(creativeBody, &creativeData)

	adPayload := map[string]interface{}{
		"name":     req.Name,
		"adset_id": req.AdSetID,
		"creative": map[string]string{"creative_id": creativeData.ID},
		"status":   req.Status,
	}

	adBody, err := metaRequest("POST", adAccountID+"/ads", nil, adPayload, accessToken)
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	json.Unmarshal(adBody, &data)

	if id, ok := data["id"].(string); ok {
		params := url.Values{
			"fields": {"id,name,status,adset_id"},
		}
		fullBody, _ := metaRequest("GET", id, params, nil, accessToken)
		var a struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Status  string `json:"status"`
			AdSetID string `json:"adset_id"`
		}
		json.Unmarshal(fullBody, &a)

		dbRecord := models.MetaAdRecord{
			ID:          a.ID,
			Name:        a.Name,
			AdSetID:     a.AdSetID,
			Status:      a.Status,
			AdAccountID: adAccountID,
		}
		if err := database.DB.Create(&dbRecord).Error; err != nil {
			return nil, fmt.Errorf("ad created on Meta but DB save failed: %w", err)
		}
		data["db_record"] = dbRecord
	}

	return data, nil
}

func UpdateCampaign(campaignID string, accessToken string, req models.CampaignUpdate) (map[string]interface{}, error) {
	body, err := metaRequest("POST", campaignID, nil, req, accessToken)
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	json.Unmarshal(body, &data)

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Status != "" {
		updates["status"] = req.Status
	}
	if req.Objective != "" {
		updates["objective"] = req.Objective
	}

	if len(updates) > 0 {
		database.DB.Model(&models.MetaCampaignRecord{}).Where("id = ?", campaignID).Updates(updates)
	}

	return data, nil
}

func GetAdSetAds(adSetID string, accessToken string) ([]models.MetaAdRecord, error) {
	var ads []models.MetaAdRecord
	result := database.DB.Where("adset_id = ?", adSetID).Find(&ads)
	if result.Error != nil {
		return nil, result.Error
	}
	return ads, nil
}

func GetAdInsights(adID string, accessToken string, startDate, endDate string) (models.MetaInsights, error) {
	params := url.Values{
		"fields":     {"spend,impressions,clicks,actions,action_values,ctr,cpm,reach"},
		"time_range": {fmt.Sprintf(`{"since":"%s","until":"%s"}`, startDate, endDate)},
	}

	body, err := metaRequest("GET", adID+"/insights", params, nil, accessToken)
	if err != nil {
		return models.MetaInsights{}, err
	}

	return parseInsights(body)
}

func UpsertMetaUser(email, longLivedToken, adAccountID, pageID, pixelID, catalogID string) (*models.User, error) {
	body, err := metaRequest("GET", "me", url.Values{"fields": {"id,name"}}, nil, longLivedToken)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Meta user info: %w", err)
	}
	var me struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	json.Unmarshal(body, &me)

	var user models.User
	database.DB.Where("email = ?", email).FirstOrInit(&user)

	isNew := user.ID == ""

	user.Email = email
	user.Name = me.Name
	user.MetaUserID = me.ID
	user.MetaAccessToken = longLivedToken
	user.MetaAdAccountID = adAccountID
	user.MetaPageID = pageID
	user.MetaPixelID = pixelID
	user.MetaCatalogID = catalogID

	var dbErr error
	if isNew {
		dbErr = database.DB.Create(&user).Error
	} else {
		dbErr = database.DB.Save(&user).Error
	}
	if dbErr != nil {
		return nil, fmt.Errorf("failed to save user: %w", dbErr)
	}

	return &user, nil
}

func GetLongLivedToken(shortLivedToken string) (models.MetaTokenResponse, error) {
	cfg := config.GetConfig()
	params := url.Values{
		"grant_type":        {"fb_exchange_token"},
		"client_id":         {cfg.MetaAppID},
		"client_secret":     {cfg.MetaAppSecret},
		"fb_exchange_token": {shortLivedToken},
	}

	resp, err := http.Get(fmt.Sprintf("%s/oauth/access_token?%s", MetaBaseURL, params.Encode()))
	if err != nil {
		return models.MetaTokenResponse{}, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var response models.MetaTokenResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return models.MetaTokenResponse{}, err
	}

	return response, nil
}

func GetCampaignInsights(campaignID string, accessToken string, startDate, endDate string) (models.MetaInsights, error) {
	params := url.Values{
		"fields":     {"spend,impressions,clicks,actions,action_values,ctr,cpm,reach"},
		"time_range": {fmt.Sprintf(`{"since":"%s","until":"%s"}`, startDate, endDate)},
	}

	body, err := metaRequest("GET", campaignID+"/insights", params, nil, accessToken)
	if err != nil {
		return models.MetaInsights{}, err
	}

	return parseInsights(body)
}

func parseInsights(body []byte) (models.MetaInsights, error) {
	var response struct {
		Data []struct {
			Spend        string                   `json:"spend"`
			Impressions  string                   `json:"impressions"`
			Clicks       string                   `json:"clicks"`
			Actions      []map[string]interface{} `json:"actions"`
			ActionValues []map[string]interface{} `json:"action_values"`
			CTR          string                   `json:"ctr"`
			CPM          string                   `json:"cpm"`
			Reach        string                   `json:"reach"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return models.MetaInsights{}, err
	}

	if len(response.Data) == 0 {
		return models.MetaInsights{}, nil
	}

	ins := response.Data[0]
	var conversions int
	var revenue float64

	for _, action := range ins.Actions {
		if action["action_type"] == "purchase" {
			switch v := action["value"].(type) {
			case string:
				fmt.Sscanf(v, "%d", &conversions)
			case float64:
				conversions = int(v)
			}
		}
	}

	for _, av := range ins.ActionValues {
		if av["action_type"] == "purchase" {
			switch v := av["value"].(type) {
			case string:
				fmt.Sscanf(v, "%f", &revenue)
			case float64:
				revenue = v
			}
		}
	}

	var spend, ctr, cpm float64
	var impressions, clicks, reach int
	fmt.Sscanf(ins.Spend, "%f", &spend)
	fmt.Sscanf(ins.CTR, "%f", &ctr)
	fmt.Sscanf(ins.CPM, "%f", &cpm)
	fmt.Sscanf(ins.Impressions, "%d", &impressions)
	fmt.Sscanf(ins.Clicks, "%d", &clicks)
	fmt.Sscanf(ins.Reach, "%d", &reach)

	return models.MetaInsights{
		Spend:       spend,
		Impressions: impressions,
		Clicks:      clicks,
		Conversions: conversions,
		Revenue:     revenue,
		CTR:         ctr,
		CPM:         cpm,
		Reach:       reach,
	}, nil
}

func SyncCatalogProduct(catalogID string, accessToken string, name, description, link, imageURL string, price float64, brand string) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"item_type": "PRODUCT_ITEM",
		"requests": []map[string]interface{}{
			{
				"method": "UPDATE",
				"data": map[string]interface{}{
					"retailer_id":  fmt.Sprintf("go_%s", name),
					"name":         name,
					"description":  description,
					"url":          link,
					"image_url":    imageURL,
					"price":        int(price * 100),
					"currency":     "INR",
					"brand":        brand,
					"condition":    "new",
					"availability": "in stock",
				},
			},
		},
	}

	body, err := metaRequest("POST", catalogID+"/items_batch", nil, payload, accessToken)
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	json.Unmarshal(body, &data)
	return data, nil
}

func parseAmount(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f / 100
}

func formatTime(s string) string {
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}
