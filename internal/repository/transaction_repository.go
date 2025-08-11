package repository

import (
	"database/sql"
	"fmt"

	"github.com/onerilhan/go-payment-api/internal/interfaces"
	"github.com/onerilhan/go-payment-api/internal/models"
)

// TransactionRepository, TransactionRepositoryInterface'in somut halidir.
type TransactionRepository struct {
	db *sql.DB
}

// NewTransactionRepository, yeni bir repository oluşturur ve arayüz olarak döndürür.
func NewTransactionRepository(db *sql.DB) interfaces.TransactionRepositoryInterface {
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

// GetByStatus, belirli bir durumdaki transaction'ları getirir
func (r *TransactionRepository) GetByStatus(status string, limit, offset int) ([]*models.Transaction, error) {
	query := `
		SELECT id, from_user_id, to_user_id, amount, type, status, description, created_at
		FROM transactions 
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(query, status, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("transaction listesi (status'a göre) alınamadı: %w", err)
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

// UpdateStatus, bir transaction'ın durumunu günceller
func (r *TransactionRepository) UpdateStatus(id int, status string) error {
	query := `UPDATE transactions SET status = $1 WHERE id = $2`

	result, err := r.db.Exec(query, status, id)
	if err != nil {
		return fmt.Errorf("transaction status güncellenemedi: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("güncelleme sonucu kontrol edilemedi: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("güncellenecek transaction bulunamadı (ID: %d)", id)
	}

	return nil
}

// GetUserTransactionStats, bir kullanıcının işlem istatistiklerini hesaplar
func (r *TransactionRepository) GetUserTransactionStats(userID int) (*models.TransactionStats, error) {
	// Bu sorgu, senin TransactionStats modelindeki tüm alanları dolduracak şekilde güncellendi.
	query := `
		SELECT
			-- Toplam işlem sayısı
			COUNT(*) AS total_transactions,
			-- Farklı tiplerdeki işlem sayıları (PostgreSQL'e özel FILTER kullanımı)
			COUNT(*) FILTER (WHERE type = 'credit') AS total_credits,
			COUNT(*) FILTER (WHERE type = 'debit') AS total_debits,
			COUNT(*) FILTER (WHERE type = 'transfer') AS total_transfers,
			-- Farklı tiplerdeki işlem tutarlarının toplamı
			COALESCE(SUM(amount) FILTER (WHERE type = 'credit'), 0) AS total_credit_amount,
			COALESCE(SUM(amount) FILTER (WHERE type = 'debit'), 0) AS total_debit_amount,
			COALESCE(SUM(amount) FILTER (WHERE type = 'transfer'), 0) AS total_transfer_amount,
			-- Son işlem tarihi (string formatında)
			TO_CHAR(MAX(created_at), 'YYYY-MM-DD"T"HH24:MI:SS"Z"') as last_transaction_date
		FROM
			transactions
		WHERE
			from_user_id = $1 OR to_user_id = $1
	`
	var stats models.TransactionStats
	stats.UserID = userID // UserID'yi manuel olarak atıyoruz, çünkü sorgudan dönmüyor.

	err := r.db.QueryRow(query, userID).Scan(
		&stats.TotalTransactions,
		&stats.TotalCredits,
		&stats.TotalDebits,
		&stats.TotalTransfers,
		&stats.TotalCreditAmount,
		&stats.TotalDebitAmount,
		&stats.TotalTransferAmount,
		&stats.LastTransactionDate,
	)

	if err != nil {
		// Eğer hiç işlem yoksa Scan hata verebilir ama bu bir sorun değil.
		// Bu durumu kontrol edebilir veya stats'ın sıfır değerlerini dönebiliriz.
		// Şimdilik, hata varsa direkt dönelim.
		return nil, fmt.Errorf("kullanıcı işlem istatistikleri alınamadı: %w", err)
	}

	return &stats, nil
}
