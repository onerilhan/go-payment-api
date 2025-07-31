package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"

	"github.com/onerilhan/go-payment-api/internal/auth"
	"github.com/onerilhan/go-payment-api/internal/middleware"
	"github.com/onerilhan/go-payment-api/internal/middleware/errors"
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
		panic(&errors.ValidationError{
			Message:    "Sadece POST metoduna izin verilir",
			StatusCode: http.StatusMethodNotAllowed,
			Field:      "method",
			Value:      r.Method,
		})
	}

	// JSON'u parse et
	var req models.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		panic(&errors.ValidationError{
			Message:    "Geçersiz JSON formatı",
			StatusCode: http.StatusBadRequest,
			Field:      "body",
			Value:      err.Error(),
		})
	}

	//  YENİ VALİDASYON KONTROLÜ
	if err := req.Validate(); err != nil {
		log.Warn().
			Err(err).
			Str("email", req.Email).
			Str("name", req.Name).
			Msg("❌ Validation hatası")
		panic(&errors.ValidationError{
			Message:    err.Error(),
			StatusCode: http.StatusBadRequest,
			Field:      "validation",
			Value:      req,
		})
	}

	// Kullanıcıyı oluştur
	user, err := h.userService.Register(&req)
	if err != nil {
		log.Error().Err(err).Msg("Kullanıcı kaydı başarısız")
		panic(&errors.ValidationError{
			Message:    err.Error(),
			StatusCode: http.StatusBadRequest,
			Field:      "registration",
			Value:      req.Email,
		})
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
		panic(&errors.ValidationError{
			Message:    "Sadece POST metoduna izin verilir",
			StatusCode: http.StatusMethodNotAllowed,
			Field:      "method",
			Value:      r.Method,
		})
	}

	// JSON'u parse et
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		panic(&errors.ValidationError{
			Message:    "Geçersiz JSON formatı",
			StatusCode: http.StatusBadRequest,
			Field:      "body",
			Value:      err.Error(),
		})
	}

	//  YENİ VALİDASYON KONTROLÜ
	if err := req.Validate(); err != nil {
		log.Warn().
			Err(err).
			Str("email", req.Email).
			Msg("❌ Login validation hatası")
		panic(&errors.ValidationError{
			Message:    err.Error(),
			StatusCode: http.StatusBadRequest,
			Field:      "validation",
			Value:      req.Email,
		})
	}

	// Kullanıcı girişi yap
	user, err := h.userService.Login(&req)
	if err != nil {
		log.Error().Err(err).Msg("Giriş başarısız")
		panic(&errors.AuthError{
			Message:    err.Error(),
			StatusCode: http.StatusUnauthorized,
		})
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
		panic(&errors.ValidationError{
			Message:    "Sadece GET metoduna izin verilir",
			StatusCode: http.StatusMethodNotAllowed,
			Field:      "method",
			Value:      r.Method,
		})
	}

	// Context'ten user bilgilerini al
	claims, ok := r.Context().Value(middleware.UserContextKey).(*auth.Claims)
	if !ok {
		panic(&errors.AuthError{
			Message:    "User bilgisi bulunamadı",
			StatusCode: http.StatusInternalServerError,
		})
	}

	// User ID ile kullanıcıyı bul
	user, err := h.userService.GetUserByID(claims.UserID)
	if err != nil {
		log.Error().Err(err).Int("user_id", claims.UserID).Msg("Kullanıcı bulunamadı")
		panic(&errors.ValidationError{
			Message:    "Kullanıcı bulunamadı",
			StatusCode: http.StatusNotFound,
			Field:      "user_id",
			Value:      claims.UserID,
		})
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
		panic(&errors.ValidationError{
			Message:    "Sadece POST metoduna izin verilir",
			StatusCode: http.StatusMethodNotAllowed,
			Field:      "method",
			Value:      r.Method,
		})
	}

	var req struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		panic(&errors.ValidationError{
			Message:    "Geçersiz JSON formatı",
			StatusCode: http.StatusBadRequest,
			Field:      "body",
			Value:      err.Error(),
		})
	}

	newToken, expiresIn, err := auth.RefreshToken(req.Token)
	if err != nil {
		log.Error().Err(err).Msg("Token refresh başarısız")
		panic(&errors.AuthError{
			Message:    err.Error(),
			StatusCode: http.StatusUnauthorized,
		})
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
		panic(&errors.ValidationError{
			Message:    "Sadece GET metoduna izin verilir",
			StatusCode: http.StatusMethodNotAllowed,
			Field:      "method",
			Value:      r.Method,
		})
	}

	// Context'ten user bilgilerini al (authentication kontrolü)
	_, ok := r.Context().Value(middleware.UserContextKey).(*auth.Claims)
	if !ok {
		panic(&errors.AuthError{
			Message:    "Yetkilendirme hatası",
			StatusCode: http.StatusUnauthorized,
		})
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
		panic(&errors.ValidationError{
			Message:    "Kullanıcı listesi alınamadı",
			StatusCode: http.StatusInternalServerError,
			Field:      "users",
			Value:      nil,
		})
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
		panic(&errors.AuthError{
			Message:    "Yetkilendirme hatası",
			StatusCode: http.StatusUnauthorized,
		})
	}

	// Gorilla Mux'tan URL parameter'ı al
	vars := mux.Vars(r)
	idStr, exists := vars["id"]
	if !exists {
		panic(&errors.ValidationError{
			Message:    "Kullanıcı ID parametresi gerekli",
			StatusCode: http.StatusBadRequest,
			Field:      "id",
			Value:      nil,
		})
	}

	// ID'yi parse et
	userID, err := strconv.Atoi(idStr)
	if err != nil {
		panic(&errors.ValidationError{
			Message:    "Geçersiz kullanıcı ID",
			StatusCode: http.StatusBadRequest,
			Field:      "id",
			Value:      idStr,
		})
	}

	// Kullanıcıyı getir
	user, err := h.userService.GetUserByID(userID)
	if err != nil {
		log.Error().Err(err).Int("user_id", userID).Msg("Kullanıcı bulunamadı")
		panic(&errors.ValidationError{
			Message:    "Kullanıcı bulunamadı",
			StatusCode: http.StatusNotFound,
			Field:      "user_id",
			Value:      userID,
		})
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
		panic(&errors.AuthError{
			Message:    "Yetkilendirme hatası",
			StatusCode: http.StatusUnauthorized,
		})
	}

	// Gorilla Mux'tan URL parameter'ı al
	vars := mux.Vars(r)
	idStr, exists := vars["id"]
	if !exists {
		panic(&errors.ValidationError{
			Message:    "Kullanıcı ID parametresi gerekli",
			StatusCode: http.StatusBadRequest,
			Field:      "id",
			Value:      nil,
		})
	}

	// ID'yi parse et
	targetUserID, err := strconv.Atoi(idStr)
	if err != nil {
		panic(&errors.ValidationError{
			Message:    "Geçersiz kullanıcı ID",
			StatusCode: http.StatusBadRequest,
			Field:      "id",
			Value:      idStr,
		})
	}

	// JSON'u parse et
	var req models.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		panic(&errors.ValidationError{
			Message:    "Geçersiz JSON formatı",
			StatusCode: http.StatusBadRequest,
			Field:      "body",
			Value:      err.Error(),
		})
	}

	//  YENİ VALİDASYON KONTROLÜ
	if err := req.Validate(); err != nil {
		log.Warn().
			Err(err).
			Int("user_id", targetUserID).
			Msg(" Update validation hatası")
		panic(&errors.ValidationError{
			Message:    err.Error(),
			StatusCode: http.StatusBadRequest,
			Field:      "validation",
			Value:      req,
		})
	}

	// Authorization: Sadece kendi hesabını güncelleyebilir (RBAC middleware'de kontrol edilir)
	if claims.UserID != targetUserID {
		log.Warn().
			Int("requester_id", claims.UserID).
			Int("target_id", targetUserID).
			Msg(" Yetkisiz kullanıcı güncelleme denemesi")
		panic(&errors.RBACError{
			Message:    "Sadece kendi hesabınızı güncelleyebilirsiniz",
			StatusCode: http.StatusForbidden,
			Resource:   "user",
			Action:     "update",
		})
	}

	// Güncelleme işlemini yap
	updatedUser, err := h.userService.UpdateUser(targetUserID, &req)
	if err != nil {
		log.Error().Err(err).Int("user_id", targetUserID).Msg("Kullanıcı güncellenemedi")
		panic(&errors.ValidationError{
			Message:    err.Error(),
			StatusCode: http.StatusBadRequest,
			Field:      "update",
			Value:      targetUserID,
		})
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
		panic(&errors.AuthError{
			Message:    "Yetkilendirme hatası",
			StatusCode: http.StatusUnauthorized,
		})
	}

	// Gorilla Mux'tan URL parameter'ı al
	vars := mux.Vars(r)
	idStr, exists := vars["id"]
	if !exists {
		panic(&errors.ValidationError{
			Message:    "Kullanıcı ID parametresi gerekli",
			StatusCode: http.StatusBadRequest,
			Field:      "id",
			Value:      nil,
		})
	}

	// ID'yi parse et
	targetUserID, err := strconv.Atoi(idStr)
	if err != nil {
		panic(&errors.ValidationError{
			Message:    "Geçersiz kullanıcı ID",
			StatusCode: http.StatusBadRequest,
			Field:      "id",
			Value:      idStr,
		})
	}

	// Authorization: Sadece kendi hesabını silebilir (RBAC middleware'de kontrol edilir)
	if claims.UserID != targetUserID {
		log.Warn().
			Int("requester_id", claims.UserID).
			Int("target_id", targetUserID).
			Msg(" Yetkisiz kullanıcı silme denemesi")
		panic(&errors.RBACError{
			Message:    "Sadece kendi hesabınızı silebilirsiniz",
			StatusCode: http.StatusForbidden,
			Resource:   "user",
			Action:     "delete",
		})
	}

	// Silme işlemini yap
	err = h.userService.DeleteUser(targetUserID)
	if err != nil {
		log.Error().Err(err).Int("user_id", targetUserID).Msg("Kullanıcı silinemedi")
		panic(&errors.ValidationError{
			Message:    err.Error(),
			StatusCode: http.StatusBadRequest,
			Field:      "delete",
			Value:      targetUserID,
		})
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

// PromoteToMod kullanıcıyı moderator yapma endpoint'i (sadece admin)
func (h *UserHandler) PromoteToMod(w http.ResponseWriter, r *http.Request) {
	// Context'ten admin user bilgilerini al
	claims, ok := r.Context().Value(middleware.UserContextKey).(*auth.Claims)
	if !ok {
		panic(&errors.AuthError{
			Message:    "Yetkilendirme hatası",
			StatusCode: http.StatusUnauthorized,
		})
	}

	// Gorilla Mux'tan URL parameter'ı al
	vars := mux.Vars(r)
	idStr, exists := vars["id"]
	if !exists {
		panic(&errors.ValidationError{
			Message:    "Kullanıcı ID parametresi gerekli",
			StatusCode: http.StatusBadRequest,
			Field:      "id",
			Value:      nil,
		})
	}

	// ID'yi parse et
	targetUserID, err := strconv.Atoi(idStr)
	if err != nil {
		panic(&errors.ValidationError{
			Message:    "Geçersiz kullanıcı ID",
			StatusCode: http.StatusBadRequest,
			Field:      "id",
			Value:      idStr,
		})
	}

	// Promote işlemini yap
	err = h.userService.PromoteUserToMod(claims.UserID, targetUserID)
	if err != nil {
		log.Error().Err(err).Int("target_user_id", targetUserID).Msg("Moderator promotion başarısız")
		panic(&errors.ValidationError{
			Message:    err.Error(),
			StatusCode: http.StatusBadRequest,
			Field:      "promotion",
			Value:      targetUserID,
		})
	}

	// Başarılı yanıt
	response := map[string]interface{}{
		"success": true,
		"message": "Kullanıcı başarıyla moderator yapıldı",
		"data": map[string]interface{}{
			"user_id":  targetUserID,
			"new_role": "mod",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

	log.Info().
		Int("admin_user_id", claims.UserID).
		Int("target_user_id", targetUserID).
		Msg("Kullanıcı moderator yapıldı")
}

// DemoteUser kullanıcıyı user yapma endpoint'i (sadece admin)
func (h *UserHandler) DemoteUser(w http.ResponseWriter, r *http.Request) {
	// Context'ten admin user bilgilerini al
	claims, ok := r.Context().Value(middleware.UserContextKey).(*auth.Claims)
	if !ok {
		panic(&errors.AuthError{
			Message:    "Yetkilendirme hatası",
			StatusCode: http.StatusUnauthorized,
		})
	}

	// Gorilla Mux'tan URL parameter'ı al
	vars := mux.Vars(r)
	idStr, exists := vars["id"]
	if !exists {
		panic(&errors.ValidationError{
			Message:    "Kullanıcı ID parametresi gerekli",
			StatusCode: http.StatusBadRequest,
			Field:      "id",
			Value:      nil,
		})
	}

	// ID'yi parse et
	targetUserID, err := strconv.Atoi(idStr)
	if err != nil {
		panic(&errors.ValidationError{
			Message:    "Geçersiz kullanıcı ID",
			StatusCode: http.StatusBadRequest,
			Field:      "id",
			Value:      idStr,
		})
	}

	// Demote işlemini yap
	err = h.userService.DemoteUser(claims.UserID, targetUserID)
	if err != nil {
		log.Error().Err(err).Int("target_user_id", targetUserID).Msg("User demotion başarısız")
		panic(&errors.ValidationError{
			Message:    err.Error(),
			StatusCode: http.StatusBadRequest,
			Field:      "demotion",
			Value:      targetUserID,
		})
	}

	// Başarılı yanıt
	response := map[string]interface{}{
		"success": true,
		"message": "Kullanıcı başarıyla user yapıldı",
		"data": map[string]interface{}{
			"user_id":  targetUserID,
			"new_role": "user",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

	log.Info().
		Int("admin_user_id", claims.UserID).
		Int("target_user_id", targetUserID).
		Msg("Kullanıcı user yapıldı")
}
