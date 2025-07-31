package middleware

import (
	"net/http"

	"github.com/rs/zerolog/log"

	"github.com/onerilhan/go-payment-api/internal/middleware/errors"
)

// logAPIError API error'ları özel olarak loglar
func logAPIError(err errors.APIError, r *http.Request, errorType string) {
	// Base log event
	logEvent := log.Warn().
		Str("error_type", errorType).
		Str("error_message", err.Error()).
		Int("status_code", err.Status()).
		Str("path", r.URL.Path).
		Str("method", r.Method).
		Str("client_ip", getClientIP(r))

	// Error type'ına göre ekstra bilgiler
	switch e := err.(type) {
	case *errors.AuthError:
		logEvent.Str("category", "authentication").
			Msg("Authentication failed")

	case *errors.RBACError:
		logEvent.Str("category", "authorization").
			Str("resource", e.Resource).
			Str("action", e.Action).
			Msg("Authorization failed")

	case *errors.ValidationError:
		logEvent.Str("category", "validation").
			Str("field", e.Field).
			Interface("value", e.Value).
			Msg("Validation failed")

	default:
		logEvent.Str("category", "api_error").
			Msg("API error occurred")
	}
}

// logPanic panic durumunu detaylı şekilde loglar
func logPanic(panicInfo *errors.PanicInfo, config *errors.ErrorConfig) {
	logEvent := log.Error().
		Str("type", "panic").
		Str("request_id", panicInfo.RequestID).
		Str("method", panicInfo.Method).
		Str("path", panicInfo.Path).
		Str("client_ip", panicInfo.ClientIP).
		Str("user_agent", panicInfo.UserAgent).
		Time("timestamp", panicInfo.Timestamp).
		Interface("panic_value", panicInfo.Value)

	// Stack trace'i logla (config'e göre)
	if config.EnablePanicLogs {
		logEvent.Str("stack_trace", panicInfo.Stack)
	}

	logEvent.Msg("Server panic occurred")

	// Critical error için ayrı bir log da at - ancak FATAL değil!
	log.Error().
		Interface("panic", panicInfo.Value).
		Str("path", panicInfo.Path).
		Str("level", "CRITICAL").
		Msg("CRITICAL: Server panic recovered - requires immediate attention")
}

// logError error'u uygun seviyede loglar
func logError(r *http.Request, statusCode int, message string, requestID string) {
	logEvent := log.With().
		Str("request_id", requestID).
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Str("client_ip", getClientIP(r)).
		Int("status_code", statusCode).
		Str("error", message).
		Logger()

	// Status code'a göre log level'ı belirle
	switch {
	case statusCode >= 500:
		logEvent.Error().Msg("Server error occurred")
	case statusCode >= 400:
		logEvent.Warn().Msg("Client error occurred")
	default:
		logEvent.Info().Msg("Request processed with error")
	}
}
