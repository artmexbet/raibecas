package models

//go:generate easyjson -all models.go

// ChatRequest represents a request for chat processing.
type ChatRequest struct {
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id,omitempty"`
	Input     string `json:"input"`
}
