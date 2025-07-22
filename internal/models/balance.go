package models

import "time"

// Balance kullanıcı bakiye modelini temsil eder
type Balance struct {
	UserID        int       `json:"user_id" db:"user_id"`
	Amount        float64   `json:"amount" db:"amount"`
	LastUpdatedAt time.Time `json:"last_updated_at" db:"last_updated_at"`
}
