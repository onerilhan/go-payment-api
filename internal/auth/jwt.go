package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog/log"
)

// JWT için secret key (production'da env'den okunmalı)
var jwtSecret = []byte("your-secret-key-change-this-in-production")

// Claims JWT payload'ını temsil eder
type Claims struct {
	UserID int    `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// GenerateToken kullanıcı için JWT token oluşturur
func GenerateToken(userID int, email string) (string, error) {
	// Token 24 saat geçerli olacak
	expirationTime := time.Now().Add(24 * time.Hour)

	// Claims oluştur
	claims := &Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	// Token oluştur
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Token'ı imzala ve string'e çevir
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", fmt.Errorf("token oluşturulamadı: %w", err)
	}

	return tokenString, nil
}

// ValidateToken JWT token'ını doğrular ve claims'i döner
func ValidateToken(tokenString string) (*Claims, error) {
	// Token'ı parse et
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Signing method kontrolü
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("beklenmeyen signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("token parse edilemedi: %w", err)
	}

	// Claims'i al
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("geçersiz token")
}

func RefreshToken(tokenString string) (string, int64, error) {
	// Token'ı parse et
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("beklenmeyen signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	// Token geçerliyse refresh gerekmiyor
	if err == nil && token.Valid {
		log.Warn().Msg("Token refresh denendi ama token hala geçerli")
		return "", 0, fmt.Errorf("token hala geçerli, refresh gerekmiyor")
	}

	// Token nil ise parse hatası
	if token == nil {
		log.Error().Err(err).Msg("Token parse edilemedi")
		return "", 0, fmt.Errorf("token parse edilemedi: %w", err)
	}

	// JWT v5 hataları
	if errors.Is(err, jwt.ErrTokenExpired) {
		claims, ok := token.Claims.(*Claims)
		if !ok {
			log.Error().Msg("Token claims alınamadı")
			return "", 0, fmt.Errorf("token claims alınamadı")
		}

		// Yeni token oluştur
		newToken, genErr := GenerateToken(claims.UserID, claims.Email)
		if genErr != nil {
			log.Error().Err(genErr).Msg("Yeni token oluşturulamadı")
			return "", 0, fmt.Errorf("yeni token oluşturulamadı: %w", genErr)
		}

		expiresIn := int64(24 * 60 * 60) // 24 saat
		log.Info().Int("user_id", claims.UserID).Msg("Token başarıyla refresh edildi")
		return newToken, expiresIn, nil
	}

	if errors.Is(err, jwt.ErrTokenMalformed) {
		log.Warn().Msg("Malformed token ile refresh denendi")
		return "", 0, fmt.Errorf("token malformed")
	}

	if errors.Is(err, jwt.ErrTokenSignatureInvalid) {
		log.Warn().Msg("Invalid signature ile refresh denendi")
		return "", 0, fmt.Errorf("token signature invalid")
	}

	// Diğer her şey
	log.Error().Err(err).Msg("Token refresh başarısız")
	return "", 0, fmt.Errorf("token refresh edilemedi: %w", err)
}
