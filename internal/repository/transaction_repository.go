package repository

import (
	"database/sql"
	"fmt"

	"github.com/onerilhan/go-payment-api/internal/models"
)

// TransactionRepository transaction database işlemleri
type TransactionRepository struct {
	db *sql.DB
}

// NewTransactionRepository yeni repository oluşturur
func NewTransactionRepository(db *sql.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

// Create yeni transaction oluşturur
func (r *TransactionRepository) Create(tx *models.Transaction) (*models.Transaction, error) {
	query := `
		INSERT INTO transactions (from_user_id, to_user_id, amount, type, status, description) 
		VALUES ($1, $2, $3, $4, $5, $6) 
		RETURNING id, created_at
	`

	err := r.db.QueryRow(
		query,
		tx.FromUserID,
		tx.ToUserID,
		tx.Amount,
		tx.Type,
		tx.Status,
		tx.Description,
	).Scan(&tx.ID, &tx.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("transaction oluşturulamadı: %w", err)
	}

	return tx, nil
}

// GetByID ID ile transaction getirir
func (r *TransactionRepository) GetByID(id int) (*models.Transaction, error) {
	query := `
		SELECT id, from_user_id, to_user_id, amount, type, status, description, created_at
		FROM transactions 
		WHERE id = $1
	`

	var tx models.Transaction
	err := r.db.QueryRow(query, id).Scan(
		&tx.ID,
		&tx.FromUserID,
		&tx.ToUserID,
		&tx.Amount,
		&tx.Type,
		&tx.Status,
		&tx.Description,
		&tx.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("transaction bulunamadı")
		}
		return nil, fmt.Errorf("transaction arama hatası: %w", err)
	}

	return &tx, nil
}

// GetByUserID kullanıcının transaction'larını getirir
func (r *TransactionRepository) GetByUserID(userID int, limit, offset int) ([]*models.Transaction, error) {
	query := `
		SELECT id, from_user_id, to_user_id, amount, type, status, description, created_at
		FROM transactions 
		WHERE from_user_id = $1 OR to_user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("transaction listesi alınamadı: %w", err)
	}
	defer rows.Close()

	var transactions []*models.Transaction
	for rows.Next() {
		var tx models.Transaction
		err := rows.Scan(
			&tx.ID,
			&tx.FromUserID,
			&tx.ToUserID,
			&tx.Amount,
			&tx.Type,
			&tx.Status,
			&tx.Description,
			&tx.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("transaction scan hatası: %w", err)
		}
		transactions = append(transactions, &tx)
	}

	return transactions, nil
}
