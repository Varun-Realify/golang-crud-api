package models

import (
	"time"

	"gorm.io/gorm"
)

// User matches the existing DB schema where id is a UUID column.
// We do not embed gorm.Model because that uses uint; the DB uses uuid.
type User struct {
	ID        string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Email           string `gorm:"uniqueIndex;not null" json:"email"`
	Name            string `json:"name"`
	MetaUserID      string `gorm:"uniqueIndex" json:"meta_user_id"`
	MetaAccessToken string `json:"-"` // never expose in JSON
	MetaAdAccountID string `json:"meta_ad_account_id"`
	MetaPageID      string `json:"meta_page_id"`
	MetaCatalogID   string `json:"meta_catalog_id"`
	MetaPixelID     string `json:"meta_pixel_id"`
}
