package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"

	"github.com/onerilhan/go-payment-api/internal/auth"
	"github.com/onerilhan/go-payment-api/internal/middleware"
	"github.com/onerilhan/go-payment-api/internal/models"
	"github.com/onerilhan/go-payment-api/internal/services"
)

// UserHandler HTTP isteklerini yönetir
type UserHandler struct {
	userService *services.UserService
}

// NewUserHandler yeni handler oluşturur
func NewUserHandler(userService *services.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// Register kullanıcı kayıt endpoint'i
func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	// Sadece POST metoduna izin ver
	if r.Method != http.MethodPost {
		http.Error(w, "Sadece POST metoduna izin verilir", http.StatusMethodNotAllowed)
		return
	}

	// JSON'u parse et
	var req models.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Geçersiz JSON formatı", http.StatusBadRequest)
		return
	}

	// Kullanıcıyı oluştur
	user, err := h.userService.Register(&req)
	if err != nil {
		log.Error().Err(err).Msg("Kullanıcı kaydı başarısız")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Başarılı yanıt
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)

	log.Info().Str("email", user.Email).Msg("Yeni kullanıcı kaydedildi")
}

// Login kullanıcı giriş endpoint'i
func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	// Sadece POST metoduna izin ver
	if r.Method != http.MethodPost {
		http.Error(w, "Sadece POST metoduna izin verilir", http.StatusMethodNotAllowed)
		return
	}

	// JSON'u parse et
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Geçersiz JSON formatı", http.StatusBadRequest)
		return
	}

	// Kullanıcı girişi yap
	user, err := h.userService.Login(&req)
	if err != nil {
		log.Error().Err(err).Msg("Giriş başarısız")
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Başarılı yanıt
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)

	log.Info().Str("email", user.User.Email).Msg("Kullanıcı giriş yaptı")
}

// GetProfile kullanıcının kendi profilini döner (protected endpoint)
func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	// Sadece GET metoduna izin ver
	if r.Method != http.MethodGet {
		http.Error(w, "Sadece GET metoduna izin verilir", http.StatusMethodNotAllowed)
		return
	}

	// Context'ten user bilgilerini al
	claims, ok := r.Context().Value(middleware.UserContextKey).(*auth.Claims)
	if !ok {
		http.Error(w, "User bilgisi bulunamadı", http.StatusInternalServerError)
		return
	}

	// User ID ile kullanıcıyı bul
	user, err := h.userService.GetUserByID(claims.UserID)
	if err != nil {
		log.Error().Err(err).Int("user_id", claims.UserID).Msg("Kullanıcı bulunamadı")
		http.Error(w, "Kullanıcı bulunamadı", http.StatusNotFound)
		return
	}

	// Başarılı yanıt
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)

	log.Info().Int("user_id", claims.UserID).Msg("Profil bilgileri getirildi")
}

// Refresh JWT token yenileme endpoint'i
func (h *UserHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "Sadece POST metoduna izin verilir", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Geçersiz JSON formatı", http.StatusBadRequest)
		return
	}

	newToken, expiresIn, err := auth.RefreshToken(req.Token)
	if err != nil {
		log.Error().Err(err).Msg("Token refresh başarısız")
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	response := models.RefreshResponse{
		Success:   true,
		Token:     newToken,
		ExpiresIn: expiresIn,
		Message:   "Token başarıyla yenilendi",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
