package handlers

import (
	"encoding/json"
	"net/http"

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
