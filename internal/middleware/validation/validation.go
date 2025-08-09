// internal/middleware/validation/validation.go
package validation

import (
	"net/http"

	"github.com/rs/zerolog/log"

	"github.com/onerilhan/go-payment-api/internal/middleware/errors"
)

// Config validation middleware ayarları
type Config struct {
	MaxBodySize         int64             // Maximum request body size (bytes)
	MaxParamSize        int               // Maximum parameter size for security checks
	RequiredHeaders     []string          // Required headers
	AllowedMethods      []string          // Allowed HTTP methods
	ContentTypes        []string          // Allowed content types
	JSONValidation      bool              // Enable JSON validation
	SQLInjection        bool              // Enable SQL injection detection
	XSSProtection       bool              // Enable XSS protection
	PathValidation      map[string]string // Path parameter validation rules
	RequireNonEmptyJSON bool              // Require non-empty JSON body for JSON requests
}

// DefaultConfig varsayılan validation ayarları
func DefaultConfig() *Config {
	return &Config{
		MaxBodySize:     1024 * 1024, // 1MB
		MaxParamSize:    2048,        // 2KB per parameter
		RequiredHeaders: []string{},
		AllowedMethods: []string{
			"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH", "HEAD",
		},
		ContentTypes: []string{
			"application/json",
			"application/x-www-form-urlencoded",
		},
		JSONValidation:      true,
		SQLInjection:        true,
		XSSProtection:       true,
		PathValidation:      make(map[string]string),
		RequireNonEmptyJSON: false,
	}
}

// StrictConfig API endpoints için sıkı validation
func StrictConfig() *Config {
	config := DefaultConfig()
	config.MaxBodySize = 512 * 1024 // 512KB
	config.MaxParamSize = 1024      // 1KB
	config.RequiredHeaders = []string{"Content-Type", "User-Agent"}
	config.RequireNonEmptyJSON = true
	config.PathValidation = map[string]string{
		"id":      "positive_integer",
		"user_id": "positive_integer",
	}
	return config
}

// Middleware ana validation middleware'i
func Middleware(config *Config) func(http.Handler) http.Handler {
	if config == nil {
		config = DefaultConfig()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			// OPTIONS requests bypass (CORS preflight)
			if r.Method == "OPTIONS" {
				log.Debug().
					Str("method", r.Method).
					Str("path", r.URL.Path).
					Msg("OPTIONS request bypassing validation")
				next.ServeHTTP(w, r)
				return
			}

			// 1. HTTP Method validation
			if err := ValidateMethod(r, config.AllowedMethods); err != nil {
				panic(&errors.ValidationError{
					Message:    err.Error(),
					StatusCode: http.StatusMethodNotAllowed,
					Field:      "method",
					Value:      r.Method,
				})
			}

			// 2. Required Headers validation
			if err := ValidateHeaders(r, config.RequiredHeaders); err != nil {
				panic(&errors.ValidationError{
					Message:    err.Error(),
					StatusCode: http.StatusBadRequest,
					Field:      "headers",
					Value:      "missing_required_header",
				})
			}

			// 3. Content validation (JSON, Content-Type, Content-Length)
			if err := ValidateContent(r, config); err != nil {
				panic(&errors.ValidationError{
					Message:    err.Error(),
					StatusCode: http.StatusBadRequest,
					Field:      "content",
					Value:      "content_validation_failed",
				})
			}

			// 4. Path parameter validation
			if err := ValidatePathParameters(r, config.PathValidation); err != nil {
				panic(&errors.ValidationError{
					Message:    err.Error(),
					StatusCode: http.StatusBadRequest,
					Field:      "path_parameter",
					Value:      "invalid_path_parameter",
				})
			}

			// 5. Security validation (SQL injection, XSS)
			if err := ValidateSecurity(r, config); err != nil {
				log.Warn().
					Str("client_ip", getClientIP(r)).
					Str("path", r.URL.Path).
					Err(err).
					Msg("Security threat detected")

				panic(&errors.ValidationError{
					Message:    "Güvenlik ihlali tespit edildi",
					StatusCode: http.StatusBadRequest,
					Field:      "security",
					Value:      "security_threat_detected",
				})
			}

			log.Debug().
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int64("content_length", r.ContentLength).
				Msg("Request validation passed")

			// Sonraki middleware'a geç
			next.ServeHTTP(w, r)
		})
	}
}

// MiddlewareWithDefaults varsayılan ayarlarla middleware döner
func MiddlewareWithDefaults() func(http.Handler) http.Handler {
	return Middleware(DefaultConfig())
}

// getClientIP helper function
func getClientIP(r *http.Request) string {
	// X-Forwarded-For header'ını kontrol et
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		return xff
	}

	// X-Real-IP header'ını kontrol et
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	return r.RemoteAddr
}
