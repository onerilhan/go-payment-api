package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/onerilhan/go-payment-api/internal/interfaces"
	"github.com/onerilhan/go-payment-api/internal/models"
)

// MockUserRepository - test için mock repository
type MockUserRepository struct {
	mock.Mock
}

var _ interfaces.UserRepositoryInterface = (*MockUserRepository)(nil)

func (m *MockUserRepository) Create(user *models.CreateUserRequest) (*models.User, error) {
	args := m.Called(user)
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(email string) (*models.User, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByID(id int) (*models.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) Update(id int, user *models.UpdateUserRequest) (*models.User, error) {
	args := m.Called(id, user)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) Delete(id int) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockUserRepository) GetAll(limit, offset int) ([]*models.User, int, error) {
	args := m.Called(limit, offset)
	return args.Get(0).([]*models.User), args.Int(1), args.Error(2)
}

// İlk basit test - kullanıcı kaydı
func TestUserService_Register_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockUserRepository)
	userService := NewUserService(mockRepo)

	req := &models.CreateUserRequest{
		Name:            "Test User",
		Email:           "test@example.com",
		Password:        "Password123!",
		ConfirmPassword: "Password123!",
		Role:            "user",
	}

	expectedUser := &models.User{
		ID:    1,
		Name:  "Test User",
		Email: "test@example.com",
		Role:  "user",
	}

	// Mock expectations
	mockRepo.On("GetByEmail", "test@example.com").Return(nil, nil) // Email yok
	mockRepo.On("Create", mock.AnythingOfType("*models.CreateUserRequest")).Return(expectedUser, nil)

	// Act
	result, err := userService.Register(req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Test User", result.Name)
	assert.Equal(t, "test@example.com", result.Email)
	assert.Equal(t, "user", result.Role)

	// Mock assertions
	mockRepo.AssertExpectations(t)
}

// Email already exists test
func TestUserService_Register_EmailExists(t *testing.T) {
	// Arrange
	mockRepo := new(MockUserRepository)
	userService := NewUserService(mockRepo)

	req := &models.CreateUserRequest{
		Name:            "Test User",
		Email:           "existing@example.com",
		Password:        "Password123!",
		ConfirmPassword: "Password123!",
		Role:            "user",
	}

	existingUser := &models.User{
		ID:    1,
		Email: "existing@example.com",
	}

	// Mock: Email zaten var
	mockRepo.On("GetByEmail", "existing@example.com").Return(existingUser, nil)

	// Act
	result, err := userService.Register(req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "bu email zaten kullanılıyor")

	// Mock assertions
	mockRepo.AssertExpectations(t)
}
