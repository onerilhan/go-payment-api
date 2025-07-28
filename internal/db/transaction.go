package db

import (
	"database/sql"
	"fmt"

	"github.com/rs/zerolog/log"
)

// TransactionFunc database transaction içinde çalışacak fonksiyon tipi
type TransactionFunc func(tx *sql.Tx) error

// WithTransaction database transaction'ı yönetir
// Hata durumunda otomatik rollback, başarı durumunda commit yapar
func WithTransaction(db *sql.DB, fn TransactionFunc) error {
	// Transaction başlat
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("transaction başlatılamadı: %w", err)
	}

	// Defer ile transaction'ı yönet
	defer func() {
		if r := recover(); r != nil {
			// Panic durumunda rollback
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Error().Err(rollbackErr).Msg("Rollback hatası (panic)")
			}
			log.Error().Interface("panic", r).Msg("Transaction panic ile rollback yapıldı")
			panic(r) // Panic'i yeniden fırlat
		}
	}()

	// İş mantığını çalıştır
	if err := fn(tx); err != nil {
		// Hata durumunda rollback
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			log.Error().Err(rollbackErr).Msg("Rollback hatası")
			return fmt.Errorf("transaction hatası ve rollback hatası: %w, rollback: %v", err, rollbackErr)
		}
		log.Warn().Err(err).Msg("Transaction rollback yapıldı")
		return err
	}

	// Başarı durumunda commit
	if err := tx.Commit(); err != nil {
		log.Error().Err(err).Msg("Commit hatası")
		return fmt.Errorf("transaction commit hatası: %w", err)
	}

	log.Debug().Msg("Transaction başarıyla commit edildi")
	return nil
}

// TransactionRepository transaction içinde repository işlemleri için helper
type TransactionRepository struct {
	tx *sql.Tx
}

// NewTransactionRepository transaction-aware repository oluşturur
func NewTransactionRepository(tx *sql.Tx) *TransactionRepository {
	return &TransactionRepository{tx: tx}
}

// Exec transaction içinde SQL çalıştırır
func (tr *TransactionRepository) Exec(query string, args ...interface{}) (sql.Result, error) {
	return tr.tx.Exec(query, args...)
}

// QueryRow transaction içinde tek satır sorgusu çalıştırır
func (tr *TransactionRepository) QueryRow(query string, args ...interface{}) *sql.Row {
	return tr.tx.QueryRow(query, args...)
}

// Query transaction içinde çoklu satır sorgusu çalıştırır
func (tr *TransactionRepository) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return tr.tx.Query(query, args...)
}
