package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
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
