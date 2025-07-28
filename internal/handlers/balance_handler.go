package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/rs/zerolog/log"

	"github.com/onerilhan/go-payment-api/internal/auth"
	"github.com/onerilhan/go-payment-api/internal/middleware"
	"github.com/onerilhan/go-payment-api/internal/services"
)

// BalanceHandler balance HTTP isteklerini yönetir
type BalanceHandler struct {
	balanceService *services.BalanceService
}

// NewBalanceHandler yeni handler oluşturur
func NewBalanceHandler(balanceService *services.BalanceService) *BalanceHandler {
	return &BalanceHandler{balanceService: balanceService}
}

// GetCurrentBalance kullanıcının mevcut bakiyesini döner (protected)
func (h *BalanceHandler) GetCurrentBalance(w http.ResponseWriter, r *http.Request) {
	// Sadece GET metoduna izin ver
	if r.Method != http.MethodGet {
		http.Error(w, "Geçersiz HTTP metodu", http.StatusMethodNotAllowed)
		return
	}

	// Context'ten user bilgilerini al
	claims, ok := r.Context().Value(middleware.UserContextKey).(*auth.Claims)
	if !ok {
		http.Error(w, "Yetkilendirme hatası. Lütfen tekrar giriş yapın.", http.StatusUnauthorized)
		return
	}

	// Kullanıcının bakiyesini getir
	balance, err := h.balanceService.GetBalance(claims.UserID)
	if err != nil {
		log.Error().Err(err).Int("user_id", claims.UserID).Msg("Bakiye getirilemedi")
		http.Error(w, "Bakiye bilgisi alınamadı. Lütfen tekrar deneyin.", http.StatusInternalServerError)
		return
	}

	// Standardized success response
	response := map[string]interface{}{
		"success": true,
		"data":    balance,
		"message": "Bakiye bilgisi başarıyla getirildi",
	}

	// Başarılı yanıt
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

	log.Info().Int("user_id", claims.UserID).Float64("balance", balance.Amount).Msg("Bakiye bilgisi getirildi")
}

// GetBalanceHistory kullanıcının bakiye geçmişi endpoint'i (protected)
func (h *BalanceHandler) GetBalanceHistory(w http.ResponseWriter, r *http.Request) {
	// Sadece GET metoduna izin ver
	if r.Method != http.MethodGet {
		http.Error(w, "Geçersiz HTTP metodu", http.StatusMethodNotAllowed)
		return
	}

	// Context'ten user bilgilerini al
	claims, ok := r.Context().Value(middleware.UserContextKey).(*auth.Claims)
	if !ok {
		http.Error(w, "Yetkilendirme hatası. Lütfen tekrar giriş yapın.", http.StatusUnauthorized)
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

	// Bakiye geçmişini getir
	history, err := h.balanceService.GetBalanceHistory(claims.UserID, limit, offset)
	if err != nil {
		log.Error().Err(err).Int("user_id", claims.UserID).Msg("Bakiye geçmişi getirilemedi")
		http.Error(w, "Bakiye geçmişi alınamadı. Lütfen tekrar deneyin.", http.StatusInternalServerError)
		return
	}

	// Standardized success response
	response := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"history": history,
			"limit":   limit,
			"offset":  offset,
			"count":   len(history),
		},
		"message": "Bakiye geçmişi başarıyla getirildi",
	}

	// Başarılı yanıt
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

	log.Info().
		Int("user_id", claims.UserID).
		Int("count", len(history)).
		Int("limit", limit).
		Int("offset", offset).
		Msg("Bakiye geçmişi getirildi")
}

// GetBalanceAtTime belirli tarihte bakiye endpoint'i (protected)
func (h *BalanceHandler) GetBalanceAtTime(w http.ResponseWriter, r *http.Request) {
	// Sadece GET metoduna izin ver
	if r.Method != http.MethodGet {
		http.Error(w, "Geçersiz HTTP metodu", http.StatusMethodNotAllowed)
		return
	}

	// Context'ten user bilgilerini al
	claims, ok := r.Context().Value(middleware.UserContextKey).(*auth.Claims)
	if !ok {
		http.Error(w, "Yetkilendirme hatası. Lütfen tekrar giriş yapın.", http.StatusUnauthorized)
		return
	}

	// Query parameter'dan tarihi al
	timeStr := r.URL.Query().Get("time")
	if timeStr == "" {
		http.Error(w, "Tarih parametresi gerekli. Format: ?time=2025-07-28T15:30:00Z", http.StatusBadRequest)
		return
	}

	// Bakiyeyi belirli tarihte hesapla
	balanceAtTime, err := h.balanceService.GetBalanceAtTime(claims.UserID, timeStr)
	if err != nil {
		log.Error().Err(err).Int("user_id", claims.UserID).Str("time", timeStr).Msg("Belirli tarihteki bakiye hesaplanamadı")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Standardized success response
	response := map[string]interface{}{
		"success": true,
		"data":    balanceAtTime,
		"message": "Belirli tarihteki bakiye başarıyla hesaplandı",
	}

	// Başarılı yanıt
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

	log.Info().
		Int("user_id", claims.UserID).
		Str("time", timeStr).
		Float64("amount", balanceAtTime.Amount).
		Msg("Belirli tarihteki bakiye hesaplandı")
}
