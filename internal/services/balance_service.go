package services

import (
	"fmt"
	"sync"
	"time"

	"github.com/onerilhan/go-payment-api/internal/interfaces"
	"github.com/onerilhan/go-payment-api/internal/models"
)

// BalanceService, bakiye işlemlerini thread-safe (aynı anda birden fazla işlem için güvenli) bir şekilde yönetir.
type BalanceService struct {
	balanceRepo interfaces.BalanceRepositoryInterface
	mutex       sync.RWMutex // Thread-safe operations için
}

// NewBalanceService, yeni bir service oluşturur.
func NewBalanceService(balanceRepo interfaces.BalanceRepositoryInterface) *BalanceService {
	return &BalanceService{
		balanceRepo: balanceRepo,
	}
}

var _ interfaces.BalanceServiceInterface = (*BalanceService)(nil)

// GetBalance, kullanıcının mevcut bakiyesini getirir.
func (s *BalanceService) GetBalance(userID int) (*models.Balance, error) {
	s.mutex.RLock() // Okuma kilidi
	defer s.mutex.RUnlock()

	balance, err := s.balanceRepo.GetByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("bakiye alınamadı: %w", err)
	}
	return balance, nil
}

// UpdateBalance, kullanıcının bakiyesini günceller.
func (s *BalanceService) UpdateBalance(userID int, amount float64) error {
	s.mutex.Lock() // Yazma kilidi
	defer s.mutex.Unlock()

	return s.balanceRepo.UpdateBalance(userID, amount)
}

// GetBalanceHistory, kullanıcının bakiye geçmişini listeler.
func (s *BalanceService) GetBalanceHistory(userID int, limit, offset int) ([]*models.BalanceHistory, error) {
	s.mutex.RLock() // Okuma kilidi
	defer s.mutex.RUnlock()

	// Pagination validasyonu
	if limit <= 0 || limit > 100 {
		limit = 10 // default limit
	}
	if offset < 0 {
		offset = 0 // default offset
	}

	history, err := s.balanceRepo.GetBalanceHistory(userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("bakiye geçmişi alınamadı: %w", err)
	}
	return history, nil
}

// GetBalanceAtTime, belirli bir tarihte kullanıcının bakiyesini hesaplar.
func (s *BalanceService) GetBalanceAtTime(userID int, targetTime string) (*models.BalanceAtTime, error) {
	s.mutex.RLock() // Okuma kilidi
	defer s.mutex.RUnlock()

	// Tarih formatını parse et (ISO 8601)
	parsedTime, err := time.Parse("2006-01-02T15:04:05Z", targetTime)
	if err != nil {
		return nil, fmt.Errorf("geçersiz tarih formatı. Format: 2006-01-02T15:04:05Z")
	}

	balance, err := s.balanceRepo.GetBalanceAtTime(userID, parsedTime)
	if err != nil {
		return nil, fmt.Errorf("belirli tarihteki bakiye hesaplanamadı: %w", err)
	}
	return balance, nil
}

// CreateBalanceSnapshot, bir bakiye anlık görüntüsü oluşturur.
func (s *BalanceService) CreateBalanceSnapshot(userID int, amount float64, reason string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	err := s.balanceRepo.CreateBalanceSnapshot(userID, amount, reason)
	if err != nil {
		return fmt.Errorf("servis katmanında bakiye anlık görüntüsü oluşturulamadı: %w", err)
	}
	return nil
}
