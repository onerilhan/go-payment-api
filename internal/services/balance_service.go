package services

import (
	"fmt"
	"sync"
	"time"

	"github.com/onerilhan/go-payment-api/internal/models"
	"github.com/onerilhan/go-payment-api/internal/repository"
)

// BalanceService thread-safe balance operations
type BalanceService struct {
	balanceRepo *repository.BalanceRepository
	mutex       sync.RWMutex // Thread-safe operations için
}

// NewBalanceService yeni service oluşturur
func NewBalanceService(balanceRepo *repository.BalanceRepository) *BalanceService {
	return &BalanceService{
		balanceRepo: balanceRepo,
		mutex:       sync.RWMutex{},
	}
}

// GetBalance thread-safe balance okuma
func (s *BalanceService) GetBalance(userID int) (*models.Balance, error) {
	s.mutex.RLock()         // Read lock
	defer s.mutex.RUnlock() // Release when done

	return s.balanceRepo.GetByUserID(userID)
}

// UpdateBalance thread-safe balance güncelleme
func (s *BalanceService) UpdateBalance(userID int, amount float64) error {
	s.mutex.Lock()         // Write lock
	defer s.mutex.Unlock() // Release when done

	return s.balanceRepo.UpdateBalance(userID, amount)
}

// GetBalanceHistory kullanıcının bakiye geçmişini getirir
func (s *BalanceService) GetBalanceHistory(userID int, limit, offset int) ([]*models.BalanceHistory, error) {
	s.mutex.RLock()         // Read lock
	defer s.mutex.RUnlock() // Release when done

	// Pagination validation
	if limit <= 0 || limit > 100 {
		limit = 10 // default limit
	}
	if offset < 0 {
		offset = 0 // default offset
	}

	// Repository'den bakiye geçmişini al
	history, err := s.balanceRepo.GetBalanceHistory(userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("bakiye geçmişi alınamadı: %w", err)
	}

	return history, nil
}

// GetBalanceAtTime belirli bir tarihte kullanıcının bakiyesini hesaplar
func (s *BalanceService) GetBalanceAtTime(userID int, targetTime string) (*models.BalanceAtTime, error) {
	s.mutex.RLock()         // Read lock
	defer s.mutex.RUnlock() // Release when done

	// Tarih formatını parse et (ISO 8601: 2025-07-28T15:30:00Z)
	parsedTime, err := time.Parse("2006-01-02T15:04:05Z", targetTime)
	if err != nil {
		return nil, fmt.Errorf("geçersiz tarih formatı. Format: 2006-01-02T15:04:05Z")
	}

	// Repository'den o tarihteki bakiyeyi hesapla
	balance, err := s.balanceRepo.GetBalanceAtTime(userID, parsedTime)
	if err != nil {
		return nil, fmt.Errorf("belirli tarihteki bakiye hesaplanamadı: %w", err)
	}

	return balance, nil
}
