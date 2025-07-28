package models

import "time"

type Transaction struct {
	ID          int       `json:"id" db:"id"`
	FromUserID  *int      `json:"from_user_id" db:"from_user_id"`
	ToUserID    *int      `json:"to_user_id" db:"to_user_id"`
	Amount      float64   `json:"amount" db:"amount"`
	Type        string    `json:"type" db:"type"`
	Status      string    `json:"status" db:"status"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type TransferRequest struct {
	ToUserID    int     `json:"to_user_id"`
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
}

// CreditRequest hesaba para yatırma isteği
type CreditRequest struct {
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
}

// DebitRequest hesaptan para çekme isteği
type DebitRequest struct {
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
}

// DebitResponse para çekme yanıtı
type DebitResponse struct {
	Success     bool                `json:"success"`
	Transaction *TransactionSummary `json:"transaction"`
	NewBalance  float64             `json:"new_balance"`
	Message     string              `json:"message"`
}

// CreditResponse para yatırma yanıtı
type CreditResponse struct {
	Success     bool                `json:"success"`
	Transaction *TransactionSummary `json:"transaction"`
	NewBalance  float64             `json:"new_balance"`
	Message     string              `json:"message"`
}

// TransactionSummary hassas bilgileri filtrelenmiş transaction
type TransactionSummary struct {
	ID          int     `json:"id"`
	Amount      float64 `json:"amount"`
	Type        string  `json:"type"`
	Status      string  `json:"status"`
	Description string  `json:"description"`
	CreatedAt   string  `json:"created_at"`
	// UserID'ler ve diğer hassas bilgiler dahil edilmez
}

// TransactionStats kullanıcının transaction istatistikleri
type TransactionStats struct {
	UserID              int     `json:"user_id" db:"user_id"`
	TotalTransactions   int     `json:"total_transactions" db:"total_transactions"`
	TotalCredits        int     `json:"total_credits" db:"total_credits"`
	TotalDebits         int     `json:"total_debits" db:"total_debits"`
	TotalTransfers      int     `json:"total_transfers" db:"total_transfers"`
	TotalCreditAmount   float64 `json:"total_credit_amount" db:"total_credit_amount"`
	TotalDebitAmount    float64 `json:"total_debit_amount" db:"total_debit_amount"`
	TotalTransferAmount float64 `json:"total_transfer_amount" db:"total_transfer_amount"`
	LastTransactionDate *string `json:"last_transaction_date" db:"last_transaction_date"`
}
