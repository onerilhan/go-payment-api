package middleware

import (
	"fmt"
	"net/http"

	"github.com/rs/zerolog/log"
)

// SecurityConfig security headers ayarları
type SecurityConfig struct {
	// Content Security Policy
	ContentSecurityPolicy string

	// HTTP Strict Transport Security (HSTS)
	HSTSMaxAge            int
	HSTSIncludeSubdomains bool
	HSTSPreload           bool

	// X-Frame-Options
	FrameOptions string // DENY, SAMEORIGIN, ALLOW-FROM uri

	// X-Content-Type-Options
	ContentTypeNosniff bool

	// X-XSS-Protection
	XSSProtection string // 0, 1, 1; mode=block

	// Referrer-Policy
	ReferrerPolicy string

	// Additional custom headers
	CustomHeaders map[string]string
}

// DefaultSecurityConfig varsayılan güvenlik ayarları
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		ContentSecurityPolicy: "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:",
		HSTSMaxAge:            31536000, // 1 yıl
		HSTSIncludeSubdomains: true,
		HSTSPreload:           false, // Preload listesi için manual submission gerekli
		FrameOptions:          "DENY",
		ContentTypeNosniff:    true,
		XSSProtection:         "1; mode=block",
		ReferrerPolicy:        "strict-origin-when-cross-origin",
		CustomHeaders:         make(map[string]string),
	}
}

// ProductionSecurityConfig production için sıkı güvenlik ayarları
func ProductionSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		ContentSecurityPolicy: "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self'; connect-src 'self'; font-src 'self'; object-src 'none'; media-src 'self'; frame-src 'none';",
		HSTSMaxAge:            63072000, // 2 yıl
		HSTSIncludeSubdomains: true,
		HSTSPreload:           true,
		FrameOptions:          "DENY",
		ContentTypeNosniff:    true,
		XSSProtection:         "1; mode=block",
		ReferrerPolicy:        "no-referrer",
		CustomHeaders: map[string]string{
			"X-Permitted-Cross-Domain-Policies": "none",
			"X-Download-Options":                "noopen",
		},
	}
}

// DevelopmentSecurityConfig development için esnek güvenlik ayarları
func DevelopmentSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		ContentSecurityPolicy: "default-src 'self' 'unsafe-inline' 'unsafe-eval'; img-src 'self' data: https: http:",
		HSTSMaxAge:            0, // Development'ta HSTS kapalı (HTTP kullanımı için)
		HSTSIncludeSubdomains: false,
		HSTSPreload:           false,
		FrameOptions:          "SAMEORIGIN", // Development tools için daha esnek
		ContentTypeNosniff:    true,
		XSSProtection:         "1; mode=block",
		ReferrerPolicy:        "strict-origin-when-cross-origin",
		CustomHeaders:         make(map[string]string),
	}
}

// SecurityHeadersMiddleware güvenlik header'larını ekler
func SecurityHeadersMiddleware(config *SecurityConfig) func(http.Handler) http.Handler {
	// Config nil ise default kullan
	if config == nil {
		config = DefaultSecurityConfig()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Content Security Policy
			if config.ContentSecurityPolicy != "" {
				w.Header().Set("Content-Security-Policy", config.ContentSecurityPolicy)
			}

			// HTTP Strict Transport Security (HSTS)
			if config.HSTSMaxAge > 0 {
				hstsValue := formatHSTSHeader(config.HSTSMaxAge, config.HSTSIncludeSubdomains, config.HSTSPreload)
				w.Header().Set("Strict-Transport-Security", hstsValue)
			}

			// X-Frame-Options
			if config.FrameOptions != "" {
				w.Header().Set("X-Frame-Options", config.FrameOptions)
			}

			// X-Content-Type-Options
			if config.ContentTypeNosniff {
				w.Header().Set("X-Content-Type-Options", "nosniff")
			}

			// X-XSS-Protection
			if config.XSSProtection != "" {
				w.Header().Set("X-XSS-Protection", config.XSSProtection)
			}

			// Referrer-Policy
			if config.ReferrerPolicy != "" {
				w.Header().Set("Referrer-Policy", config.ReferrerPolicy)
			}

			// Custom headers
			for key, value := range config.CustomHeaders {
				w.Header().Set(key, value)
			}

			// Debug log (sadece development'ta)
			log.Debug().
				Str("path", r.URL.Path).
				Msg("Security headers applied")

			// Sonraki handler'a geç
			next.ServeHTTP(w, r)
		})
	}
}

// formatHSTSHeader HSTS header değerini formatlar
func formatHSTSHeader(maxAge int, includeSubdomains, preload bool) string {
	hsts := fmt.Sprintf("max-age=%d", maxAge)

	if includeSubdomains {
		hsts += "; includeSubDomains"
	}

	if preload {
		hsts += "; preload"
	}

	return hsts
}

// SecurityHeadersMiddlewareWithDefaults varsayılan ayarlarla security middleware döner
func SecurityHeadersMiddlewareWithDefaults() func(http.Handler) http.Handler {
	return SecurityHeadersMiddleware(DefaultSecurityConfig())
}

// SecurityHeadersMiddlewareForProduction production ayarlarla security middleware döner
func SecurityHeadersMiddlewareForProduction() func(http.Handler) http.Handler {
	return SecurityHeadersMiddleware(ProductionSecurityConfig())
}

// SecurityHeadersMiddlewareForDevelopment development ayarlarla security middleware döner
func SecurityHeadersMiddlewareForDevelopment() func(http.Handler) http.Handler {
	return SecurityHeadersMiddleware(DevelopmentSecurityConfig())
}
