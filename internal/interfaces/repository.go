// internal/interfaces/repository.go
package interfaces

import (
	"time"

	"github.com/onerilhan/go-payment-api/internal/models"
)

// UserRepositoryInterface kullanıcı database işlemleri için interface
type UserRepositoryInterface interface {
	// Create yeni kullanıcı oluşturur
	Create(user *models.CreateUserRequest) (*models.User, error)

	// GetByEmail email ile kullanıcı bulur
	GetByEmail(email string) (*models.User, error)

	// GetByID ID ile kullanıcı bulur
	GetByID(id int) (*models.User, error)

	// Update kullanıcı bilgilerini günceller
	Update(id int, user *models.UpdateUserRequest) (*models.User, error)

	// Delete kullanıcıyı siler (soft delete)
	Delete(id int) error

	// GetAll tüm kullanıcıları listeler (pagination ile)
	GetAll(limit, offset int) ([]*models.User, int, error) // users, total_count, error
}

// TransactionRepositoryInterface transaction database işlemleri için interface
type TransactionRepositoryInterface interface {
	// Create yeni transaction oluşturur
	Create(tx *models.Transaction) (*models.Transaction, error)

	// GetByID ID ile transaction getirir
	GetByID(id int) (*models.Transaction, error)

	// GetByUserID kullanıcının transaction'larını getirir
	GetByUserID(userID int, limit, offset int) ([]*models.Transaction, error)

	// GetByStatus belirli status'taki transaction'ları getirir
	GetByStatus(status string, limit, offset int) ([]*models.Transaction, error)

	// UpdateStatus transaction status'unu günceller
	UpdateStatus(id int, status string) error

	// GetUserTransactionStats kullanıcının transaction istatistiklerini getirir
	GetUserTransactionStats(userID int) (*models.TransactionStats, error)
}

// BalanceRepositoryInterface balance database işlemleri için interface
type BalanceRepositoryInterface interface {
	// GetByUserID kullanıcının bakiyesini getirir
	GetByUserID(userID int) (*models.Balance, error)

	// CreateBalance yeni bakiye oluşturur
	CreateBalance(userID int) (*models.Balance, error)

	// UpdateBalance kullanıcının bakiyesini günceller
	UpdateBalance(userID int, newAmount float64) error

	// GetBalanceHistory kullanıcının bakiye geçmişini getirir
	GetBalanceHistory(userID int, limit, offset int) ([]*models.BalanceHistory, error)

	// CreateBalanceSnapshot belirli bir anda bakiye snapshot'ı oluşturur
	CreateBalanceSnapshot(userID int, amount float64, reason string) error

	// Belirli bir zamandaki bakiyeyi getirir.
	GetBalanceAtTime(userID int, atTime time.Time) (*models.BalanceAtTime, error)
}

// AuditRepositoryInterface audit log database işlemleri için interface
type AuditRepositoryInterface interface {
	// Create yeni audit log oluşturur
	Create(log *models.AuditLog) error

	// GetByEntity belirli entity'nin audit loglarını getirir
	GetByEntity(entityType string, entityID int, limit, offset int) ([]*models.AuditLog, error)

	// GetByUser kullanıcının yaptığı tüm işlemleri getirir
	GetByUser(userID int, limit, offset int) ([]*models.AuditLog, error)

	// GetByDateRange belirli tarih aralığındaki logları getirir
	GetByDateRange(startDate, endDate string, limit, offset int) ([]*models.AuditLog, error)
}
