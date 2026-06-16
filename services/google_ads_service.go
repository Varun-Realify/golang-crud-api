package services

import (
	"Realify/config"
	"Realify/database"
	"Realify/models"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	GoogleAdsBaseURL = "https://googleads.googleapis.com/v24" // Reverting to v24 as it was found previously
)

func SystemTime() int64 {
	return time.Now().Unix()
}

func googleAdsRequest(method string, path string, payload interface{}) ([]byte, error) {
	customerID := os.Getenv("GOOGLE_ADS_CUSTOMER_ID")
	loginCustomerID := os.Getenv("GOOGLE_ADS_LOGIN_CUSTOMER_ID")
	developerToken := os.Getenv("GOOGLE_ADS_DEVELOPER_TOKEN")
	accessToken, err := config.GetGoogleAccessToken()
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/customers/%s/%s", GoogleAdsBaseURL, customerID, path)
	if path == "customers:listAccessibleCustomers" {
		url = fmt.Sprintf("%s/%s", GoogleAdsBaseURL, path)
	}

	var body io.Reader
	if payload != nil {
		jsonData, _ := json.Marshal(payload)
		body = bytes.NewBuffer(jsonData)
	}

	req, _ := http.NewRequest(method, url, body)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("developer-token", developerToken)
	if loginCustomerID != "" {
		req.Header.Set("login-customer-id", loginCustomerID)
	}
	req.Header.Set("Content-Type", "application/json")

	fmt.Printf("DEBUG: [%s] %s\n", method, url)

	client := &http.Client{Timeout: 60 * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	responseBytes, err := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("ERROR: Google Ads API [%d]: %s\n", resp.StatusCode, string(responseBytes))
		return nil, fmt.Errorf("google ads API error [%d]: %s", resp.StatusCode, string(responseBytes))
	}

	return responseBytes, err
}

func GetGoogleCustomers() (interface{}, error) {
	bodyBytes, err := googleAdsRequest("GET", "customers:listAccessibleCustomers", nil)
	if err != nil {
		return nil, err
	}
	var result interface{}
	json.Unmarshal(bodyBytes, &result)
	return result, nil
}

func ListGoogleCampaigns() (interface{}, error) {
	var campaigns []models.GoogleCampaign
	result := database.DB.Find(&campaigns)
	if result.Error != nil {
		return nil, result.Error
	}
	return campaigns, nil
}

