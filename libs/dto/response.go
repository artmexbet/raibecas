package dto

//go:generate easyjson -all response.go

// StandardResponse represents a standard API response wrapper
//
//easyjson:json
type StandardResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// ErrorCode represents error codes used across services
type ErrorCode string

const (
	ErrCodeInvalidRequest ErrorCode = "invalid_request"
	ErrCodeNotFound       ErrorCode = "not_found"
	ErrCodeInternal       ErrorCode = "internal_error"
	ErrCodeUnauthorized   ErrorCode = "unauthorized"
	ErrCodeForbidden      ErrorCode = "forbidden"
)

// ErrorResponse represents a standard error response
//
//easyjson:json
type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}
