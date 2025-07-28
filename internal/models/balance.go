package models

import "time"

// Balance kullanıcı bakiye modelini temsil eder
type Balance struct {
	UserID        int       `json:"user_id" db:"user_id"`
	Amount        float64   `json:"amount" db:"amount"`
	LastUpdatedAt time.Time `json:"last_updated_at" db:"last_updated_at"`
}

// BalanceHistory kullanıcının bakiye geçmişini tutar
type BalanceHistory struct {
	ID             int       `json:"id" db:"id"`
	UserID         int       `json:"user_id" db:"user_id"`
	PreviousAmount float64   `json:"previous_amount" db:"previous_amount"`
	NewAmount      float64   `json:"new_amount" db:"new_amount"`
	ChangeAmount   float64   `json:"change_amount" db:"change_amount"`   // +/- değişim miktarı
	Reason         string    `json:"reason" db:"reason"`                 // "credit", "debit", "transfer_in", "transfer_out"
	TransactionID  *int      `json:"transaction_id" db:"transaction_id"` // İlgili transaction ID (opsiyonel)
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// BalanceAtTime belirli bir tarihte bakiye bilgisi
type BalanceAtTime struct {
	UserID  int     `json:"user_id"`
	Amount  float64 `json:"amount"`
	AtTime  string  `json:"at_time"`
	Message string  `json:"message"`
}
