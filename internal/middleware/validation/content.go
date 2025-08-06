// internal/middleware/validation/content.go
package validation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ValidateContent content validation (JSON, Content-Type, Content-Length)
func ValidateContent(r *http.Request, config *Config) error {
	// Content-Length validation
	if err := validateContentLength(r, config.MaxBodySize); err != nil {
		return err
	}

	// Content-Type validation (sadece POST/PUT/PATCH için)
	if r.Method == "POST" || r.Method == "PUT" || r.Method == "PATCH" {
		if err := validateContentType(r, config.ContentTypes); err != nil {
			return err
		}
	}

	// JSON body validation (eğer JSON ise)
	if config.JSONValidation && isJSONRequest(r) {
		if err := validateJSONBody(r, config.RequireNonEmptyJSON); err != nil {
			return err
		}
	}

	return nil
}

// validateContentLength content length'i doğrular (chunked transfer desteği)
func validateContentLength(r *http.Request, maxSize int64) error {
	// Transfer-Encoding: chunked durumunda ContentLength -1 olur
	if r.ContentLength == -1 {
		transferEncoding := r.Header.Get("Transfer-Encoding")
		if strings.Contains(strings.ToLower(transferEncoding), "chunked") {
			return nil // Chunked transfer'a izin ver
		}
		return fmt.Errorf("Content-Length header gerekli")
	}

	if r.ContentLength > maxSize {
		return fmt.Errorf("request body çok büyük. Maksimum boyut: %d bytes", maxSize)
	}
	return nil
}

// validateContentType content type'ı doğrular
func validateContentType(r *http.Request, allowedTypes []string) error {
	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		return fmt.Errorf("Content-Type header gerekli")
	}

	// Content-Type'ın başlangıcını kontrol et (charset parametresi olabilir)
	for _, allowedType := range allowedTypes {
		if strings.HasPrefix(contentType, allowedType) {
			return nil
		}
	}

	return fmt.Errorf("desteklenmeyen Content-Type: %s. İzin verilen tipler: %s",
		contentType, strings.Join(allowedTypes, ", "))
}

// isJSONRequest JSON request mi kontrol eder
func isJSONRequest(r *http.Request) bool {
	contentType := r.Header.Get("Content-Type")
	return strings.HasPrefix(contentType, "application/json")
}

// validateJSONBody JSON body'nin valid olup olmadığını kontrol eder
func validateJSONBody(r *http.Request, requireNonEmpty bool) error {
	if r.Body == nil {
		if requireNonEmpty {
			return fmt.Errorf("JSON body gerekli")
		}
		return nil
	}

	// Body'yi oku
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("request body okunamadı: %w", err)
	}

	// Body'yi geri yerine koy (middleware chain için)
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	// Boş body kontrolü
	if len(bodyBytes) == 0 {
		if requireNonEmpty {
			return fmt.Errorf("JSON body boş olamaz")
		}
		return nil
	}

	// JSON parse et
	var jsonData interface{}
	if err := json.Unmarshal(bodyBytes, &jsonData); err != nil {
		return fmt.Errorf("geçersiz JSON formatı: %w", err)
	}

	return nil
}

// ValidateHeaders required headers kontrolü
func ValidateHeaders(r *http.Request, requiredHeaders []string) error {
	for _, header := range requiredHeaders {
		if r.Header.Get(header) == "" {
			return fmt.Errorf("gerekli header eksik: %s", header)
		}
	}
	return nil
}

// ValidateMethod HTTP methodunu doğrular
func ValidateMethod(r *http.Request, allowedMethods []string) error {
	for _, method := range allowedMethods {
		if r.Method == method {
			return nil
		}
	}
	return fmt.Errorf("HTTP method '%s' desteklenmiyor. İzin verilen metodlar: %s",
		r.Method, strings.Join(allowedMethods, ", "))
}
