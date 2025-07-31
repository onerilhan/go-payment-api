package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/onerilhan/go-payment-api/internal/middleware/errors"
)

// getErrorMessage status code'a göre custom error mesajı alır
func getErrorMessage(statusCode int, config *errors.ErrorConfig) string {
	if customMessage, exists := config.CustomErrorMap[statusCode]; exists {
		return customMessage
	}

	// Default messages
	switch statusCode {
	case 400:
		return "Bad Request"
	case 401:
		return "Unauthorized"
	case 403:
		return "Forbidden"
	case 404:
		return "Not Found"
	case 409:
		return "Conflict"
	case 429:
		return "Too Many Requests"
	case 500:
		return "Internal Server Error"
	case 503:
		return "Service Unavailable"
	default:
		return fmt.Sprintf("HTTP Error %d", statusCode)
	}
}

// getClientIP client IP'sini alır
func getClientIP(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		return strings.Split(xff, ",")[0]
	}

	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	return strings.Split(r.RemoteAddr, ":")[0]
}

// contains slice'da item var mı kontrol eder
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// truncateString string'i belirtilen uzunlukta keser
func truncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength-3] + "..."
}
