package models

import (
	"time"
)

// User kullanıcı modelini temsil eder
type User struct {
	ID        int       `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Email     string    `json:"email" db:"email"`
	Password  string    `json:"-" db:"password"` // JSON'da gösterilmez
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// CreateUserRequest kullanıcı oluşturma isteği
type CreateUserRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest giriş isteği
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse giriş yanıtı
type LoginResponse struct {
	User  *User  `json:"user"`
	Token string `json:"token"`
}

// RefreshResponse token refresh yanıtı
type RefreshResponse struct {
	Success   bool   `json:"success"`
	Token     string `json:"token"`
	ExpiresIn int64  `json:"expires_in"`
	Message   string `json:"message"`
}
