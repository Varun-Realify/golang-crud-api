package database

import (
	"log"

	"Realify/config"
	"Realify/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {
	cfg := config.GetConfig()
	dsn := cfg.GetDSN()

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	log.Println("Database connection established")

	// Run Migrations
	DB.AutoMigrate(
		&models.User{},
		&models.GoogleCampaign{},
		&models.GoogleAdGroup{},
		&models.GoogleAd{},
		&models.MetaCampaignRecord{},
		&models.MetaAdSetRecord{},
		&models.MetaAdRecord{},
	)
	log.Println("Database migration completed")
}
