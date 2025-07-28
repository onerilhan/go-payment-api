// internal/interfaces/service.go
package interfaces

import "github.com/onerilhan/go-payment-api/internal/models"

// UserServiceInterface kullanıcı business logic için interface
type UserServiceInterface interface {
	// Register yeni kullanıcı kaydeder
	Register(req *models.CreateUserRequest) (*models.User, error)

	// Login kullanıcı girişi yapar ve token döner
	Login(req *models.LoginRequest) (*models.LoginResponse, error)

	// GetUserByID ID ile kullanıcı getirir
	GetUserByID(userID int) (*models.User, error)

	// UpdateUser kullanıcı bilgilerini günceller
	UpdateUser(userID int, req *models.UpdateUserRequest) (*models.User, error)

	// DeleteUser kullanıcıyı siler (soft delete)
	DeleteUser(userID int) error

	// GetAllUsers tüm kullanıcıları listeler
	GetAllUsers(limit, offset int) ([]*models.User, int, error)
}

// TransactionServiceInterface transaction business logic için interface
type TransactionServiceInterface interface {
	// Transfer kullanıcılar arası para transferi yapar
	Transfer(fromUserID int, req *models.TransferRequest) (*models.Transaction, error)

	// Credit kullanıcının hesabına para yatırır
	Credit(userID int, req *models.CreditRequest) (*models.Transaction, error)

	// Debit kullanıcının hesabından para çeker
	Debit(userID int, req *models.DebitRequest) (*models.Transaction, error)

	// GetUserTransactions kullanıcının transaction geçmişini getirir
	GetUserTransactions(userID int, limit, offset int) ([]*models.Transaction, error)

	// GetTransactionByID ID ile transaction getirir
	GetTransactionByID(id int) (*models.Transaction, error)

	// GetTransactionStats kullanıcının transaction istatistiklerini getirir
	GetTransactionStats(userID int) (*models.TransactionStats, error)

	// ValidateTransactionType transaction type'ını doğrular
	ValidateTransactionType(txType string) error

	// ValidateAmount para miktarını doğrular
	ValidateAmount(amount float64) error
}

// BalanceServiceInterface balance business logic için interface
type BalanceServiceInterface interface {
	// GetBalance thread-safe balance okuma
	GetBalance(userID int) (*models.Balance, error)

	// UpdateBalance thread-safe balance güncelleme
	UpdateBalance(userID int, amount float64) error

	// GetBalanceHistory kullanıcının bakiye geçmişini getirir
	GetBalanceHistory(userID int, limit, offset int) ([]*models.BalanceHistory, error)

	// CreateBalanceSnapshot belirli bir anda bakiye snapshot'ı oluşturur
	CreateBalanceSnapshot(userID int, amount float64, reason string) error
}