func SyncGoogleCampaigns() error {
	query := map[string]string{
		"query": "SELECT campaign.id, campaign.name, campaign.status, campaign.resource_name FROM campaign",
	}
	body, err := googleAdsRequest("POST", "googleAds:search", query)
	if err != nil {
		return err
	}

	var response struct {
		Results []struct {
			Campaign struct {
				ID           string `json:"id"`
				Name         string `json:"name"`
				Status       string `json:"status"`
				ResourceName string `json:"resourceName"`
			} `json:"campaign"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return err
	}

	for _, r := range response.Results {
		dbCampaign := models.GoogleCampaign{
			GoogleID:     r.Campaign.ID,
			Name:         r.Campaign.Name,
			Status:       r.Campaign.Status,
			ResourceName: r.Campaign.ResourceName,
		}
		// Upsert using GoogleID as unique key (implied by previous logic)
		if err := database.DB.Where("google_id = ?", dbCampaign.GoogleID).Save(&dbCampaign).Error; err != nil {
			fmt.Printf("Error saving Google campaign %s: %v\n", dbCampaign.GoogleID, err)
		}
	}
	return nil
}

func ListGoogleAdGroups(campaignResourceName string) (interface{}, error) {
	var adGroups []models.GoogleAdGroup
	query := database.DB
	if campaignResourceName != "" {
		query = query.Where("campaign_resource_name = ?", campaignResourceName)
	}
	result := query.Find(&adGroups)
	if result.Error != nil {
		return nil, result.Error
	}
	return adGroups, nil
}

func ListGoogleAds(adGroupResourceName string) (interface{}, error) {
	var ads []models.GoogleAd
	query := database.DB
	if adGroupResourceName != "" {
		query = query.Where("ad_group_resource_name = ?", adGroupResourceName)
	}
	result := query.Find(&ads)
	if result.Error != nil {
		return nil, result.Error
	}
	return ads, nil
}

func CreateGoogleKeywords(req models.GoogleKeywordRequest) (interface{}, error) {
	operations := []map[string]interface{}{}
	for _, keyword := range req.Keywords {
		operations = append(operations, map[string]interface{}{
			"create": map[string]interface{}{
				"ad_group": req.AdGroupResourceName,
				"status":   "ENABLED",
				"keyword": map[string]interface{}{
					"text":       keyword,
					"match_type": req.MatchType,
				},
				"cpc_bid_micros": req.CpcBidMicros,
			},
		})
	}
	payload := map[string]interface{}{
		"operations": operations,
	}
	body, err := googleAdsRequest("POST", "adGroupCriteria:mutate", payload)
	if err != nil {
		return nil, err
	}
	var result interface{}
	json.Unmarshal(body, &result)
	return result, nil
}

func ListGoogleKeywords(adGroupResourceName string) (interface{}, error) {
	query := map[string]string{
		"query": fmt.Sprintf(
			"SELECT ad_group_criterion.criterion_id, ad_group_criterion.keyword.text, ad_group_criterion.keyword.match_type, ad_group_criterion.status, ad_group_criterion.cpc_bid_micros FROM ad_group_criterion WHERE ad_group_criterion.ad_group = '%s' AND ad_group_criterion.type = 'KEYWORD'",
			adGroupResourceName,
		),
	}
	body, err := googleAdsRequest("POST", "googleAds:search", query)
	if err != nil {
		return nil, err
	}
	var result interface{}
	json.Unmarshal(body, &result)
	return result, nil
}

func DeleteGoogleKeyword(req models.GoogleKeywordDeleteRequest) (interface{}, error) {
	payload := map[string]interface{}{
		"operations": []map[string]interface{}{
			{
				"remove": req.ResourceName,
			},
		},
	}
	body, err := googleAdsRequest("POST", "adGroupCriteria:mutate", payload)
	if err != nil {
		return nil, err
	}
	var result interface{}
	json.Unmarshal(body, &result)
	return result, nil
}

func GetGoogleCampaignPerformanceInsights(campaignID string) (interface{}, error) {
	query := map[string]string{
		"query": fmt.Sprintf(`
            SELECT
                campaign.id,
                campaign.name,
                campaign.status,
                metrics.impressions,
                metrics.clicks,
                metrics.cost_micros,
                metrics.conversions,
                metrics.ctr,
                metrics.average_cpc
            FROM campaign
            WHERE campaign.id = '%s'
            AND segments.date DURING LAST_30_DAYS
        `, campaignID),
	}
	body, err := googleAdsRequest("POST", "googleAds:search", query)
	if err != nil {
		return nil, err
	}
	var result interface{}
	json.Unmarshal(body, &result)
	return result, nil
}

func GetGoogleAdGroupPerformanceInsights(adGroupID string) (interface{}, error) {
	query := map[string]string{
		"query": fmt.Sprintf(`
            SELECT
                ad_group.id,
                ad_group.name,
                ad_group.status,
                metrics.impressions,
                metrics.clicks,
                metrics.cost_micros,
                metrics.conversions,
                metrics.ctr,
                metrics.average_cpc
            FROM ad_group
            WHERE ad_group.id = '%s'
            AND segments.date DURING LAST_30_DAYS
        `, adGroupID),
	}
	body, err := googleAdsRequest("POST", "googleAds:search", query)
	if err != nil {
		return nil, err
	}
	var result interface{}
	json.Unmarshal(body, &result)
	return result, nil
}

func CreateGoogleCampaign(req models.GoogleCampaignRequest) (interface{}, error) {
	budgetPayload := map[string]interface{}{
		"operations": []map[string]interface{}{
			{
				"create": map[string]interface{}{
					"name":              fmt.Sprintf("Budget-%s-%d", req.Name, SystemTime()),
					"amount_micros":     req.DailyBudgetMicros,
					"delivery_method":   "STANDARD",
					"explicitly_shared": false,
				},
			},
		},
	}

	budgetResp, err := googleAdsRequest("POST", "campaignBudgets:mutate", budgetPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to create budget: %v", err)
	}

	var budgetResult struct {
		Results []struct {
			ResourceName string `json:"resourceName"`
		} `json:"results"`
	}
	if err := json.Unmarshal(budgetResp, &budgetResult); err != nil || len(budgetResult.Results) == 0 {
		return nil, fmt.Errorf("invalid budget response: %s", string(budgetResp))
	}

	budgetResourceName := budgetResult.Results[0].ResourceName

	payload := map[string]interface{}{
		"operations": []map[string]interface{}{
			{
				"create": map[string]interface{}{
					"name":                              req.Name,
					"advertising_channel_type":          "SEARCH",
					"status":                            "ENABLED",
					"campaign_budget":                   budgetResourceName,
					"contains_eu_political_advertising": "DOES_NOT_CONTAIN_EU_POLITICAL_ADVERTISING",
					"manual_cpc": map[string]interface{}{
						"enhanced_cpc_enabled": false,
					},
					"network_settings": map[string]interface{}{
						"target_google_search":   true,
						"target_search_network":  true,
						"target_content_network": false,
					},
				},
			},
		},
	}

	campaignResp, err := googleAdsRequest("POST", "campaigns:mutate", payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create campaign: %v", err)
	}

	var googleResponse struct {
		Results []struct {
			ResourceName string `json:"resourceName"`
		} `json:"results"`
	}
	if err := json.Unmarshal(campaignResp, &googleResponse); err != nil || len(googleResponse.Results) == 0 {
		return nil, fmt.Errorf("invalid campaign response: %s", string(campaignResp))
	}

	resourceName := googleResponse.Results[0].ResourceName
	parts := strings.Split(resourceName, "/")
	googleID := parts[len(parts)-1]

	// Ingest into Database
	dbCampaign := models.GoogleCampaign{
		GoogleID:     googleID,
		ResourceName: resourceName,
		Name:         req.Name,
		Status:       "ENABLED",
	}
	if err := database.DB.Create(&dbCampaign).Error; err != nil {
		return nil, fmt.Errorf("campaign created on Google but DB save failed: %w", err)
	}

	return map[string]interface{}{
		"status":        "success",
		"message":       "Campaign created and ingested successfully",
		"id":            googleID,
		"resource_name": resourceName,
		"db_record":     dbCampaign,
	}, nil
}

func DeleteGoogleCampaign(resourceName string) (interface{}, error) {
	payload := map[string]interface{}{
		"operations": []map[string]interface{}{
			{
				"remove": resourceName,
			},
		},
	}

	body, err := googleAdsRequest("POST", "campaigns:mutate", payload)
	if err != nil {
		return nil, err
	}

	// Remove from Database
	database.DB.Where("resource_name = ?", resourceName).Delete(&models.GoogleCampaign{})

	var result interface{}
	json.Unmarshal(body, &result)
	return result, nil
}

func CreateGoogleAdGroup(req models.GoogleAdGroupRequest) (interface{}, error) {
	payload := map[string]interface{}{
		"operations": []map[string]interface{}{
			{
				"create": map[string]interface{}{
					"name":     req.Name,
					"campaign": req.CampaignResourceName,
					"status":   "ENABLED",
					"type":     "SEARCH_STANDARD",
				},
			},
		},
	}
	body, err := googleAdsRequest("POST", "adGroups:mutate", payload)
	if err != nil {
		return nil, err
	}

	var googleResponse struct {
		Results []struct {
			ResourceName string `json:"resourceName"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &googleResponse); err != nil || len(googleResponse.Results) == 0 {
		return nil, fmt.Errorf("invalid adgroup response: %s", string(body))
	}

	resourceName := googleResponse.Results[0].ResourceName
	parts := strings.Split(resourceName, "/")
	googleID := parts[len(parts)-1]

	// Ingest into Database
	dbAdGroup := models.GoogleAdGroup{
		GoogleID:             googleID,
		ResourceName:         resourceName,
		Name:                 req.Name,
		Status:               "ENABLED",
		CampaignResourceName: req.CampaignResourceName,
	}
	if err := database.DB.Create(&dbAdGroup).Error; err != nil {
		return nil, fmt.Errorf("adgroup created on Google but DB save failed: %w", err)
	}

	return dbAdGroup, nil
}

func CreateGoogleAd(req models.GoogleAdRequest) (interface{}, error) {
	payload := map[string]interface{}{
		"operations": []map[string]interface{}{
			{
				"create": map[string]interface{}{
					"ad_group": req.AdGroupResourceName,
					"status":   "ENABLED",
					"ad": map[string]interface{}{
						"final_urls": []string{req.FinalUrl},
						"responsive_search_ad": map[string]interface{}{
							"headlines": []map[string]interface{}{
								{"text": req.Headline},
								{"text": "Best " + req.Headline},
								{"text": req.Headline + " Today"},
							},
							"descriptions": []map[string]interface{}{
								{"text": req.Description},
								{"text": "Learn more about " + req.Headline},
							},
						},
					},
				},
			},
		},
	}
	body, err := googleAdsRequest("POST", "adGroupAds:mutate", payload)
	if err != nil {
		return nil, err
	}

	var googleResponse struct {
		Results []struct {
			ResourceName string `json:"resourceName"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &googleResponse); err != nil || len(googleResponse.Results) == 0 {
		return nil, fmt.Errorf("invalid ad response: %s", string(body))
	}

	resourceName := googleResponse.Results[0].ResourceName
	// ResourceName for adGroupAd looks like customers/123/adGroupAds/456~789
	parts := strings.Split(resourceName, "/")
	googleID := parts[len(parts)-1]

	// Ingest into Database
	dbAd := models.GoogleAd{
		GoogleID:            googleID,
		ResourceName:        resourceName,
		Status:              "ENABLED",
		AdGroupResourceName: req.AdGroupResourceName,
		FinalURL:            req.FinalUrl,
	}
	if err := database.DB.Create(&dbAd).Error; err != nil {
		return nil, fmt.Errorf("ad created on Google but DB save failed: %w", err)
	}

	return dbAd, nil
}
