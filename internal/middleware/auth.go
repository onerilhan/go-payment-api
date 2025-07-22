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

// AuthMiddleware JWT token kontrolü yapar
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Authorization header'ını al
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			log.Warn().Msg("Authorization header eksik")
			http.Error(w, "Authorization header gerekli", http.StatusUnauthorized)
			return
		}

		// "Bearer " prefix'ini kontrol et
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			log.Warn().Msg("Geçersiz Authorization format")
			http.Error(w, "Authorization format: 'Bearer <token>'", http.StatusUnauthorized)
			return
		}

		// Token'ı al
		tokenString := tokenParts[1]

		// Token'ı doğrula
		claims, err := auth.ValidateToken(tokenString)
		if err != nil {
			log.Warn().Err(err).Msg("Token doğrulama başarısız")
			http.Error(w, "Geçersiz token", http.StatusUnauthorized)
			return
		}

		// User bilgilerini context'e ekle
		ctx := context.WithValue(r.Context(), UserContextKey, claims)
		r = r.WithContext(ctx)

		// Sonraki handler'a geç
		next.ServeHTTP(w, r)
	}
}
