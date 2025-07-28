package models

import (
	"fmt"
	"time"
)

// Transaction status constants
const (
	StatusPending   = "pending"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
	StatusCancelled = "cancelled"
)

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

//         TRANSACTION STATE MANAGEMENT METHODS

// ValidateStatus status'un geçerli olup olmadığını kontrol eder
func (t *Transaction) ValidateStatus() error {
	validStatuses := map[string]bool{
		StatusPending:   true,
		StatusCompleted: true,
		StatusFailed:    true,
		StatusCancelled: true,
	}

	if !validStatuses[t.Status] {
		return fmt.Errorf("geçersiz transaction status: %s. Geçerli statuslar: pending, completed, failed, cancelled", t.Status)
	}

	return nil
}

// CanTransition belirli bir status'a geçiş yapılabilir mi kontrol eder
func (t *Transaction) CanTransition(newStatus string) error {
	// Önce yeni status'un geçerli olup olmadığını kontrol et
	tempTx := &Transaction{Status: newStatus}
	if err := tempTx.ValidateStatus(); err != nil {
		return err
	}

	// State transition rules (finite state machine)
	transitions := map[string][]string{
		StatusPending:   {StatusCompleted, StatusFailed, StatusCancelled},
		StatusCompleted: {}, // Completed'dan başka yere geçilemez
		StatusFailed:    {}, // Failed'dan başka yere geçilemez
		StatusCancelled: {}, // Cancelled'dan başka yere geçilemez
	}

	allowedTransitions, exists := transitions[t.Status]
	if !exists {
		return fmt.Errorf("mevcut status geçersiz: %s", t.Status)
	}

	// Aynı status'a geçiş kontrolü
	if t.Status == newStatus {
		return fmt.Errorf("transaction zaten %s durumunda", newStatus)
	}

	// İzin verilen geçişleri kontrol et
	for _, allowed := range allowedTransitions {
		if allowed == newStatus {
			return nil // Geçiş izinli
		}
	}

	return fmt.Errorf("'%s' durumundan '%s' durumuna geçiş yapılamaz. İzin verilen geçişler: %v",
		t.Status, newStatus, allowedTransitions)
}

// SetStatus status'u güvenli bir şekilde değiştirir
func (t *Transaction) SetStatus(newStatus string) error {
	// Geçiş kontrolü
	if err := t.CanTransition(newStatus); err != nil {
		return err
	}

	// Status'u değiştir
	t.Status = newStatus
	return nil
}

// GetValidTransitions mevcut status'tan geçilebilecek status'ları döner
func (t *Transaction) GetValidTransitions() []string {
	transitions := map[string][]string{
		StatusPending:   {StatusCompleted, StatusFailed, StatusCancelled},
		StatusCompleted: {},
		StatusFailed:    {},
		StatusCancelled: {},
	}

	if allowedTransitions, exists := transitions[t.Status]; exists {
		return allowedTransitions
	}

	return []string{}
}

//            STATUS CHECK METHODS

// IsPending transaction pending durumunda mı
func (t *Transaction) IsPending() bool {
	return t.Status == StatusPending
}

// IsCompleted transaction completed durumunda mı
func (t *Transaction) IsCompleted() bool {
	return t.Status == StatusCompleted
}

// IsFailed transaction failed durumunda mı
func (t *Transaction) IsFailed() bool {
	return t.Status == StatusFailed
}

// IsCancelled transaction cancelled durumunda mı
func (t *Transaction) IsCancelled() bool {
	return t.Status == StatusCancelled
}

// IsFinished transaction bitmiş durumda mı (completed, failed, cancelled)
func (t *Transaction) IsFinished() bool {
	return t.IsCompleted() || t.IsFailed() || t.IsCancelled()
}

// CanBeModified transaction değiştirilebilir mi (sadece pending'de değiştirilebilir)
func (t *Transaction) CanBeModified() bool {
	return t.IsPending()
}

//            TRANSACTION TYPE VALIDATION

// ValidateType transaction type'ının geçerli olup olmadığını kontrol eder
func (t *Transaction) ValidateType() error {
	validTypes := map[string]bool{
		"credit":   true,
		"debit":    true,
		"transfer": true,
	}

	if !validTypes[t.Type] {
		return fmt.Errorf("geçersiz transaction type: %s. Geçerli tipler: credit, debit, transfer", t.Type)
	}

	return nil
}

