package handlers

import (
	"Realify/config"
	"Realify/database"
	"Realify/models"
	"net/http"
)

// GetUserContext fetches the user specific credentials.
// For demo purposes, it checks "X-User-Email" header.
// If not provided, it falls back to the system default (.env).
func GetUserContext(r *http.Request) models.User {
	email := r.Header.Get("X-User-Email")
	cfg := config.GetConfig()

	// Default User based on .env
	defaultUser := models.User{
		Email:           "admin@realify.com",
		MetaAccessToken: cfg.MetaAccessToken,
		MetaAdAccountID: cfg.MetaAdAccountID,
		MetaPageID:      cfg.MetaPageID,
		MetaCatalogID:   cfg.MetaCatalogID,
		MetaPixelID:     cfg.MetaPixelID,
	}

	if email == "" {
		return defaultUser
	}

	var user models.User
	result := database.DB.Where("email = ?", email).First(&user)
	if result.Error != nil {
		return defaultUser
	}

	return user
}
