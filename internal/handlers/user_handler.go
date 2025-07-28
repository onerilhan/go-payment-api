package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
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

// Register kullanıcı kayıt endpoint'i - VALİDASYON EKLENDİ
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

	//  YENİ VALİDASYON KONTROLÜ
	if err := req.Validate(); err != nil {
		log.Warn().
			Err(err).
			Str("email", req.Email).
			Str("name", req.Name).
			Msg("❌ Validation hatası")
		http.Error(w, err.Error(), http.StatusBadRequest)
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

	log.Info().
		Str("email", user.Email).
		Str("role", user.Role).
		Msg(" Yeni kullanıcı kaydedildi")
}

// Login kullanıcı giriş endpoint'i - VALİDASYON EKLENDİ
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

	//  YENİ VALİDASYON KONTROLÜ
	if err := req.Validate(); err != nil {
		log.Warn().
			Err(err).
			Str("email", req.Email).
			Msg("❌ Login validation hatası")
		http.Error(w, err.Error(), http.StatusBadRequest)
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

	log.Info().
		Str("email", user.User.Email).
		Str("role", user.User.Role).
		Msg(" Kullanıcı giriş yaptı")
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

// GetAllUsers tüm kullanıcıları listeler (protected endpoint)
func (h *UserHandler) GetAllUsers(w http.ResponseWriter, r *http.Request) {
	// Sadece GET metoduna izin ver
	if r.Method != http.MethodGet {
		http.Error(w, "Sadece GET metoduna izin verilir", http.StatusMethodNotAllowed)
		return
	}

	// Context'ten user bilgilerini al (authentication kontrolü)
	_, ok := r.Context().Value(middleware.UserContextKey).(*auth.Claims)
	if !ok {
		http.Error(w, "Yetkilendirme hatası", http.StatusUnauthorized)
		return
	}

	// Query parameters (pagination)
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	// Default değerler
	limit := 10
	offset := 0

	// Limit parse et
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	// Offset parse et
	if offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	// Kullanıcı listesini getir
	users, totalCount, err := h.userService.GetAllUsers(limit, offset)
	if err != nil {
		log.Error().Err(err).Msg("Kullanıcı listesi getirilemedi")
		http.Error(w, "Kullanıcı listesi alınamadı", http.StatusInternalServerError)
		return
	}

	// Standardized success response
	response := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"users":       users,
			"total_count": totalCount,
			"limit":       limit,
			"offset":      offset,
			"count":       len(users),
		},
		"message": "Kullanıcı listesi başarıyla getirildi",
	}

	// Başarılı yanıt
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

	log.Info().
		Int("total_count", totalCount).
		Int("returned_count", len(users)).
		Int("limit", limit).
		Int("offset", offset).
		Msg("Kullanıcı listesi getirildi")
}

// GetUserByID ID ile tek kullanıcı getirme endpoint'i (Gorilla Mux version)
func (h *UserHandler) GetUserByID(w http.ResponseWriter, r *http.Request) {
	// Context'ten user bilgilerini al (authentication kontrolü)
	_, ok := r.Context().Value(middleware.UserContextKey).(*auth.Claims)
	if !ok {
		http.Error(w, "Yetkilendirme hatası", http.StatusUnauthorized)
		return
	}

	// Gorilla Mux'tan URL parameter'ı al
	vars := mux.Vars(r)
	idStr, exists := vars["id"]
	if !exists {
		http.Error(w, "Kullanıcı ID parametresi gerekli", http.StatusBadRequest)
		return
	}

	// ID'yi parse et
	userID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Geçersiz kullanıcı ID", http.StatusBadRequest)
		return
	}

	// Kullanıcıyı getir
	user, err := h.userService.GetUserByID(userID)
	if err != nil {
		log.Error().Err(err).Int("user_id", userID).Msg("Kullanıcı bulunamadı")
		http.Error(w, "Kullanıcı bulunamadı", http.StatusNotFound)
		return
	}

	// Başarılı yanıt
	response := map[string]interface{}{
		"success": true,
		"data":    user,
		"message": "Kullanıcı başarıyla getirildi",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

	log.Info().Int("user_id", userID).Msg("Kullanıcı detayı getirildi")
}