// IsCredit credit transaction mı
func (t *Transaction) IsCredit() bool {
	return t.Type == "credit"
}

// IsDebit debit transaction mı
func (t *Transaction) IsDebit() bool {
	return t.Type == "debit"
}

// IsTransfer transfer transaction mı
func (t *Transaction) IsTransfer() bool {
	return t.Type == "transfer"
}

//               TRANSACTION VALIDATION

// Validate transaction'ın tüm alanlarını doğrular
func (t *Transaction) Validate() error {
	// Amount kontrolü
	if t.Amount <= 0 {
		return fmt.Errorf("transaction miktarı sıfırdan büyük olmalıdır")
	}

	// Type kontrolü
	if err := t.ValidateType(); err != nil {
		return err
	}

	// Status kontrolü
	if err := t.ValidateStatus(); err != nil {
		return err
	}

	// Type'a göre user ID kontrolü
	switch t.Type {
	case "credit":
		if t.ToUserID == nil {
			return fmt.Errorf("credit transaction için to_user_id gerekli")
		}
		if t.FromUserID != nil {
			return fmt.Errorf("credit transaction için from_user_id olmamalı")
		}
	case "debit":
		if t.FromUserID == nil {
			return fmt.Errorf("debit transaction için from_user_id gerekli")
		}
		if t.ToUserID != nil {
			return fmt.Errorf("debit transaction için to_user_id olmamalı")
		}
	case "transfer":
		if t.FromUserID == nil || t.ToUserID == nil {
			return fmt.Errorf("transfer transaction için hem from_user_id hem to_user_id gerekli")
		}
		if *t.FromUserID == *t.ToUserID {
			return fmt.Errorf("transfer transaction'da from_user_id ve to_user_id aynı olamaz")
		}
	}

	return nil
}

//           TRANSACTION FACTORY METHODS

// NewCreditTransaction yeni credit transaction oluşturur
func NewCreditTransaction(toUserID int, amount float64, description string) *Transaction {
	return &Transaction{
		ToUserID:    &toUserID,
		FromUserID:  nil,
		Amount:      amount,
		Type:        "credit",
		Status:      StatusPending,
		Description: description,
		CreatedAt:   time.Now(),
	}
}

// NewDebitTransaction yeni debit transaction oluşturur
func NewDebitTransaction(fromUserID int, amount float64, description string) *Transaction {
	return &Transaction{
		FromUserID:  &fromUserID,
		ToUserID:    nil,
		Amount:      amount,
		Type:        "debit",
		Status:      StatusPending,
		Description: description,
		CreatedAt:   time.Now(),
	}
}

// NewTransferTransaction yeni transfer transaction oluşturur
func NewTransferTransaction(fromUserID, toUserID int, amount float64, description string) *Transaction {
	return &Transaction{
		FromUserID:  &fromUserID,
		ToUserID:    &toUserID,
		Amount:      amount,
		Type:        "transfer",
		Status:      StatusPending,
		Description: description,
		CreatedAt:   time.Now(),
	}
}

//         REQUEST VALIDATION METHODS

// Validate TransferRequest'i doğrular
func (req *TransferRequest) Validate() error {
	if req.ToUserID <= 0 {
		return fmt.Errorf("geçersiz kullanıcı ID")
	}

	if req.Amount <= 0 {
		return fmt.Errorf("miktar sıfırdan büyük olmalıdır")
	}

	if req.Amount > 1000000 {
		return fmt.Errorf("maksimum transfer limiti: 1,000,000 TL")
	}

	return nil
}

// Validate CreditRequest'i doğrular
func (req *CreditRequest) Validate() error {
	if req.Amount <= 0 {
		return fmt.Errorf("miktar sıfırdan büyük olmalıdır")
	}

	if req.Amount > 1000000 {
		return fmt.Errorf("maksimum yatırma limiti: 1,000,000 TL")
	}

	return nil
}

// Validate DebitRequest'i doğrular
func (req *DebitRequest) Validate() error {
	if req.Amount <= 0 {
		return fmt.Errorf("miktar sıfırdan büyük olmalıdır")
	}

	if req.Amount > 1000000 {
		return fmt.Errorf("maksimum çekme limiti: 1,000,000 TL")
	}

	return nil
}
