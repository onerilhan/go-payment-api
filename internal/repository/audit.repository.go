package repository

import (
	"database/sql"
	"fmt"

	"github.com/onerilhan/go-payment-api/internal/models"
)

// AuditRepository audit log database işlemleri
type AuditRepository struct {
	db *sql.DB
}

// NewAuditRepository yeni repository oluşturur
func NewAuditRepository(db *sql.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

// Create yeni audit log oluşturur
func (r *AuditRepository) Create(log *models.AuditLog) error {
	query := `
		INSERT INTO audit_logs (entity_type, entity_id, action, user_id, old_data, new_data, details, ip_address, user_agent) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.Exec(
		query,
		log.EntityType,
		log.EntityID,
		log.Action,
		log.UserID,
		log.OldData,
		log.NewData,
		log.Details,
		log.IPAddress,
		log.UserAgent,
	)

	if err != nil {
		return fmt.Errorf("audit log oluşturulamadı: %w", err)
	}

	return nil
}
