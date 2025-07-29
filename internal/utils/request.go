package utils

import (
	"net/http"
	"strings"
)

// GetClientIP gerçek client IP'sini alır (proxy, load balancer desteği ile)
func GetClientIP(r *http.Request) string {
	// X-Forwarded-For header'ını kontrol et (load balancer/proxy)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// İlk IP'yi al (chain'deki ilk IP gerçek client)
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// X-Real-IP header'ını kontrol et
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Cloudflare IP
	cfIP := r.Header.Get("CF-Connecting-IP")
	if cfIP != "" {
		return cfIP
	}

	// RemoteAddr'yi kullan (son çare)
	ip := r.RemoteAddr
	if strings.Contains(ip, ":") {
		// Port'u kaldır
		parts := strings.Split(ip, ":")
		if len(parts) > 0 {
			return parts[0]
		}
	}
	return ip
}
