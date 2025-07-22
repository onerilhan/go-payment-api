package repository

import (
	"database/sql"
	"fmt"

	"github.com/onerilhan/go-payment-api/internal/models"
)

// BalanceRepository balance database işlemleri
type BalanceRepository struct {
	db *sql.DB
}

// NewBalanceRepository yeni repository oluşturur
func NewBalanceRepository(db *sql.DB) *BalanceRepository {
	return &BalanceRepository{db: db}
}

// GetByUserID kullanıcının bakiyesini getirir
func (r *BalanceRepository) GetByUserID(userID int) (*models.Balance, error) {
	query := `
		SELECT user_id, amount, last_updated_at
		FROM balances 
		WHERE user_id = $1
	`

	var balance models.Balance
	err := r.db.QueryRow(query, userID).Scan(
		&balance.UserID,
		&balance.Amount,
		&balance.LastUpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// Bakiye yoksa sıfır bakiye oluştur
			return r.CreateBalance(userID)
		}
		return nil, fmt.Errorf("bakiye arama hatası: %w", err)
	}

	return &balance, nil
}

// CreateBalance yeni bakiye oluşturur
func (r *BalanceRepository) CreateBalance(userID int) (*models.Balance, error) {
	query := `
		INSERT INTO balances (user_id, amount) 
		VALUES ($1, 0.00) 
		RETURNING user_id, amount, last_updated_at
	`

	var balance models.Balance
	err := r.db.QueryRow(query, userID).Scan(
		&balance.UserID,
		&balance.Amount,
		&balance.LastUpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("bakiye oluşturulamadı: %w", err)
	}

	return &balance, nil
}

// UpdateBalance kullanıcının bakiyesini günceller
func (r *BalanceRepository) UpdateBalance(userID int, newAmount float64) error {
	query := `
		UPDATE balances 
		SET amount = $1
		WHERE user_id = $2
	`

	_, err := r.db.Exec(query, newAmount, userID)
	if err != nil {
		return fmt.Errorf("bakiye güncellenemedi: %w", err)
	}

	return nil
}
