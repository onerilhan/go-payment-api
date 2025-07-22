package models

import (
	"encoding/json"
	"time"
)

// AuditLog audit log modelini temsil eder
type AuditLog struct {
	ID         int             `json:"id" db:"id"`
	EntityType string          `json:"entity_type" db:"entity_type"`
	EntityID   int             `json:"entity_id" db:"entity_id"`
	Action     string          `json:"action" db:"action"`
	UserID     *int            `json:"user_id" db:"user_id"`
	OldData    json.RawMessage `json:"old_data" db:"old_data"`
	NewData    json.RawMessage `json:"new_data" db:"new_data"`
	Details    string          `json:"details" db:"details"`
	IPAddress  string          `json:"ip_address" db:"ip_address"`
	UserAgent  string          `json:"user_agent" db:"user_agent"`
	CreatedAt  time.Time       `json:"created_at" db:"created_at"`
}
