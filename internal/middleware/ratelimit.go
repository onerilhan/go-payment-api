package middleware

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"

	"github.com/onerilhan/go-payment-api/internal/utils"
)

// RateLimitConfig rate limiting ayarları
type RateLimitConfig struct {
	RequestsPerMinute int
	Burst             int
	WindowSize        time.Duration
	WhitelistIPs      []string
	BlacklistIPs      []string
	SkipPaths         []string
	CustomMessage     string
}

// DefaultRateLimitConfig varsayılan rate limit ayarları
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		RequestsPerMinute: 60,
		Burst:             10,
		WindowSize:        time.Minute,
		WhitelistIPs:      []string{},
		BlacklistIPs:      []string{},
		SkipPaths: []string{
			"/health",
			"/favicon.ico",
		},
		CustomMessage: "Rate limit exceeded. Please try again later.",
	}
}

// ipLimiter tek bir IP için rate limiter
type ipLimiter struct {
	limiter     *rate.Limiter
	lastSeen    time.Time
	windowStart time.Time
}

// RateLimitMiddleware rate limiting middleware
type RateLimitMiddleware struct {
	config   *RateLimitConfig
	limiters map[string]*ipLimiter
	mutex    sync.RWMutex
}

// NewRateLimitMiddleware yeni rate limit middleware oluşturur
func NewRateLimitMiddleware(config *RateLimitConfig) *RateLimitMiddleware {
	if config == nil {
		config = DefaultRateLimitConfig()
	}

	middleware := &RateLimitMiddleware{
		config:   config,
		limiters: make(map[string]*ipLimiter),
		mutex:    sync.RWMutex{},
	}

	go middleware.cleanupLimiters()

	return middleware
}

// Handler rate limiting middleware handler döner
func (rlm *RateLimitMiddleware) Handler() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if rlm.shouldSkipPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			clientIP := utils.GetClientIP(r)

			if rlm.isBlacklisted(clientIP) {
				log.Warn().Str("client_ip", clientIP).Msg("Request blocked - IP blacklisted")
				rlm.sendRateLimitResponse(w, "IP address is blacklisted", 403, 0, time.Now())
				return
			}

			if rlm.isWhitelisted(clientIP) {
				next.ServeHTTP(w, r)
				return
			}

			allowed, remaining, resetTime := rlm.checkRateLimit(clientIP)

			rlm.setRateLimitHeaders(w, remaining, resetTime)

			if !allowed {
				log.Warn().Str("client_ip", clientIP).Msg("Request blocked - rate limit exceeded")
				rlm.sendRateLimitResponse(w, rlm.config.CustomMessage, 429, remaining, resetTime)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// checkRateLimit IP'nin rate limit'ini kontrol eder
func (rlm *RateLimitMiddleware) checkRateLimit(ip string) (allowed bool, remaining int, resetTime time.Time) {
	rlm.mutex.Lock()
	defer rlm.mutex.Unlock()

	now := time.Now()

	limiter, exists := rlm.limiters[ip]
	if !exists {
		rateLimit := rate.Every(rlm.config.WindowSize / time.Duration(rlm.config.RequestsPerMinute))
		limiter = &ipLimiter{
			limiter:     rate.NewLimiter(rateLimit, rlm.config.Burst),
			lastSeen:    now,
			windowStart: now,
		}
		rlm.limiters[ip] = limiter
	}

	limiter.lastSeen = now

	if now.Sub(limiter.windowStart) >= rlm.config.WindowSize {
		limiter.windowStart = now
	}

	allowed = limiter.limiter.Allow()

	tokens := limiter.limiter.Tokens()
	remaining = int(tokens)
	if remaining < 0 {
		remaining = 0
	}

	resetTime = limiter.windowStart.Add(rlm.config.WindowSize)

	return allowed, remaining, resetTime
}

// setRateLimitHeaders rate limit header'larını set eder
func (rlm *RateLimitMiddleware) setRateLimitHeaders(w http.ResponseWriter, remaining int, resetTime time.Time) {
	w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rlm.config.RequestsPerMinute))
	w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
	w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))
	w.Header().Set("X-RateLimit-Window", rlm.config.WindowSize.String())
}

// shouldSkipPath path kontrolü
func (rlm *RateLimitMiddleware) shouldSkipPath(path string) bool {
	for _, skipPath := range rlm.config.SkipPaths {
		if path == skipPath {
			return true
		}
	}
	return false
}

// isWhitelisted whitelist kontrolü
func (rlm *RateLimitMiddleware) isWhitelisted(ip string) bool {
	for _, whiteIP := range rlm.config.WhitelistIPs {
		if ip == whiteIP {
			return true
		}
	}
	return false
}

// isBlacklisted blacklist kontrolü
func (rlm *RateLimitMiddleware) isBlacklisted(ip string) bool {
	for _, blackIP := range rlm.config.BlacklistIPs {
		if ip == blackIP {
			return true
		}
	}
	return false
}

// sendRateLimitResponse rate limit response
func (rlm *RateLimitMiddleware) sendRateLimitResponse(w http.ResponseWriter, message string, statusCode int, remaining int, resetTime time.Time) {
	w.Header().Set("Content-Type", "application/json")

	retryAfterSeconds := int(time.Until(resetTime).Seconds())
	if retryAfterSeconds < 0 {
		retryAfterSeconds = int(rlm.config.WindowSize.Seconds())
	}
	w.Header().Set("Retry-After", strconv.Itoa(retryAfterSeconds))

	w.WriteHeader(statusCode)

	response := map[string]interface{}{
		"success":             false,
		"error":               message,
		"code":                statusCode,
		"retry_after_seconds": retryAfterSeconds,
		"rate_limit": map[string]interface{}{
			"limit":     rlm.config.RequestsPerMinute,
			"remaining": remaining,
			"reset_at":  resetTime.Unix(),
			"window":    rlm.config.WindowSize.String(),
		},
	}

	json.NewEncoder(w).Encode(response)
}

// / cleanupLimiters eski limiter'ları temizler
func (rlm *RateLimitMiddleware) cleanupLimiters() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rlm.mutex.Lock()

		now := time.Now()
		for ip, limiter := range rlm.limiters {
			if now.Sub(limiter.lastSeen) > 30*time.Minute {
				delete(rlm.limiters, ip)
			}
		}

		log.Debug().Int("active_limiters", len(rlm.limiters)).Msg("Rate limiter cleanup completed")

		rlm.mutex.Unlock()
	}
}

// RateLimitMiddlewareWithDefaults varsayılan ayarlarla middleware döner
func RateLimitMiddlewareWithDefaults() func(http.Handler) http.Handler {
	middleware := NewRateLimitMiddleware(DefaultRateLimitConfig())
	return middleware.Handler()
}
