package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/onerilhan/go-payment-api/internal/auth"
	"github.com/rs/zerolog/log"
)

// ContextKey middleware'de context için key tipi
type ContextKey string

const UserContextKey ContextKey = "user"

// AuthMiddleware JWT token kontrolü yapar (Gorilla Mux için middleware)
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Authorization header'ını al
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			log.Warn().
				Str("path", r.URL.Path).
				Str("method", r.Method).
				Msg("Authorization header eksik")
			http.Error(w, "Authorization header gerekli", http.StatusUnauthorized)
			return
		}

		// "Bearer " prefix'ini kontrol et
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			log.Warn().
				Str("path", r.URL.Path).
				Str("auth_header", authHeader).
				Msg("Geçersiz Authorization format")
			http.Error(w, "Authorization format: 'Bearer <token>'", http.StatusUnauthorized)
			return
		}

		// Token'ı al
		tokenString := tokenParts[1]

		// Token'ı doğrula
		claims, err := auth.ValidateToken(tokenString)
		if err != nil {
			log.Warn().
				Err(err).
				Str("path", r.URL.Path).
				Msg("Token doğrulama başarısız")
			http.Error(w, "Geçersiz token", http.StatusUnauthorized)
			return
		}

		// User bilgilerini context'e ekle
		ctx := context.WithValue(r.Context(), UserContextKey, claims)
		r = r.WithContext(ctx)

		log.Debug().
			Int("user_id", claims.UserID).
			Str("email", claims.Email).
			Str("path", r.URL.Path).
			Str("method", r.Method).
			Msg("🔐 Authentication successful")

		// Sonraki handler'a geç
		next.ServeHTTP(w, r)
	})
}
