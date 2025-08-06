package validation

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// Pre-compiled security patterns for performance
var (
	sqlPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(\bor\b|\band\b)\s+\d+\s*=\s*\d+`),
		regexp.MustCompile(`(?i)union\s+select`),
		regexp.MustCompile(`(?i)drop\s+table`),
		regexp.MustCompile(`(?i)delete\s+from`),
		regexp.MustCompile(`(?i)insert\s+into`),
		regexp.MustCompile(`(?i)update\s+set`),
		regexp.MustCompile(`(?i)information_schema`),
		regexp.MustCompile(`(?i);--`),             // SQL comment
		regexp.MustCompile(`(?i)waitfor\s+delay`), // MSSQL time delay
		regexp.MustCompile(`(?i)-{2,}`),           // Multi-line comments
		regexp.MustCompile(`(?i)/\*.*?\*/`),       // Block comments
		regexp.MustCompile(`(?i)\bexec\b|\bexecute\b`),
		regexp.MustCompile(`(?i)\bsp_\w+`),
	}

	xssPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`),
		regexp.MustCompile(`(?i)javascript:`),
		regexp.MustCompile(`(?i)data:`),
		regexp.MustCompile(`(?i)on\w+\s*=`),
		regexp.MustCompile(`(?i)<iframe\b`),
		regexp.MustCompile(`(?i)<object\b`),
		regexp.MustCompile(`(?i)<embed\b`),
		regexp.MustCompile(`(?i)srcdoc\s*=`),
		regexp.MustCompile(`(?i)src\s*=\s*["']\s*javascript:`),
		regexp.MustCompile(`(?i)eval\s*\(`),
		regexp.MustCompile(`(?i)expression\s*\(`),
	}

	// Expanded suspicious bots list
	suspiciousBots = []string{
		"bot", "crawler", "spider", "scraper", "wget", "curl",
		"python-requests", "httpclient", "libwww-perl", "go-http-client",
	}
)

// ValidateSecurity performs security validation (SQL injection, XSS)
func ValidateSecurity(r *http.Request, config *Config) error {
	if config.SQLInjection {
		if err := detectSQLInjection(r, config.MaxParamSize); err != nil {
			return fmt.Errorf("SQL injection detected: %w", err)
		}
	}

	if config.XSSProtection {
		if err := detectXSS(r, config.MaxParamSize); err != nil {
			return fmt.Errorf("XSS attack detected: %w", err)
		}
	}

	return nil
}

// detectSQLInjection checks for malicious SQL patterns
func detectSQLInjection(r *http.Request, maxParamSize int) error {
	// Check query parameters
	for _, values := range r.URL.Query() {
		for _, param := range values {
			param = strings.TrimSpace(param)
			if param == "" {
				continue
			}
			if len(param) > maxParamSize {
				return fmt.Errorf("query parameter too long")
			}
			if detectMaliciousPatterns(param, sqlPatterns) {
				return fmt.Errorf("malicious SQL pattern in query parameter")
			}
		}
	}

	// Check form values only if content type is form-data
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "application/x-www-form-urlencoded") ||
		strings.HasPrefix(contentType, "multipart/form-data") {
		if err := r.ParseForm(); err == nil {
			for _, values := range r.Form {
				for _, value := range values {
					value = strings.TrimSpace(value)
					if value == "" {
						continue
					}
					if len(value) > maxParamSize {
						return fmt.Errorf("form parameter too long")
					}
					if detectMaliciousPatterns(value, sqlPatterns) {
						return fmt.Errorf("malicious SQL pattern in form data")
					}
				}
			}
		}
	}

	return nil
}

// detectXSS checks for malicious XSS patterns
func detectXSS(r *http.Request, maxParamSize int) error {
	// Check query parameters
	for _, values := range r.URL.Query() {
		for _, param := range values {
			param = strings.TrimSpace(param)
			if param == "" {
				continue
			}
			if len(param) > maxParamSize {
				return fmt.Errorf("query parameter too long")
			}
			if detectMaliciousPatterns(param, xssPatterns) {
				return fmt.Errorf("malicious XSS pattern in query parameter")
			}
		}
	}

	// Check suspicious headers
	suspiciousHeaders := []string{"Referer", "User-Agent"}
	for _, headerName := range suspiciousHeaders {
		headerValue := strings.TrimSpace(r.Header.Get(headerName))
		if headerValue == "" {
			continue
		}
		if len(headerValue) > maxParamSize {
			continue // Skip overly long headers
		}
		if detectMaliciousPatterns(headerValue, xssPatterns) {
			return fmt.Errorf("malicious XSS pattern in header: %s", headerName)
		}
	}

	return nil
}

// detectMaliciousPatterns checks a string against pre-compiled regex patterns
func detectMaliciousPatterns(input string, patterns []*regexp.Regexp) bool {
	for _, pattern := range patterns {
		if pattern.MatchString(input) {
			return true
		}
	}
	return false
}

// ValidateUserAgent basic User-Agent validation
func ValidateUserAgent(r *http.Request) error {
	userAgent := strings.TrimSpace(r.Header.Get("User-Agent"))
	if userAgent == "" {
		return fmt.Errorf("User-Agent header gerekli")
	}
	userAgentLower := strings.ToLower(userAgent)
	for _, bot := range suspiciousBots {
		if strings.Contains(userAgentLower, bot) {
			return fmt.Errorf("bot traffic detected")
		}
	}
	return nil
}

// ValidateReferer ensures the referer is from an allowed domain
func ValidateReferer(r *http.Request, allowedDomains []string) error {
	referer := strings.TrimSpace(r.Header.Get("Referer"))
	if referer == "" {
		return nil // Optional
	}
	refererURL, err := url.Parse(referer)
	if err != nil {
		return fmt.Errorf("invalid referer format")
	}
	host := strings.ToLower(refererURL.Hostname())
	for _, domain := range allowedDomains {
		domain = strings.ToLower(domain)
		if host == domain || strings.HasSuffix(host, "."+domain) {
			return nil
		}
	}
	return fmt.Errorf("referer domain not allowed")
}
