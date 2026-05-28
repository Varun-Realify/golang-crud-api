package config

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
)

// GetGoogleAccessToken exchanges the refresh token for a fresh access token
func GetGoogleAccessToken() (string, error) {
	data := map[string]string{
		"client_id":     os.Getenv("GOOGLE_ADS_CLIENT_ID"),
		"client_secret": os.Getenv("GOOGLE_ADS_CLIENT_SECRET"),
		"refresh_token": os.Getenv("GOOGLE_ADS_REFRESH_TOKEN"),
		"redirect_uri":  os.Getenv("GOOGLE_ADS_REDIRECT_URL"),
		"grant_type":    "refresh_token",
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(
		"https://oauth2.googleapis.com/token",
		"application/json",
		bytes.NewBuffer(jsonData),
	)

	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	token, ok := result["access_token"].(string)
	if !ok {
		return "", io.ErrUnexpectedEOF
	}

	return token, nil
}

// ExchangeGoogleCodeForToken exchanges the authorization code for access and refresh tokens
func ExchangeGoogleCodeForToken(code string) (map[string]interface{}, error) {
	data := map[string]string{
		"client_id":     os.Getenv("GOOGLE_ADS_CLIENT_ID"),
		"client_secret": os.Getenv("GOOGLE_ADS_CLIENT_SECRET"),
		"code":          code,
		"redirect_uri":  os.Getenv("GOOGLE_ADS_REDIRECT_URL"),
		"grant_type":    "authorization_code",
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(
		"https://oauth2.googleapis.com/token",
		"application/json",
		bytes.NewBuffer(jsonData),
	)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result, nil
}
