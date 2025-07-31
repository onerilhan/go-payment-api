package errors

import "time"

// ErrorResponse standardized error response formatı
type ErrorResponse struct {
	Success   bool                   `json:"success"`
	Error     string                 `json:"error"`
	Code      int                    `json:"code"`
	Timestamp string                 `json:"timestamp"`
	RequestID string                 `json:"request_id,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Stack     string                 `json:"stack,omitempty"` // Sadece development'ta
}

// PanicInfo panic durumu hakkında bilgi
type PanicInfo struct {
	Value     interface{}
	Stack     string
	RequestID string
	Method    string
	Path      string
	UserAgent string
	ClientIP  string
	Timestamp time.Time
}
