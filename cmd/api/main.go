package main

// @title Realify API
// @version 1.0
// @description Multi-Platform Ads Management API (Meta & Google) for Realify.
// @host localhost:8080
// @BasePath /

import (
	"log"
	"net/http"

	"Realify/config"
	"Realify/database"
	_ "Realify/docs"
	"Realify/handlers"
	"Realify/workers"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	httpSwagger "github.com/swaggo/http-swagger"
)

func main() {
	// Initialize Database
	database.InitDB()

	// Initialize Asynq Client
	workers.InitClient()
	defer workers.CloseClient()

	// Setup Router
	r := mux.NewRouter()

	// Serve Static Files (Frontend)
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./public"))))
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./public/index.html")
	})

	// Swagger Route
	r.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	// Meta Ads Routes
	r.HandleFunc("/meta/status", handlers.GetMetaStatus).Methods("GET")
	r.HandleFunc("/meta/campaigns", handlers.GetAllCampaigns).Methods("GET")
	r.HandleFunc("/meta/sync", handlers.SyncCampaigns).Methods("POST")
	r.HandleFunc("/meta/campaigns", handlers.CreateCampaign).Methods("POST")
	r.HandleFunc("/meta/campaigns/{id}", handlers.UpdateCampaign).Methods("PATCH")
	r.HandleFunc("/meta/adsets", handlers.CreateAdSet).Methods("POST")
	r.HandleFunc("/meta/adsets/{id}/ads", handlers.GetAdSetAds).Methods("GET")
	r.HandleFunc("/meta/ads", handlers.CreateAd).Methods("POST")
	r.HandleFunc("/meta/ads/{id}/insights", handlers.GetAdInsights).Methods("GET")
	r.HandleFunc("/meta/campaigns/{id}/insights", handlers.GetCampaignInsights).Methods("GET")
	r.HandleFunc("/meta/auth/token", handlers.ExchangeToken).Methods("POST")

	// Google Ads Routes
	r.HandleFunc("/google/auth", handlers.GoogleLogin).Methods("GET")
	r.HandleFunc("/callback", handlers.GoogleCallback).Methods("GET")
	r.HandleFunc("/google/customers", handlers.GetGoogleCustomers).Methods("GET")

	r.HandleFunc("/google/campaigns", handlers.ListGoogleCampaigns).Methods("GET")
	r.HandleFunc("/google/sync", handlers.SyncGoogleCampaigns).Methods("POST")
	r.HandleFunc("/google/campaigns", handlers.CreateGoogleCampaign).Methods("POST")
	r.HandleFunc("/google/campaigns", handlers.DeleteGoogleCampaign).Methods("DELETE")
	r.HandleFunc("/google/campaigns/{id}/insights", handlers.GetGoogleCampaignPerformanceInsights).Methods("GET")

	r.HandleFunc("/google/adgroups", handlers.ListGoogleAdGroups).Methods("GET")
	r.HandleFunc("/google/adgroups", handlers.CreateGoogleAdGroup).Methods("POST")
	r.HandleFunc("/google/adgroups/{id}/insights", handlers.GetGoogleAdGroupPerformanceInsights).Methods("GET")

	r.HandleFunc("/google/ads", handlers.ListGoogleAds).Methods("GET")
	r.HandleFunc("/google/ads", handlers.CreateGoogleAd).Methods("POST")

	// Keywords
	r.HandleFunc("/google/keywords", handlers.CreateGoogleKeywords).Methods("POST")
	r.HandleFunc("/google/keywords", handlers.ListGoogleKeywords).Methods("GET")
	r.HandleFunc("/google/keywords", handlers.DeleteGoogleKeyword).Methods("DELETE")

	// Apply CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})

	handler := c.Handler(r)

	// Get Config
	cfg := config.GetConfig()

	// Start Server
	log.Printf("Server starting on :%s\n", cfg.ServerPort)
	log.Fatal(http.ListenAndServe(":"+cfg.ServerPort, handler))
}