// UpdateUser kullanıcı güncelleme endpoint'i - VALİDASYON EKLENDİ
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	// Context'ten user bilgilerini al
	claims, ok := r.Context().Value(middleware.UserContextKey).(*auth.Claims)
	if !ok {
		http.Error(w, "Yetkilendirme hatası", http.StatusUnauthorized)
		return
	}

	// Gorilla Mux'tan URL parameter'ı al
	vars := mux.Vars(r)
	idStr, exists := vars["id"]
	if !exists {
		http.Error(w, "Kullanıcı ID parametresi gerekli", http.StatusBadRequest)
		return
	}

	// ID'yi parse et
	targetUserID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Geçersiz kullanıcı ID", http.StatusBadRequest)
		return
	}

	// JSON'u parse et
	var req models.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Geçersiz JSON formatı", http.StatusBadRequest)
		return
	}

	//  YENİ VALİDASYON KONTROLÜ
	if err := req.Validate(); err != nil {
		log.Warn().
			Err(err).
			Int("user_id", targetUserID).
			Msg(" Update validation hatası")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Authorization: Sadece kendi hesabını güncelleyebilir
	if claims.UserID != targetUserID {
		log.Warn().
			Int("requester_id", claims.UserID).
			Int("target_id", targetUserID).
			Msg(" Yetkisiz kullanıcı güncelleme denemesi")
		http.Error(w, "Sadece kendi hesabınızı güncelleyebilirsiniz", http.StatusForbidden)
		return
	}

	// Güncelleme işlemini yap
	updatedUser, err := h.userService.UpdateUser(targetUserID, &req)
	if err != nil {
		log.Error().Err(err).Int("user_id", targetUserID).Msg("Kullanıcı güncellenemedi")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Başarılı yanıt
	response := map[string]interface{}{
		"success": true,
		"data":    updatedUser,
		"message": "Kullanıcı başarıyla güncellendi",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

	log.Info().
		Int("user_id", targetUserID).
		Str("updated_by", claims.Email).
		Msg(" Kullanıcı güncellendi")
}

// DeleteUser kullanıcı silme endpoint'i (Gorilla Mux version)
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	// Context'ten user bilgilerini al
	claims, ok := r.Context().Value(middleware.UserContextKey).(*auth.Claims)
	if !ok {
		http.Error(w, "Yetkilendirme hatası", http.StatusUnauthorized)
		return
	}

	// Gorilla Mux'tan URL parameter'ı al
	vars := mux.Vars(r)
	idStr, exists := vars["id"]
	if !exists {
		http.Error(w, "Kullanıcı ID parametresi gerekli", http.StatusBadRequest)
		return
	}

	// ID'yi parse et
	targetUserID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Geçersiz kullanıcı ID", http.StatusBadRequest)
		return
	}

	// Authorization: Sadece kendi hesabını silebilir
	if claims.UserID != targetUserID {
		log.Warn().
			Int("requester_id", claims.UserID).
			Int("target_id", targetUserID).
			Msg(" Yetkisiz kullanıcı silme denemesi")
		http.Error(w, "Sadece kendi hesabınızı silebilirsiniz", http.StatusForbidden)
		return
	}

	// Silme işlemini yap
	err = h.userService.DeleteUser(targetUserID)
	if err != nil {
		log.Error().Err(err).Int("user_id", targetUserID).Msg("Kullanıcı silinemedi")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Başarılı yanıt
	response := map[string]interface{}{
		"success": true,
		"message": "Kullanıcı başarıyla silindi",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

	log.Info().
		Int("user_id", targetUserID).
		Str("deleted_by", claims.Email).
		Msg(" Kullanıcı silindi (soft delete)")
}
