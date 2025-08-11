package services

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/onerilhan/go-payment-api/internal/interfaces"
	"github.com/onerilhan/go-payment-api/internal/models"
)

// MockBalanceRepository, BalanceRepositoryInterface için sahte (mock) bir yapıdır.
type MockBalanceRepository struct {
	mock.Mock
}

var _ interfaces.BalanceRepositoryInterface = (*MockBalanceRepository)(nil)

func (m *MockBalanceRepository) GetByUserID(userID int) (*models.Balance, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Balance), args.Error(1)
}

func (m *MockBalanceRepository) CreateBalance(userID int) (*models.Balance, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Balance), args.Error(1)
}

func (m *MockBalanceRepository) UpdateBalance(userID int, newAmount float64) error {
	args := m.Called(userID, newAmount)
	return args.Error(0)
}

func (m *MockBalanceRepository) GetBalanceHistory(userID int, limit, offset int) ([]*models.BalanceHistory, error) {
	args := m.Called(userID, limit, offset)
	return args.Get(0).([]*models.BalanceHistory), args.Error(1)
}

func (m *MockBalanceRepository) CreateBalanceSnapshot(userID int, amount float64, reason string) error {
	args := m.Called(userID, amount, reason)
	return args.Error(0)
}

func (m *MockBalanceRepository) GetBalanceAtTime(userID int, atTime time.Time) (*models.BalanceAtTime, error) {
	args := m.Called(userID, atTime)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BalanceAtTime), args.Error(1)
}

// TestBalanceService_GetBalance_Success, bakiye getirme işleminin başarılı senaryosunu test eder.
func TestBalanceService_GetBalance_Success(t *testing.T) {
	// Arrange
	mockBalanceRepo := new(MockBalanceRepository)
	balanceService := NewBalanceService(mockBalanceRepo)

	userID := 1
	expectedBalance := &models.Balance{
		UserID: userID,
		Amount: 500.0,
	}

	mockBalanceRepo.On("GetByUserID", userID).Return(expectedBalance, nil)

	// Act
	result, err := balanceService.GetBalance(userID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedBalance, result)
	mockBalanceRepo.AssertExpectations(t)
}

// TestBalanceService_GetBalance_Error, bakiye getirme işleminde hata senaryosunu test eder.
func TestBalanceService_GetBalance_Error(t *testing.T) {
	// Arrange
	mockBalanceRepo := new(MockBalanceRepository)
	balanceService := NewBalanceService(mockBalanceRepo)

	userID := 1
	mockBalanceRepo.On("GetByUserID", userID).Return(nil, errors.New("veritabanı hatası"))

	// Act
	result, err := balanceService.GetBalance(userID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	mockBalanceRepo.AssertExpectations(t)
}
