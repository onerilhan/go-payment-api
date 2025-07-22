package repository

import (
	"database/sql"
	"fmt"

	"github.com/onerilhan/go-payment-api/internal/models"
)

// UserRepository kullanıcı database işlemleri
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository yeni repository oluşturur
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create yeni kullanıcı oluşturur
func (r *UserRepository) Create(user *models.CreateUserRequest) (*models.User, error) {
	query := `
		INSERT INTO users (name, email, password) 
		VALUES ($1, $2, $3) 
		RETURNING id, name, email, created_at
	`

	var result models.User
	err := r.db.QueryRow(query, user.Name, user.Email, user.Password).Scan(
		&result.ID,
		&result.Name,
		&result.Email,
		&result.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("kullanıcı oluşturulamadı: %w", err)
	}

	return &result, nil
}

// GetByEmail email ile kullanıcı bulur
func (r *UserRepository) GetByEmail(email string) (*models.User, error) {
	query := `
		SELECT id, name, email, password, created_at 
		FROM users 
		WHERE email = $1
	`

	var user models.User
	err := r.db.QueryRow(query, email).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Password,
		&user.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("kullanıcı bulunamadı")
		}
		return nil, fmt.Errorf("kullanıcı arama hatası: %w", err)
	}

	return &user, nil
}

// GetByID ID ile kullanıcı bulur
func (r *UserRepository) GetByID(id int) (*models.User, error) {
	query := `
		SELECT id, name, email, created_at 
		FROM users 
		WHERE id = $1
	`

	var user models.User
	err := r.db.QueryRow(query, id).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("kullanıcı bulunamadı")
		}
		return nil, fmt.Errorf("kullanıcı arama hatası: %w", err)
	}

	return &user, nil
}
