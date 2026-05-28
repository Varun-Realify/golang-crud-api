package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort string
	Host       string
	Port       string
	User       string
	Password   string
	DBName     string
	SSLMode    string

	// Meta Ads Config
	MetaAppID       string
	MetaAppSecret   string
	MetaAccessToken string
	MetaAdAccountID string
	MetaPixelID     string
	MetaPageID      string
	MetaCatalogID   string

	// Google Ads Config
	GoogleAdsCustomerID     string
	GoogleAdsDeveloperToken string
	GoogleAdsClientID       string
	GoogleAdsClientSecret   string
	GoogleAdsRefreshToken   string
	GoogleAdsRedirectURL    string
}

func GetConfig() Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	return Config{
		ServerPort: getEnv("SERVER_PORT", "8080"),
		Host:       getEnv("DB_HOST", "localhost"),
		Port:       getEnv("DB_PORT", "5432"),
		User:       getEnv("DB_USER", "postgres"),
		Password:   getEnv("DB_PASSWORD", "root"),
		DBName:     getEnv("DB_NAME", "Realify"),
		SSLMode:    getEnv("DB_SSLMODE", "disable"),

		MetaAppID:       getEnv("META_APP_ID", ""),
		MetaAppSecret:   getEnv("META_APP_SECRET", ""),
		MetaAccessToken: getEnv("META_ACCESS_TOKEN", ""),
		MetaAdAccountID: getEnv("META_AD_ACCOUNT_ID", ""),
		MetaPixelID:     getEnv("META_PIXEL_ID", ""),
		MetaPageID:      getEnv("META_PAGE_ID", ""),
		MetaCatalogID:   getEnv("META_CATALOG_ID", ""),

		GoogleAdsCustomerID:     getEnv("GOOGLE_ADS_CUSTOMER_ID", ""),
		GoogleAdsDeveloperToken: getEnv("GOOGLE_ADS_DEVELOPER_TOKEN", ""),
		GoogleAdsClientID:       getEnv("GOOGLE_ADS_CLIENT_ID", ""),
		GoogleAdsClientSecret:   getEnv("GOOGLE_ADS_CLIENT_SECRET", ""),
		GoogleAdsRefreshToken:   getEnv("GOOGLE_ADS_REFRESH_TOKEN", ""),
		GoogleAdsRedirectURL:    getEnv("GOOGLE_ADS_REDIRECT_URL", ""),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func (c Config) GetDSN() string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		c.Host, c.User, c.Password, c.DBName, c.Port, c.SSLMode)
}
