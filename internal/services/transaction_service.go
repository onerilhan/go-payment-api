package services

import (
	"fmt"

	"github.com/onerilhan/go-payment-api/internal/models"
	"github.com/onerilhan/go-payment-api/internal/repository"
)

// TransactionService transaction business logic'i
type TransactionService struct {
	transactionRepo *repository.TransactionRepository
	balanceService  *BalanceService // ← Repository yerine Service
}

// NewTransactionService yeni service oluşturur
func NewTransactionService(transactionRepo *repository.TransactionRepository, balanceService *BalanceService) *TransactionService {
	return &TransactionService{
		transactionRepo: transactionRepo,
		balanceService:  balanceService, // ← Service inject et
	}
}

// ValidateTransactionType transaction type'ını doğrular
func (s *TransactionService) ValidateTransactionType(txType string) error {
	validTypes := map[string]bool{
		"credit":   true,
		"debit":    true,
		"transfer": true,
	}

	if !validTypes[txType] {
		return fmt.Errorf("geçersiz transaction tipi: %s. Geçerli tipler: credit, debit, transfer", txType)
	}

	return nil
}

// ValidateAmount para miktarını doğrular
func (s *TransactionService) ValidateAmount(amount float64) error {
	if amount <= 0 {
		return fmt.Errorf("miktar sıfırdan büyük olmalıdır")
	}

	if amount > 1000000 { // maksimum limit
		return fmt.Errorf("maksimum transfer limiti: 1,000,000 TL")
	}

	return nil
}

// Transfer kullanıcılar arası para transferi yapar
func (s *TransactionService) Transfer(fromUserID int, req *models.TransferRequest) (*models.Transaction, error) {
	// Amount validation
	if err := s.ValidateAmount(req.Amount); err != nil {
		return nil, err
	}

	// Aynı kullanıcıya transfer kontrolü
	if fromUserID == req.ToUserID {
		return nil, fmt.Errorf("kendinize para gönderemezsiniz")
	}

	// Gönderen kullanıcının bakiyesini kontrol et
	fromBalance, err := s.balanceService.GetBalance(fromUserID)
	if err != nil {
		return nil, fmt.Errorf("bakiye alınamadı: %w", err)
	}

	// Yeterli bakiye kontrolü
	if fromBalance.Amount < req.Amount {
		return nil, fmt.Errorf("yetersiz bakiye. Mevcut bakiye: %.2f TL", fromBalance.Amount)
	}

	// Alan kullanıcının bakiyesini al (yoksa oluştur)
	toBalance, err := s.balanceService.GetBalance(req.ToUserID)
	if err != nil {
		return nil, fmt.Errorf("alıcı kullanıcı bakiye hatası: %w", err)
	}

	// Transaction oluştur
	transaction := &models.Transaction{
		FromUserID:  &fromUserID,
		ToUserID:    &req.ToUserID,
		Amount:      req.Amount,
		Type:        "transfer",
		Status:      "completed",
		Description: req.Description,
	}

	// Transaction'ı kaydet
	createdTx, err := s.transactionRepo.Create(transaction)
	if err != nil {
		return nil, fmt.Errorf("transaction oluşturulamadı: %w", err)
	}

	// Bakiyeleri güncelle (BUG DÜZELTME: toBalance.Amount kullan)
	newFromAmount := fromBalance.Amount - req.Amount
	newToAmount := toBalance.Amount + req.Amount

	if err := s.balanceService.UpdateBalance(fromUserID, newFromAmount); err != nil {
		return nil, fmt.Errorf("gönderen bakiye güncellenemedi: %w", err)
	}

	if err := s.balanceService.UpdateBalance(req.ToUserID, newToAmount); err != nil {
		return nil, fmt.Errorf("alan bakiye güncellenemedi: %w", err)
	}

	return createdTx, nil
}

// GetUserTransactions kullanıcının transaction geçmişini getirir
func (s *TransactionService) GetUserTransactions(userID int, limit, offset int) ([]*models.Transaction, error) {
	transactions, err := s.transactionRepo.GetByUserID(userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("transaction geçmişi alınamadı: %w", err)
	}

	return transactions, nil
}
