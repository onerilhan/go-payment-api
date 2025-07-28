package repository

import (
	"database/sql"
	"fmt"
	"time"

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

// GetBalanceHistory kullanıcının bakiye geçmişini getirir
func (r *BalanceRepository) GetBalanceHistory(userID int, limit, offset int) ([]*models.BalanceHistory, error) {
	query := `
		SELECT id, user_id, previous_amount, new_amount, change_amount, reason, transaction_id, created_at
		FROM balance_history 
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("bakiye geçmişi sorgusu hatası: %w", err)
	}
	defer rows.Close()

	var history []*models.BalanceHistory
	for rows.Next() {
		var h models.BalanceHistory
		err := rows.Scan(
			&h.ID,
			&h.UserID,
			&h.PreviousAmount,
			&h.NewAmount,
			&h.ChangeAmount,
			&h.Reason,
			&h.TransactionID,
			&h.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("bakiye geçmişi scan hatası: %w", err)
		}
		history = append(history, &h)
	}

	return history, nil
}

// GetBalanceAtTime belirli bir tarihte kullanıcının bakiyesini hesaplar
func (r *BalanceRepository) GetBalanceAtTime(userID int, targetTime time.Time) (*models.BalanceAtTime, error) {
	// O tarihe kadar olan tüm balance değişikliklerini topla
	query := `
		SELECT COALESCE(SUM(change_amount), 0) as total_change
		FROM balance_history 
		WHERE user_id = $1 AND created_at <= $2
	`

	var totalChange float64
	err := r.db.QueryRow(query, userID, targetTime).Scan(&totalChange)
	if err != nil {
		return nil, fmt.Errorf("bakiye hesaplama hatası: %w", err)
	}

	// Kullanıcının ilk bakiyesi genelde 0, sonra change_amount'ları topla
	// Not: Eğer başlangıç bakiyesi 0 değilse, bu query'i güncelle
	finalAmount := totalChange

	// Negatif bakiye olmasın
	if finalAmount < 0 {
		finalAmount = 0
	}

	result := &models.BalanceAtTime{
		UserID:  userID,
		Amount:  finalAmount,
		AtTime:  targetTime.Format("2006-01-02T15:04:05Z"),
		Message: fmt.Sprintf("Bakiye %s tarihinde hesaplandı", targetTime.Format("2006-01-02 15:04:05")),
	}

	return result, nil
}
