package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

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
		INSERT INTO users (name, email, password, role) 
		VALUES ($1, $2, $3, $4) 
		RETURNING id, name, email, role, created_at
	`

	var result models.User
	err := r.db.QueryRow(query, user.Name, user.Email, user.Password, user.Role).Scan(
		&result.ID,
		&result.Name,
		&result.Email,
		&result.Role,
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
		SELECT id, name, email, password, role, created_at 
		FROM users 
		WHERE email = $1 AND deleted_at IS NULL
	`

	var user models.User
	err := r.db.QueryRow(query, email).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Password,
		&user.Role,
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
		SELECT id, name, email, role, created_at 
		FROM users 
		WHERE id = $1 AND deleted_at IS NULL
	`

	var user models.User
	err := r.db.QueryRow(query, id).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Role,
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

// Update kullanıcı bilgilerini günceller
func (r *UserRepository) Update(id int, req *models.UpdateUserRequest) (*models.User, error) {
	// Dynamic query building - sadece gönderilen fieldlar güncellenecek
	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	// Name güncellenmeli mi?
	if req.Name != nil {
		setParts = append(setParts, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, *req.Name)
		argIndex++
	}

	// Email güncellenmeli mi?
	if req.Email != nil {
		setParts = append(setParts, fmt.Sprintf("email = $%d", argIndex))
		args = append(args, *req.Email)
		argIndex++
	}

	// Password güncellenmeli mi?
	if req.Password != nil {
		// Şifreyi hashle
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("şifre hashlenemedi: %w", err)
		}
		setParts = append(setParts, fmt.Sprintf("password = $%d", argIndex))
		args = append(args, string(hashedPassword))
		argIndex++
	}

	// Role güncellenmeli mi?
	if req.Role != nil {
		setParts = append(setParts, fmt.Sprintf("role = $%d", argIndex))
		args = append(args, *req.Role)
		argIndex++
	}

	// Hiçbir field gönderilmemişse hata
	if len(setParts) == 0 {
		return nil, fmt.Errorf("güncellenecek alan bulunamadı")
	}

	// Updated_at fieldını ekle (eğer updated_at column'u varsa)
	setParts = append(setParts, fmt.Sprintf("updated_at = $%d", argIndex))
	args = append(args, time.Now())
	argIndex++

	// ID'yi en sona ekle
	args = append(args, id)

	// Query'yi oluştur
	query := fmt.Sprintf(`
		UPDATE users 
		SET %s
		WHERE id = $%d AND deleted_at IS NULL
		RETURNING id, name, email, role, created_at
	`, strings.Join(setParts, ", "), argIndex)

	// Query'yi çalıştır
	var user models.User
	err := r.db.QueryRow(query, args...).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Role,
		&user.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("kullanıcı bulunamadı")
		}
		return nil, fmt.Errorf("kullanıcı güncellenemedi: %w", err)
	}

	return &user, nil
}

// Delete kullanıcıyı siler (soft delete)
func (r *UserRepository) Delete(id int) error {
	query := `
		UPDATE users 
		SET deleted_at = $1
		WHERE id = $2 AND deleted_at IS NULL
	`

	result, err := r.db.Exec(query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("kullanıcı silinemedi: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("silme sonucu kontrol edilemedi: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("kullanıcı bulunamadı veya zaten silinmiş")
	}

	return nil
}

// GetAll tüm kullanıcıları listeler (pagination ile)
func (r *UserRepository) GetAll(limit, offset int) ([]*models.User, int, error) {
	// Toplam sayıyı al
	countQuery := `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`
	var totalCount int
	err := r.db.QueryRow(countQuery).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("kullanıcı sayısı alınamadı: %w", err)
	}

	// Kullanıcıları al
	query := `
		SELECT id, name, email, role, created_at
		FROM users 
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("kullanıcı listesi alınamadı: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(
			&user.ID,
			&user.Name,
			&user.Email,
			&user.Role,
			&user.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("kullanıcı scan hatası: %w", err)
		}
		users = append(users, &user)
	}

	return users, totalCount, nil
}
