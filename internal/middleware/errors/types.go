package errors

// APIError interface for custom error types
type APIError interface {
	error
	Status() int
}

// AuthError authentication hatası için custom error type
type AuthError struct {
	Message    string
	StatusCode int
}

// Error AuthError'un error interface implementation'ı
func (e *AuthError) Error() string {
	return e.Message
}

// Status AuthError'un APIError interface implementation'ı
func (e *AuthError) Status() int {
	return e.StatusCode
}

// RBACError authorization hatası için custom error type
type RBACError struct {
	Message    string
	StatusCode int
	Resource   string
	Action     string
}

// Error RBACError'un error interface implementation'ı
func (e *RBACError) Error() string {
	return e.Message
}

// Status RBACError'un APIError interface implementation'ı
func (e *RBACError) Status() int {
	return e.StatusCode
}

// ValidationError validation hatası için custom error type
type ValidationError struct {
	Message    string
	StatusCode int
	Field      string
	Value      interface{}
}

// Error ValidationError'un error interface implementation'ı
func (e *ValidationError) Error() string {
	return e.Message
}

// Status ValidationError'un APIError interface implementation'ı
func (e *ValidationError) Status() int {
	return e.StatusCode
}
