package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/onerilhan/go-payment-api/internal/interfaces"
	"github.com/onerilhan/go-payment-api/internal/models"
)

// MockTransactionRepository, TransactionRepositoryInterface için sahte (mock) bir yapıdır.
type MockTransactionRepository struct {
	mock.Mock
}

var _ interfaces.TransactionRepositoryInterface = (*MockTransactionRepository)(nil)

func (m *MockTransactionRepository) Create(tx *models.Transaction) (*models.Transaction, error) {
	args := m.Called(tx)
	return args.Get(0).(*models.Transaction), args.Error(1)
}
func (m *MockTransactionRepository) GetByID(id int) (*models.Transaction, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Transaction), args.Error(1)
}
func (m *MockTransactionRepository) GetByUserID(userID, limit, offset int) ([]*models.Transaction, error) {
	args := m.Called(userID, limit, offset)
	return args.Get(0).([]*models.Transaction), args.Error(1)
}
func (m *MockTransactionRepository) GetByStatus(status string, limit, offset int) ([]*models.Transaction, error) {
	args := m.Called(status, limit, offset)
	return args.Get(0).([]*models.Transaction), args.Error(1)
}
func (m *MockTransactionRepository) UpdateStatus(id int, status string) error {
	args := m.Called(id, status)
	return args.Error(0)
}
func (m *MockTransactionRepository) GetUserTransactionStats(userID int) (*models.TransactionStats, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TransactionStats), args.Error(1)
}

// MockBalanceService, BalanceServiceInterface için sahte (mock) bir yapıdır.
type MockBalanceService struct {
	mock.Mock
}

var _ interfaces.BalanceServiceInterface = (*MockBalanceService)(nil)

func (m *MockBalanceService) GetBalance(userID int) (*models.Balance, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Balance), args.Error(1)
}
func (m *MockBalanceService) UpdateBalance(userID int, amount float64) error {
	args := m.Called(userID, amount)
	return args.Error(0)
}
func (m *MockBalanceService) GetBalanceHistory(userID, limit, offset int) ([]*models.BalanceHistory, error) {
	args := m.Called(userID, limit, offset)
	return args.Get(0).([]*models.BalanceHistory), args.Error(1)
}
func (m *MockBalanceService) GetBalanceAtTime(userID int, targetTime string) (*models.BalanceAtTime, error) {
	args := m.Called(userID, targetTime)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BalanceAtTime), args.Error(1)
}
func (m *MockBalanceService) CreateBalanceSnapshot(userID int, amount float64, reason string) error {
	args := m.Called(userID, amount, reason)
	return args.Error(0)
}

// TestTransactionService_GetByID_Success, ID ile transaction getirme senaryosunu test eder.
func TestTransactionService_GetTransactionByID_Success(t *testing.T) {
	// Arrange
	mockTxRepo := new(MockTransactionRepository)
	mockBalanceService := new(MockBalanceService)
	transactionService := NewTransactionService(mockTxRepo, mockBalanceService, nil)

	txID := 1
	fromUserID := 10
	toUserID := 20
	expectedTransaction := &models.Transaction{
		ID:         txID,
		FromUserID: &fromUserID,
		ToUserID:   &toUserID,
		Amount:     150.0,
		Type:       "transfer",
		Status:     models.StatusCompleted,
		CreatedAt:  time.Now(),
	}

	mockTxRepo.On("GetByID", txID).Return(expectedTransaction, nil)

	// Act
	result, err := transactionService.GetTransactionByID(txID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedTransaction, result)
	mockTxRepo.AssertExpectations(t)
}
