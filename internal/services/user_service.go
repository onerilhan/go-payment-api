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

	// JWT token oluştur
	token, err := auth.GenerateToken(user.ID, user.Email)
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
	if req.Name == nil && req.Email == nil && req.Password == nil {
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
