package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/onerilhan/go-payment-api/internal/auth"
	"github.com/onerilhan/go-payment-api/internal/middleware"
	"github.com/onerilhan/go-payment-api/internal/models"
	"github.com/onerilhan/go-payment-api/internal/services"
)

// TransactionHandler transaction HTTP isteklerini yönetir
type TransactionHandler struct {
	transactionService *services.TransactionService
	transactionQueue   *services.TransactionQueue
	balanceService     *services.BalanceService // ← YENİ: Queue eklendi
}

// NewTransactionHandler yeni handler oluşturur
func NewTransactionHandler(transactionService *services.TransactionService, transactionQueue *services.TransactionQueue, balanceService *services.BalanceService) *TransactionHandler {
	return &TransactionHandler{
		transactionService: transactionService,
		transactionQueue:   transactionQueue, // ← YENİ: Queue eklendi
		balanceService:     balanceService,
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

// Credit hesaba para yatırma endpoint'i
func (h *TransactionHandler) Credit(w http.ResponseWriter, r *http.Request) {
	// Sadece POST metoduna izin ver
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "Sadece POST metoduna izin verilir", http.StatusMethodNotAllowed)
		return
	}

	// Context'ten user bilgilerini al (JWT middleware tarafından eklenir)
	claims, ok := r.Context().Value(middleware.UserContextKey).(*auth.Claims)
	if !ok {
		http.Error(w, "Yetkilendirme hatası", http.StatusUnauthorized)
		return
	}

	// JSON'u parse et
	var req models.CreditRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Geçersiz JSON formatı", http.StatusBadRequest)
		return
	}

	// Credit işlemini yap
	transaction, err := h.transactionService.Credit(claims.UserID, &req)
	if err != nil {
		log.Error().Err(err).Int("user_id", claims.UserID).Msg("Credit işlemi başarısız")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Güncel bakiyeyi al
	newBalance, err := h.balanceService.GetBalance(claims.UserID)
	if err != nil {
		log.Error().Err(err).Int("user_id", claims.UserID).Msg("Bakiye alınamadı")
		// Transaction başarılı ama bakiye alınamadı - yine de devam et
		newBalance = &models.Balance{Amount: 0}
	}

	// Güvenli response oluştur (hassas bilgileri filtrele)
	response := models.CreditResponse{
		Success: true,
		Transaction: &models.TransactionSummary{
			ID:          transaction.ID,
			Amount:      transaction.Amount,
			Type:        transaction.Type,
			Status:      transaction.Status,
			Description: transaction.Description,
			CreatedAt:   transaction.CreatedAt.Format("2006-01-02T15:04:05Z"),
		},
		NewBalance: newBalance.Amount,
		Message:    "Para yatırma işlemi başarılı",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)

	log.Info().
		Int("user_id", claims.UserID).
		Float64("amount", req.Amount).
		Float64("new_balance", newBalance.Amount).
		Msg("Credit işlemi tamamlandı")
}

// Debit hesaptan para çekme endpoint'i
func (h *TransactionHandler) Debit(w http.ResponseWriter, r *http.Request) {
	// Sadece POST metoduna izin ver
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "Sadece POST metoduna izin verilir", http.StatusMethodNotAllowed)
		return
	}

	// Context'ten user bilgilerini al (JWT middleware tarafından eklenir)
	claims, ok := r.Context().Value(middleware.UserContextKey).(*auth.Claims)
	if !ok {
		http.Error(w, "Yetkilendirme hatası", http.StatusUnauthorized)
		return
	}

	// JSON'u parse et
	var req models.DebitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Geçersiz JSON formatı", http.StatusBadRequest)
		return
	}

	// Debit işlemini yap
	transaction, err := h.transactionService.Debit(claims.UserID, &req)
	if err != nil {
		log.Error().Err(err).Int("user_id", claims.UserID).Msg("Debit işlemi başarısız")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Güncel bakiyeyi al
	newBalance, err := h.balanceService.GetBalance(claims.UserID)
	if err != nil {
		log.Error().Err(err).Int("user_id", claims.UserID).Msg("Bakiye alınamadı")
		// Transaction başarılı ama bakiye alınamadı - yine de devam et
		newBalance = &models.Balance{Amount: 0}
	}

	// Response oluştur
	response := models.DebitResponse{
		Success: true,
		Transaction: &models.TransactionSummary{
			ID:          transaction.ID,
			Amount:      transaction.Amount,
			Type:        transaction.Type,
			Status:      transaction.Status,
			Description: transaction.Description,
			CreatedAt:   transaction.CreatedAt.Format("2006-01-02T15:04:05Z"),
		},
		NewBalance: newBalance.Amount,
		Message:    "Para çekme işlemi başarılı",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)

	log.Info().
		Int("user_id", claims.UserID).
		Float64("amount", req.Amount).
		Float64("new_balance", newBalance.Amount).
		Msg("Debit işlemi tamamlandı")
}

// GetTransactionByID ID ile transaction getirme endpoint'i
func (h *TransactionHandler) GetTransactionByID(w http.ResponseWriter, r *http.Request) {
	// Sadece GET metoduna izin ver
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "Sadece GET metoduna izin verilir", http.StatusMethodNotAllowed)
		return
	}

	// Context'ten user bilgilerini al
	claims, ok := r.Context().Value(middleware.UserContextKey).(*auth.Claims)
	if !ok {
		http.Error(w, "Yetkilendirme hatası", http.StatusUnauthorized)
		return
	}

	// URL'den transaction ID'yi al
	// URL format: /api/v1/transactions/{id}
	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 5 {
		http.Error(w, "Geçersiz URL formatı", http.StatusBadRequest)
		return
	}

	// ID'yi parse et
	idStr := parts[4] // /api/v1/transactions/{id}
	transactionID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Geçersiz transaction ID", http.StatusBadRequest)
		return
	}

	// Transaction'ı getir
	transaction, err := h.transactionService.GetTransactionByID(transactionID)
	if err != nil {
		log.Error().Err(err).Int("transaction_id", transactionID).Msg("Transaction bulunamadı")
		http.Error(w, "Transaction bulunamadı", http.StatusNotFound)
		return
	}

	// Kullanıcı bu transaction'a erişebilir mi?
	canAccess := false
	if transaction.FromUserID != nil && *transaction.FromUserID == claims.UserID {
		canAccess = true
	}
	if transaction.ToUserID != nil && *transaction.ToUserID == claims.UserID {
		canAccess = true
	}

	if !canAccess {
		log.Warn().
			Int("user_id", claims.UserID).
			Int("transaction_id", transactionID).
			Msg("Yetkisiz transaction erişim denemesi")
		http.Error(w, "Bu transaction'a erişim yetkiniz yok", http.StatusForbidden)
		return
	}

	// Başarılı yanıt
	// Güvenli response oluştur (hassas bilgileri filtrele)
	response := map[string]interface{}{
		"success": true,
		"data": &models.TransactionSummary{
			ID:          transaction.ID,
			Amount:      transaction.Amount,
			Type:        transaction.Type,
			Status:      transaction.Status,
			Description: transaction.Description,
			CreatedAt:   transaction.CreatedAt.Format("2006-01-02T15:04:05Z"),
		},
		"message": "Transaction başarıyla getirildi",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

	log.Info().
		Int("user_id", claims.UserID).
		Int("transaction_id", transactionID).
		Msg("Transaction detayı getirildi")
}
