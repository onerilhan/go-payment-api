package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/onerilhan/go-payment-api/internal/middleware/errors"
)

// ErrorHandlingMiddleware centralized error handling ve panic recovery
func ErrorHandlingMiddleware(config *errors.ErrorConfig) func(http.Handler) http.Handler {
	// Config nil ise default kullan
	if config == nil {
		config = errors.DefaultErrorConfig()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Panic recovery defer function
			defer func() {
				if recovered := recover(); recovered != nil {
					var statusCode int = 500
					var errorMessage string
					var isAPIError bool
					var errorType string

					// Type switch ile esnek error yakalama
					switch err := recovered.(type) {
					case errors.APIError:
						// APIError interface'ini implement eden tüm error'lar
						statusCode = err.Status()
						errorMessage = err.Error()
						isAPIError = true
						errorType = fmt.Sprintf("%T", err)

						// API error'u özel olarak logla
						logAPIError(err, r, errorType)

					case error:
						// Normal Go error
						statusCode = 500
						errorMessage = err.Error()
						isAPIError = false
						errorType = "error"

					default:
						// Diğer panic tipleri
						statusCode = 500
						errorMessage = fmt.Sprintf("Server panic: %v", recovered)
						isAPIError = false
						errorType = "panic"
					}

					// Panic bilgilerini topla (sadece normal panic/error için stack trace)
					var panicInfo *errors.PanicInfo
					if !isAPIError {
						panicInfo = &errors.PanicInfo{
							Value:     recovered,
							Stack:     string(debug.Stack()),
							RequestID: w.Header().Get("X-Request-ID"),
							Method:    r.Method,
							Path:      r.URL.Path,
							UserAgent: r.Header.Get("User-Agent"),
							ClientIP:  getClientIP(r),
							Timestamp: time.Now(),
						}

						// Normal panic'i logla
						logPanic(panicInfo, config)
					}

					// Response header'ları temizle (panic sonrası)
					for key := range w.Header() {
						if !contains(config.IncludeHeaders, key) {
							w.Header().Del(key)
						}
					}

					// Error response gönder
					var stack string
					if !isAPIError && panicInfo != nil {
						stack = panicInfo.Stack
					}

					sendErrorResponse(w, r, statusCode, errorMessage, config, stack)
				}
			}()

			// Response writer'ı wrap et (error handling için)
			wrapped := &errorResponseWriter{
				ResponseWriter: w,
				request:        r,
				config:         config,
			}

			// Handler'ı çalıştır
			next.ServeHTTP(wrapped, r)

			// Eğer handler HTTP error code döndüyse, custom error response gönder
			if wrapped.statusCode >= 400 && !wrapped.responseWritten {
				// Status code'a göre custom mesaj al
				errorMessage := getErrorMessage(wrapped.statusCode, config)
				sendErrorResponse(w, r, wrapped.statusCode, errorMessage, config, "")
			}
		})
	}
}

// errorResponseWriter custom response writer for error handling
type errorResponseWriter struct {
	http.ResponseWriter
	request         *http.Request
	config          *errors.ErrorConfig
	statusCode      int
	responseWritten bool
}

// WriteHeader status code'u yakala
func (erw *errorResponseWriter) WriteHeader(code int) {
	// Eğer zaten response yazıldıysa, ikinci kez yazma
	if erw.responseWritten {
		return
	}

	erw.statusCode = code

	// Eğer error status code ise ve henüz response yazılmamışsa middleware handle edecek
	if code >= 400 {
		erw.responseWritten = true
		return
	}

	erw.responseWritten = true
	erw.ResponseWriter.WriteHeader(code)
}

// Write response yazıldığını işaretle
func (erw *errorResponseWriter) Write(b []byte) (int, error) {
	// Eğer status code set edilmemişse 200 kabul et
	if erw.statusCode == 0 {
		erw.statusCode = http.StatusOK
	}

	// Error status code'da direkt response yazma (middleware handle edecek)
	if erw.statusCode >= 400 && !erw.responseWritten {
		return len(b), nil // Fake write success
	}

	erw.responseWritten = true

	// Normal response write
	if erw.statusCode != http.StatusOK {
		erw.ResponseWriter.WriteHeader(erw.statusCode)
	}

	return erw.ResponseWriter.Write(b)
}

// sendErrorResponse standardized error response gönderir
func sendErrorResponse(w http.ResponseWriter, r *http.Request, statusCode int, message string, config *errors.ErrorConfig, stack string) {
	// Response body oluştur
	response := errors.ErrorResponse{
		Success:   false,
		Error:     truncateString(message, config.MaxErrorLength),
		Code:      statusCode,
		Timestamp: time.Now().Format(time.RFC3339),
		RequestID: w.Header().Get("X-Request-ID"),
	}

	// Stack trace ekle (sadece development'ta)
	if config.ShowStackTrace && stack != "" {
		response.Stack = stack
	}

	// Additional details ekle
	response.Details = map[string]interface{}{
		"method": r.Method,
		"path":   r.URL.Path,
	}

	// JSON response gönder
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		// JSON encoding hatası - environment'a göre fallback
		if config.ShowStackTrace {
			// Development: Detaylı log
			log.Error().
				Err(err).
				Str("request_id", response.RequestID).
				Str("original_error", message).
				Int("status_code", statusCode).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Msg("Error response JSON encoding failed - development details")
		} else {
			// Production: Minimal log
			log.Error().
				Err(err).
				Str("request_id", response.RequestID).
				Msg("Error response JSON encoding failed")
		}

		// Fallback plain text response (JSON encode edilemezse)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Error'u logla
	logError(r, statusCode, message, response.RequestID)
}

// Convenience functions
func ErrorHandlingMiddlewareWithDefaults() func(http.Handler) http.Handler {
	return ErrorHandlingMiddleware(errors.DefaultErrorConfig())
}

func ErrorHandlingMiddlewareForDevelopment() func(http.Handler) http.Handler {
	return ErrorHandlingMiddleware(errors.DevelopmentErrorConfig())
}

func ErrorHandlingMiddlewareForProduction() func(http.Handler) http.Handler {
	return ErrorHandlingMiddleware(errors.ProductionErrorConfig())
}
