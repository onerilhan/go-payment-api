package models

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"
)

// User kullanıcı modelini temsil eder
type User struct {
	ID        int       `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Email     string    `json:"email" db:"email"`
	Password  string    `json:"-" db:"password"` // JSON'da gösterilmez
	Role      string    `json:"role" db:"role"`  // YENİ: Role alanı eklendi
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// CreateUserRequest kullanıcı oluşturma isteği
type CreateUserRequest struct {
	Name            string `json:"name"`
	Email           string `json:"email"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"` // YENİ: Şifre tekrarı
	Role            string `json:"role,omitempty"`   // YENİ: Role opsiyonel
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

// UpdateUserRequest kullanıcı güncelleme isteği
type UpdateUserRequest struct {
	Name     *string `json:"name,omitempty"`     // Pointer kullandık çünkü optional
	Email    *string `json:"email,omitempty"`    // nil = değiştirilmeyecek
	Password *string `json:"password,omitempty"` // empty string ≠ nil
	Role     *string `json:"role,omitempty"`     // YENİ: Role güncelleme
}

// ========== USER VALIDATION METHODS ==========

// Validate User struct'ının tüm alanlarını doğrular
func (u *User) Validate() error {
	if err := u.ValidateName(); err != nil {
		return err
	}

	if err := u.ValidateEmail(); err != nil {
		return err
	}

	if err := u.ValidateRole(); err != nil {
		return err
	}

	return nil
}

// ValidateName kullanıcı adını doğrular
func (u *User) ValidateName() error {
	// Boş kontrol
	if strings.TrimSpace(u.Name) == "" {
		return fmt.Errorf("kullanıcı adı boş olamaz")
	}

	// Uzunluk kontrol
	if len(u.Name) < 2 {
		return fmt.Errorf("kullanıcı adı en az 2 karakter olmalı")
	}

	if len(u.Name) > 50 {
		return fmt.Errorf("kullanıcı adı en fazla 50 karakter olabilir")
	}

	// Sadece harf, rakam, boşluk ve temel karakterler
	validNameRegex := regexp.MustCompile(`^[a-zA-ZğüşıöçĞÜŞİÖÇ\s\.\-\_]+$`)
	if !validNameRegex.MatchString(u.Name) {
		return fmt.Errorf("kullanıcı adı sadece harf, rakam ve temel karakterler içerebilir")
	}

	return nil
}

// ValidateEmail email formatını doğrular
func (u *User) ValidateEmail() error {
	// Boş kontrol
	if strings.TrimSpace(u.Email) == "" {
		return fmt.Errorf("email adresi boş olamaz")
	}

	// Email format kontrolü (basit regex)
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(u.Email) {
		return fmt.Errorf("geçersiz email formatı")
	}

	// Uzunluk kontrol
	if len(u.Email) > 100 {
		return fmt.Errorf("email adresi en fazla 100 karakter olabilir")
	}

	// Küçük harfe çevir (normalize)
	u.Email = strings.ToLower(strings.TrimSpace(u.Email))

	return nil
}

// ValidateRole kullanıcı rolünü doğrular
func (u *User) ValidateRole() error {
	// Geçerli roller
	validRoles := map[string]bool{
		"user":  true,
		"admin": true,
		"mod":   true, // moderator
	}

	// Boşsa default role ver
	if u.Role == "" {
		u.Role = "user"
		return nil
	}

	// Role kontrolü
	if !validRoles[strings.ToLower(u.Role)] {
		return fmt.Errorf("geçersiz rol: %s. Geçerli roller: user, admin, mod", u.Role)
	}

	// Küçük harfe çevir (normalize)
	u.Role = strings.ToLower(u.Role)

	return nil
}

// HasRole belirli bir role sahip mi kontrol eder
func (u *User) HasRole(role string) bool {
	return strings.EqualFold(u.Role, role)
}

// IsAdmin admin rolünde mi kontrol eder
func (u *User) IsAdmin() bool {
	return u.HasRole("admin")
}

// IsMod moderator rolünde mi kontrol eder
func (u *User) IsMod() bool {
	return u.HasRole("mod")
}

// CanModify başka bir kullanıcıyı modify edebilir mi
func (u *User) CanModify(targetUser *User) bool {
	// Admin herşeyi yapabilir
	if u.IsAdmin() {
		return true
	}

	// Mod sadece user'ları modify edebilir
	if u.IsMod() && targetUser.HasRole("user") {
		return true
	}

	// Sadece kendini modify edebilir
	return u.ID == targetUser.ID
}

// ========== REQUEST VALIDATION METHODS ==========

// Validate CreateUserRequest'i doğrular
func (req *CreateUserRequest) Validate() error {
	// Name kontrolü
	if err := req.ValidateName(); err != nil {
		return err
	}

	// Email kontrolü
	if err := req.ValidateEmail(); err != nil {
		return err
	}

	// Password kontrolü
	if err := req.ValidatePassword(); err != nil {
		return err
	}

	// Role kontrolü
	if err := req.ValidateRole(); err != nil {
		return err
	}

	return nil
}

// ValidateName CreateUserRequest name'ini doğrular
func (req *CreateUserRequest) ValidateName() error {
	// Boş kontrol
	if strings.TrimSpace(req.Name) == "" {
		return fmt.Errorf("kullanıcı adı boş olamaz")
	}

	// Uzunluk kontrol
	if len(req.Name) < 2 {
		return fmt.Errorf("kullanıcı adı en az 2 karakter olmalı")
	}

	if len(req.Name) > 50 {
		return fmt.Errorf("kullanıcı adı en fazla 50 karakter olabilir")
	}

	// Sadece harf, rakam, boşluk ve temel karakterler
	validNameRegex := regexp.MustCompile(`^[a-zA-ZğüşıöçĞÜŞİÖÇ\s\.\-\_]+$`)
	if !validNameRegex.MatchString(req.Name) {
		return fmt.Errorf("kullanıcı adı sadece harf, rakam ve temel karakterler içerebilir")
	}

	// Normalize et
	req.Name = strings.TrimSpace(req.Name)

	return nil
}

// ValidateEmail CreateUserRequest email'ini doğrular
func (req *CreateUserRequest) ValidateEmail() error {
	// Boş kontrol
	if strings.TrimSpace(req.Email) == "" {
		return fmt.Errorf("email adresi boş olamaz")
	}

	// Email format kontrolü
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(req.Email) {
		return fmt.Errorf("geçersiz email formatı")
	}

	// Uzunluk kontrol
	if len(req.Email) > 100 {
		return fmt.Errorf("email adresi en fazla 100 karakter olabilir")
	}

	// Normalize et
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	return nil
}

// ValidatePassword şifreyi doğrular
func (req *CreateUserRequest) ValidatePassword() error {
	// Boş kontrol
	if req.Password == "" {
		return fmt.Errorf("şifre boş olamaz")
	}

	// Confirm password boş kontrol
	if req.ConfirmPassword == "" {
		return fmt.Errorf("şifre tekrarı boş olamaz")
	}

	// Şifre eşleşme kontrolü
	if req.Password != req.ConfirmPassword {
		return fmt.Errorf("şifreler eşleşmiyor")
	}

	// Uzunluk kontrol
	if len(req.Password) < 6 {
		return fmt.Errorf("şifre en az 6 karakter olmalı")
	}

	if len(req.Password) > 100 {
		return fmt.Errorf("şifre en fazla 100 karakter olabilir")
	}

	// Güçlü şifre kontrolü (opsiyonel)
	if !req.isStrongPassword() {
		return fmt.Errorf("şifre en az bir büyük harf, bir küçük harf ve bir rakam içermeli")
	}

	return nil
}

// ValidateRole CreateUserRequest role'ünü doğrular
func (req *CreateUserRequest) ValidateRole() error {
	// Geçerli roller
	validRoles := map[string]bool{
		"user":  true,
		"admin": true,
		"mod":   true,
	}

	// Boşsa default role ver
	if req.Role == "" {
		req.Role = "user"
		return nil
	}

	// Role kontrolü
	if !validRoles[strings.ToLower(req.Role)] {
		return fmt.Errorf("geçersiz rol: %s. Geçerli roller: user, admin, mod", req.Role)
	}

	// Küçük harfe çevir
	req.Role = strings.ToLower(req.Role)

	return nil
}

// isStrongPassword güçlü şifre kontrolü
func (req *CreateUserRequest) isStrongPassword() bool {
	var (
		hasUpper   = false
		hasLower   = false
		hasNumber  = false
		hasSpecial = false
	)

	for _, char := range req.Password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	// En az 3 kriter karşılanmalı
	criteriaCount := 0
	if hasUpper {
		criteriaCount++
	}
	if hasLower {
		criteriaCount++
	}
	if hasNumber {
		criteriaCount++
	}
	if hasSpecial {
		criteriaCount++
	}

	return criteriaCount >= 3
}

// ========== LOGIN REQUEST VALIDATION ==========

// Validate LoginRequest'i doğrular
func (req *LoginRequest) Validate() error {
	// Email kontrolü
	if strings.TrimSpace(req.Email) == "" {
		return fmt.Errorf("email adresi boş olamaz")
	}

	// Email format kontrolü
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(req.Email) {
		return fmt.Errorf("geçersiz email formatı")
	}

	// Password kontrolü
	if req.Password == "" {
		return fmt.Errorf("şifre boş olamaz")
	}

	// Normalize
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	return nil
}

// ========== UPDATE REQUEST VALIDATION ==========

// Validate UpdateUserRequest'i doğrular
func (req *UpdateUserRequest) Validate() error {
	// En az bir field gönderilmiş mi?
	if req.Name == nil && req.Email == nil && req.Password == nil && req.Role == nil {
		return fmt.Errorf("güncellenecek en az bir alan belirtilmeli")
	}

	// Name kontrol
	if req.Name != nil {
		if strings.TrimSpace(*req.Name) == "" {
			return fmt.Errorf("kullanıcı adı boş olamaz")
		}
		if len(*req.Name) < 2 || len(*req.Name) > 50 {
			return fmt.Errorf("kullanıcı adı 2-50 karakter arası olmalı")
		}
		// Normalize
		*req.Name = strings.TrimSpace(*req.Name)
	}

	// Email kontrol
	if req.Email != nil {
		emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
		if !emailRegex.MatchString(*req.Email) {
			return fmt.Errorf("geçersiz email formatı")
		}
		// Normalize
		*req.Email = strings.ToLower(strings.TrimSpace(*req.Email))
	}

	// Password kontrol
	if req.Password != nil {
		if len(*req.Password) < 6 {
			return fmt.Errorf("şifre en az 6 karakter olmalı")
		}
	}

	// Role kontrol
	if req.Role != nil {
		validRoles := map[string]bool{
			"user":  true,
			"admin": true,
			"mod":   true,
		}
		if !validRoles[strings.ToLower(*req.Role)] {
			return fmt.Errorf("geçersiz rol: %s", *req.Role)
		}
		// Normalize
		*req.Role = strings.ToLower(*req.Role)
	}

	return nil
}
