package models

//go:generate easyjson -all models.go

// ChatRequest represents a request for chat processing.
type ChatRequest struct {
	UserID string `json:"user_id"`
	Input  string `json:"input"`
}
