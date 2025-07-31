package services

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"github.com/onerilhan/go-payment-api/internal/auth"
	"github.com/onerilhan/go-payment-api/internal/models"
	"github.com/onerilhan/go-payment-api/internal/repository"
)

// UserService kullanıcı business logic'i
type UserService struct {
	userRepo *repository.UserRepository
}

// NewUserService yeni service oluşturur
func NewUserService(userRepo *repository.UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

// Register yeni kullanıcı kaydeder
func (s *UserService) Register(req *models.CreateUserRequest) (*models.User, error) {
	// Email zaten var mı kontrol et
	existingUser, _ := s.userRepo.GetByEmail(req.Email)
	if existingUser != nil {
		return nil, fmt.Errorf("bu email zaten kullanılıyor")
	}

	// GÜVENLIK: Role assignment kontrolü
	// Sadece admin ve mod rolleri özel izin gerektirir
	if req.Role == "admin" || req.Role == "mod" {
		return nil, fmt.Errorf("admin ve moderator hesapları sadece sistem yöneticisi tarafından oluşturulabilir")
	}

	// Geçerli role kontrolü ve default assignment
	if req.Role == "" || req.Role == "user" {
		req.Role = "user" // Default role
	} else {
		// Geçersiz role girişi
		return nil, fmt.Errorf("geçersiz rol: %s. Sadece 'user' rolü ile kayıt olabilirsiniz", req.Role)
	}

	// Şifreyi hashle
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("şifre hashlenemedi: %w", err)
	}

	// Hashlenen şifreyi request'e ata
	req.Password = string(hashedPassword)

	// Kullanıcıyı oluştur
	user, err := s.userRepo.Create(req)
	if err != nil {
		return nil, fmt.Errorf("kullanıcı oluşturulamadı: %w", err)
	}

	// Role'u set et
	user.Role = req.Role

	return user, nil
}

// Login kullanıcı girişi yapar ve token döner
func (s *UserService) Login(req *models.LoginRequest) (*models.LoginResponse, error) {
	// Email ile kullanıcıyı bul
	user, err := s.userRepo.GetByEmail(req.Email)
	if err != nil {
		return nil, fmt.Errorf("email veya şifre hatalı")
	}

	// Şifreyi kontrol et
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		return nil, fmt.Errorf("email veya şifre hatalı")
	}

	// JWT token oluştur (role'u da dahil et)
	token, err := auth.GenerateToken(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, fmt.Errorf("token oluşturulamadı: %w", err)
	}

	// Response oluştur
	response := &models.LoginResponse{
		User:  user,
		Token: token,
	}

	return response, nil
}

// CreateAdminUser sadece sistem tarafından admin user oluşturur (direct database call)
func (s *UserService) CreateAdminUser(req *models.CreateUserRequest) (*models.User, error) {
	// Email zaten var mı kontrol et
	existingUser, _ := s.userRepo.GetByEmail(req.Email)
	if existingUser != nil {
		return nil, fmt.Errorf("bu email zaten kullanılıyor")
	}

	// Role'u admin olarak force et
	req.Role = "admin"

	// Şifreyi hashle
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("şifre hashlenemedi: %w", err)
	}

	req.Password = string(hashedPassword)

	// Admin kullanıcıyı oluştur
	user, err := s.userRepo.Create(req)
	if err != nil {
		return nil, fmt.Errorf("admin kullanıcı oluşturulamadı: %w", err)
	}

	user.Role = "admin"
	return user, nil
}

// PromoteUserToMod bir user'ı moderator yapar (sadece admin yapabilir)
func (s *UserService) PromoteUserToMod(adminUserID, targetUserID int) error {
	// Admin kontrolü burada yapılmayacak, RBAC middleware'de yapılacak

	// Target user'ı bul
	targetUser, err := s.userRepo.GetByID(targetUserID)
	if err != nil {
		return fmt.Errorf("kullanıcı bulunamadı: %w", err)
	}

	// Zaten admin ise promote etme
	if targetUser.Role == "admin" {
		return fmt.Errorf("admin kullanıcılar moderator yapılamaz")
	}

	// Role'u mod olarak güncelle
	updateReq := &models.UpdateUserRequest{
		Role: stringPtr("mod"),
	}

	_, err = s.userRepo.Update(targetUserID, updateReq)
	if err != nil {
		return fmt.Errorf("kullanıcı moderator yapılamadı: %w", err)
	}

	return nil
}

// DemoteUser bir mod/admin'i user yapar (sadece admin yapabilir)
func (s *UserService) DemoteUser(adminUserID, targetUserID int) error {
	// Target user'ı bul
	targetUser, err := s.userRepo.GetByID(targetUserID)
	if err != nil {
		return fmt.Errorf("kullanıcı bulunamadı: %w", err)
	}

	// Self-demotion kontrolü
	if adminUserID == targetUserID {
		return fmt.Errorf("kendi rolünüzü düşüremezsiniz")
	}

	// Zaten user ise demote etme (isteğe bağlı kontrol)
	if targetUser.Role == "user" {
		return fmt.Errorf("kullanici zaten user rolunde")
	}

	// Role'u user olarak güncelle
	updateReq := &models.UpdateUserRequest{
		Role: stringPtr("user"),
	}

	_, err = s.userRepo.Update(targetUserID, updateReq)
	if err != nil {
		return fmt.Errorf("kullanıcı rolü güncellenemedi: %w", err)
	}

	return nil
}

// GetUserByID ID ile kullanıcı getirir
func (s *UserService) GetUserByID(userID int) (*models.User, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, fmt.Errorf("kullanıcı bulunamadı: %w", err)
	}
	return user, nil
}

// UpdateUser kullanıcı bilgilerini günceller
func (s *UserService) UpdateUser(userID int, req *models.UpdateUserRequest) (*models.User, error) {
	// En az bir field gönderilmiş mi?
	if req.Name == nil && req.Email == nil && req.Password == nil && req.Role == nil {
		return nil, fmt.Errorf("güncellenecek en az bir alan belirtilmeli")
	}

	// Email değiştiriliyorsa çakışma kontrolü
	if req.Email != nil {
		existingUser, _ := s.userRepo.GetByEmail(*req.Email)
		if existingUser != nil && existingUser.ID != userID {
			return nil, fmt.Errorf("bu email zaten başka bir kullanıcı tarafından kullanılıyor")
		}
	}

	// Repository'den güncelle
	updatedUser, err := s.userRepo.Update(userID, req)
	if err != nil {
		return nil, fmt.Errorf("kullanıcı güncellenemedi: %w", err)
	}

	return updatedUser, nil
}

// DeleteUser kullanıcıyı siler (soft delete)
func (s *UserService) DeleteUser(userID int) error {
	// Repository'den kullanıcıyı sil
	err := s.userRepo.Delete(userID)
	if err != nil {
		return fmt.Errorf("kullanıcı silinemedi: %w", err)
	}

	return nil
}

// GetAllUsers tüm kullanıcıları listeler (pagination ile)
func (s *UserService) GetAllUsers(limit, offset int) ([]*models.User, int, error) {
	// Pagination validation
	if limit <= 0 || limit > 100 {
		limit = 10 // default limit
	}
	if offset < 0 {
		offset = 0 // default offset
	}

	// Repository'den kullanıcıları al
	users, totalCount, err := s.userRepo.GetAll(limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("kullanıcı listesi alınamadı: %w", err)
	}

	return users, totalCount, nil
}

// stringPtr helper function for string pointer
func stringPtr(s string) *string {
	return &s
}
