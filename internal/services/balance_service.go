package services

import (
	"sync"

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
