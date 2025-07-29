package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/onerilhan/go-payment-api/internal/utils"
)

// ResponseWriter wrapper to capture response data
type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	responseSize int64
}

// WriteHeader captures status code
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Write captures response size
func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.responseSize += int64(size)
	return size, err
}

// LoggingConfig logging middleware ayarları
type LoggingConfig struct {
	SkipPaths   []string // Log'lanmayacak path'ler (health check gibi)
	LogBody     bool     // Request/response body'leri logla
	MaxBodySize int64    // Maksimum body size (byte)
}

// DefaultLoggingConfig varsayılan logging ayarları
func DefaultLoggingConfig() *LoggingConfig {
	return &LoggingConfig{
		SkipPaths: []string{
			"/health",
			"/favicon.ico",
		},
		LogBody:     false, // Production'da false olmalı (security)
		MaxBodySize: 1024,  // 1KB
	}
}

// RequestLoggingMiddleware HTTP isteklerini loglar
func RequestLoggingMiddleware(config *LoggingConfig) func(http.Handler) http.Handler {
	// Config nil ise default kullan
	if config == nil {
		config = DefaultLoggingConfig()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip paths kontrolü
			if shouldSkipLogging(r.URL.Path, config.SkipPaths) {
				next.ServeHTTP(w, r)
				return
			}

			// Request başlangıç zamanı
			startTime := time.Now()

			// Response writer wrapper'ı oluştur
			wrapped := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK, // Default 200
				responseSize:   0,
			}

			// Request bilgilerini al
			method := r.Method
			path := r.URL.Path
			query := r.URL.RawQuery
			userAgent := r.Header.Get("User-Agent")
			clientIP := utils.GetClientIP(r)
			requestSize := r.ContentLength

			// Request ID oluştur (tracking için)
			requestID := generateRequestID()

			// Request ID'yi header'a ekle
			wrapped.Header().Set("X-Request-ID", requestID)

			// Request başlangıç log'u
			logEvent := log.Info().
				Str("request_id", requestID).
				Str("method", method).
				Str("path", path).
				Str("client_ip", clientIP).
				Str("user_agent", userAgent).
				Int64("request_size", requestSize)

			if query != "" {
				logEvent.Str("query", query)
			}

			logEvent.Msg("Request started")

			// Handler'ı çalıştır
			next.ServeHTTP(wrapped, r)

			// Response süresi hesapla
			duration := time.Since(startTime)

			// Response log'u
			responseLogEvent := log.Info().
				Str("request_id", requestID).
				Str("method", method).
				Str("path", path).
				Str("client_ip", clientIP).
				Int("status_code", wrapped.statusCode).
				Int64("response_size", wrapped.responseSize).
				Dur("duration", duration).
				Float64("duration_ms", float64(duration.Nanoseconds())/1e6)

			// Status code'a göre log level'ı ayarla
			switch {
			case wrapped.statusCode >= 500:
				responseLogEvent = log.Error().
					Str("request_id", requestID).
					Str("method", method).
					Str("path", path).
					Str("client_ip", clientIP).
					Int("status_code", wrapped.statusCode).
					Int64("response_size", wrapped.responseSize).
					Dur("duration", duration).
					Float64("duration_ms", float64(duration.Nanoseconds())/1e6)
			case wrapped.statusCode >= 400:
				responseLogEvent = log.Warn().
					Str("request_id", requestID).
					Str("method", method).
					Str("path", path).
					Str("client_ip", clientIP).
					Int("status_code", wrapped.statusCode).
					Int64("response_size", wrapped.responseSize).
					Dur("duration", duration).
					Float64("duration_ms", float64(duration.Nanoseconds())/1e6)
			}

			responseLogEvent.Msg("Request completed")
		})
	}
}

// shouldSkipLogging belirli path'lerin log'lanmaması gerekip gerekmediğini kontrol eder
func shouldSkipLogging(path string, skipPaths []string) bool {
	for _, skipPath := range skipPaths {
		if path == skipPath {
			return true
		}
		// Wildcard pattern matching
		if strings.HasSuffix(skipPath, "*") {
			prefix := strings.TrimSuffix(skipPath, "*")
			if strings.HasPrefix(path, prefix) {
				return true
			}
		}
	}
	return false
}

// generateRequestID benzersiz request ID oluşturur
func generateRequestID() string {
	return uuid.New().String()
}

// RequestLoggingMiddlewareWithDefaults varsayılan ayarlarla logging middleware döner
func RequestLoggingMiddlewareWithDefaults() func(http.Handler) http.Handler {
	return RequestLoggingMiddleware(DefaultLoggingConfig())
}

// ProductionLoggingConfig production için optimize edilmiş ayarlar
func ProductionLoggingConfig() *LoggingConfig {
	return &LoggingConfig{
		SkipPaths: []string{
			"/health",
			"/metrics", // Prometheus metrics
			"/favicon.ico",
			"/robots.txt",
		},
		LogBody:     false, // Güvenlik için kapalı
		MaxBodySize: 0,     // Body logging kapalı
	}
}
