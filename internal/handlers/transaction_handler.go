package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/rs/zerolog/log"

	"github.com/onerilhan/go-payment-api/internal/auth"
	"github.com/onerilhan/go-payment-api/internal/middleware"
	"github.com/onerilhan/go-payment-api/internal/models"
	"github.com/onerilhan/go-payment-api/internal/services"
)

// TransactionHandler transaction HTTP isteklerini yönetir
type TransactionHandler struct {
	transactionService *services.TransactionService
	transactionQueue   *services.TransactionQueue // ← YENİ: Queue eklendi
}

// NewTransactionHandler yeni handler oluşturur
func NewTransactionHandler(transactionService *services.TransactionService, transactionQueue *services.TransactionQueue) *TransactionHandler {
	return &TransactionHandler{
		transactionService: transactionService,
		transactionQueue:   transactionQueue, // ← YENİ: Queue eklendi
	}
}

// Transfer para transfer endpoint'i (queue ile async)
func (h *TransactionHandler) Transfer(w http.ResponseWriter, r *http.Request) {
	// Sadece POST metoduna izin ver
	if r.Method != http.MethodPost {
		http.Error(w, "Sadece POST metoduna izin verilir", http.StatusMethodNotAllowed)
		return
	}

	// Context'ten user bilgilerini al
	claims, ok := r.Context().Value(middleware.UserContextKey).(*auth.Claims)
	if !ok {
		http.Error(w, "User bilgisi bulunamadı", http.StatusInternalServerError)
		return
	}

	// JSON'u parse et
	var req models.TransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Geçersiz JSON formatı", http.StatusBadRequest)
		return
	}

	// Job'ı queue'ya ekle (async)
	resultChan := h.transactionQueue.AddJob(claims.UserID, &req)

	// Result'u bekle
	result := <-resultChan

	// Hata kontrolü
	if result.Error != nil {
		log.Error().Err(result.Error).Int("user_id", claims.UserID).Msg("Transfer başarısız")
		http.Error(w, result.Error.Error(), http.StatusBadRequest)
		return
	}

	// Başarılı yanıt
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(result.Transaction)

	log.Info().
		Int("from_user_id", claims.UserID).
		Int("to_user_id", req.ToUserID).
		Float64("amount", req.Amount).
		Msg("Para transferi queue ile başarılı")
}

// GetHistory kullanıcının transaction geçmişini döner (protected)
func (h *TransactionHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
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

	// Transaction geçmişini getir
	transactions, err := h.transactionService.GetUserTransactions(claims.UserID, limit, offset)
	if err != nil {
		log.Error().Err(err).Int("user_id", claims.UserID).Msg("Transaction geçmişi getirilemedi")
		http.Error(w, "İşlem geçmişi alınamadı. Lütfen tekrar deneyin.", http.StatusInternalServerError)
		return
	}

	// Standardized success response
	response := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"transactions": transactions,
			"limit":        limit,
			"offset":       offset,
			"count":        len(transactions),
		},
		"message": "İşlem geçmişi başarıyla getirildi",
	}

	// Başarılı yanıt
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

	log.Info().
		Int("user_id", claims.UserID).
		Int("count", len(transactions)).
		Int("limit", limit).
		Int("offset", offset).
		Msg("Transaction geçmişi getirildi")
}
