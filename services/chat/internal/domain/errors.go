package domain

import "errors"

var (
	ErrDocumentWithoutContent = errors.New("document without content")
	ErrInvalidChatSessionID   = errors.New("invalid chat session id")
	ErrChatSessionNotFound    = errors.New("chat session not found")
)
