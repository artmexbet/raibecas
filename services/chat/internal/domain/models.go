package domain

import "time"

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatResponse struct {
	Done      bool      `json:"done"`
	Message   *Message  `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

type Document struct {
	ID       string                 `json:"id"`
	Metadata map[string]interface{} `json:"metadata"`
}
