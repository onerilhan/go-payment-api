package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

// CORSConfig CORS middleware ayarları
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

// DefaultCORSConfig development için varsayılan CORS ayarları
func DefaultCORSConfig() *CORSConfig {
	return &CORSConfig{
		AllowedOrigins: []string{
			"http://localhost:3000", // React/Vue/Angular development
			"http://localhost:3001", // Alternatif port
			"http://127.0.0.1:3000", // IP bazlı erişim
			"http://127.0.0.1:3001",
		},
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowedHeaders: []string{
			"Authorization",
			"Content-Type",
			"Accept",
			"Origin",
			"User-Agent",
			"X-Requested-With",
		},
		ExposedHeaders: []string{
			"Content-Length",
			"Content-Type",
		},
		AllowCredentials: true,
		MaxAge:           86400, // 24 saat
	}
}

// ProductionCORSConfig production için güvenli CORS ayarları
func ProductionCORSConfig(allowedDomains []string) *CORSConfig {
	return &CORSConfig{
		AllowedOrigins: allowedDomains, // Sadece belirtilen domain'ler
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
		},
		AllowedHeaders: []string{
			"Authorization",
			"Content-Type",
			"Accept",
		},
		ExposedHeaders:   []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           3600, // 1 saat
	}
}

// CORSMiddleware CORS header'larını ayarlayan middleware
func CORSMiddleware(config *CORSConfig) func(http.Handler) http.Handler {
	// Config nil ise default kullan
	if config == nil {
		config = DefaultCORSConfig()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Origin kontrolü ve header set etme
			if origin != "" && isAllowedOrigin(origin, config.AllowedOrigins) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}

			// Methods
			if len(config.AllowedMethods) > 0 {
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ", "))
			}

			// Headers
			if len(config.AllowedHeaders) > 0 {
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ", "))
			}

			// Exposed Headers
			if len(config.ExposedHeaders) > 0 {
				w.Header().Set("Access-Control-Expose-Headers", strings.Join(config.ExposedHeaders, ", "))
			}

			// Credentials
			if config.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			// Max Age
			if config.MaxAge > 0 {
				w.Header().Set("Access-Control-Max-Age", strconv.Itoa(config.MaxAge))
			}

			// Preflight request (OPTIONS method) handling
			if r.Method == http.MethodOptions {
				log.Debug().
					Str("origin", origin).
					Str("method", r.Header.Get("Access-Control-Request-Method")).
					Str("headers", r.Header.Get("Access-Control-Request-Headers")).
					Msg("CORS preflight request handled")

				w.WriteHeader(http.StatusNoContent)
				return
			}

			// Debug log for development
			if origin != "" {
				log.Debug().
					Str("origin", origin).
					Str("method", r.Method).
					Str("path", r.URL.Path).
					Msg("CORS headers applied")
			}

			// Sonraki handler'a geç
			next.ServeHTTP(w, r)
		})
	}
}

// isAllowedOrigin origin'in izin verilen listede olup olmadığını kontrol eder
func isAllowedOrigin(origin string, allowedOrigins []string) bool {
	for _, allowedOrigin := range allowedOrigins {
		if allowedOrigin == origin {
			return true
		}
		// Wildcard pattern matching (*.domain.com gibi)
		if strings.HasPrefix(allowedOrigin, "*.") {
			domain := strings.TrimPrefix(allowedOrigin, "*.")
			if strings.HasSuffix(origin, "."+domain) || origin == domain {
				return true
			}
		}
	}
	return false
}

// CORSMiddlewareWithDefaults varsayılan ayarlarla CORS middleware döner
func CORSMiddlewareWithDefaults() func(http.Handler) http.Handler {
	return CORSMiddleware(DefaultCORSConfig())
}
