package middleware

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/onerilhan/go-payment-api/internal/middleware/errors"
)

// NotFoundJSONHandler JSON formatında 404 Not Found döner
func NotFoundJSONHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// ErrorResponse struct'ını kullan
		response := errors.ErrorResponse{
			Success:   false,
			Error:     "Endpoint bulunamadı. API dokümantasyonunu kontrol edin.",
			Code:      http.StatusNotFound,
			Timestamp: time.Now().Format(time.RFC3339),
			RequestID: w.Header().Get("X-Request-ID"),
			Details: map[string]interface{}{
				"method": r.Method,
				"path":   r.URL.Path,
			},
		}

		// Header'ları set et
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)

		// JSON encode et
		if err := json.NewEncoder(w).Encode(response); err != nil {
			// Fallback: JSON encode başarısızsa plain text
			log.Error().Err(err).Msg("NotFound JSON encoding failed")
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}

		// Log the 404
		log.Warn().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("client_ip", getClientIP(r)).
			Str("user_agent", r.Header.Get("User-Agent")).
			Msg("404 Not Found")
	})
}

// MethodNotAllowedJSONHandler JSON formatında 405 Method Not Allowed döner
func MethodNotAllowedJSONHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// ErrorResponse struct'ını kullan
		response := errors.ErrorResponse{
			Success:   false,
			Error:     "HTTP metodu bu endpoint için desteklenmiyor.",
			Code:      http.StatusMethodNotAllowed,
			Timestamp: time.Now().Format(time.RFC3339),
			RequestID: w.Header().Get("X-Request-ID"),
			Details: map[string]interface{}{
				"method":       r.Method,
				"path":         r.URL.Path,
				"allowed_info": "Desteklenen metodlar için OPTIONS isteği gönderin",
			},
		}

		// Header'ları set et
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)

		// JSON encode et
		if err := json.NewEncoder(w).Encode(response); err != nil {
			// Fallback: JSON encode başarısızsa plain text
			log.Error().Err(err).Msg("MethodNotAllowed JSON encoding failed")
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		// Log the 405
		log.Warn().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("client_ip", getClientIP(r)).
			Str("user_agent", r.Header.Get("User-Agent")).
			Msg("405 Method Not Allowed")
	})
}
