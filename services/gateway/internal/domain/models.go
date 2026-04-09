package domain

import (
	"time"

	"github.com/google/uuid"
)

//go:generate easyjson -all models.go

type Role string

const (
	RoleSuperAdmin Role = "SuperAdmin"
	RoleAdmin      Role = "Admin"
	RoleUser       Role = "User"
)

type Additional struct {
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Admin struct {
	ID       uuid.UUID `json:"id" validate:"required,uuid"`
	Email    string    `json:"email" validate:"required,email"`
	Username string    `json:"username" validate:"required"`
	Role     Role      `json:"role" validate:"required"`
	Additional
}

type User struct {
	ID           uuid.UUID `json:"id" validate:"required,uuid"`
	Email        string    `json:"email" validate:"required,email"`
	Username     string    `json:"username" validate:"required"`
	FullName     string    `json:"full_name" validate:"required"`
	RegisteredAt time.Time `json:"registered_at" validate:"required"`
	LastLoginAt  time.Time `json:"last_login_at" validate:"required"`
	IsActive     bool      `json:"is_active"`
}

type Author struct {
	ID   uuid.UUID `json:"id" validate:"required,uuid"`
	Name string    `json:"name" validate:"required"`
}

type Category struct {
	ID    int    `json:"id" validate:"required,number"`
	Title string `json:"title" validate:"required"`
}

type Tag struct {
	ID    int    `json:"id" validate:"required,number"`
	Title string `json:"title" validate:"required"`
}

type DocumentType struct {
	ID   int    `json:"id" validate:"required,number"`
	Name string `json:"name" validate:"required"`
}

type AuthorshipType struct {
	ID    int    `json:"id" validate:"required,number"`
	Title string `json:"title" validate:"required"`
}

type DocumentParticipant struct {
	Author         Author         `json:"author" validate:"dive"`
	AuthorshipType AuthorshipType `json:"authorshipType" validate:"dive"`
}

type DocumentParticipantRef struct {
	AuthorID string `json:"authorId" validate:"required,uuid"`
	TypeID   int    `json:"typeId" validate:"required,min=1"`
}

type Document struct {
	ID              uuid.UUID             `json:"id" validate:"required,uuid"`
	Title           string                `json:"title" validate:"required"`
	Description     *string               `json:"description" validate:"-"`
	Author          Author                `json:"author" validate:"dive"`
	Category        Category              `json:"category" validate:"dive"`
	DocumentType    *DocumentType         `json:"documentType,omitempty" validate:"omitempty,dive"`
	Participants    []DocumentParticipant `json:"participants,omitempty" validate:"dive"`
	PublicationDate time.Time             `json:"publication_date" validate:"required"`
	Tags            []Tag                 `json:"tags" validate:"dive"`
	Content         *string               `json:"content,omitempty"`
	CoverURL        *string               `json:"cover_url,omitempty"`
	Additional
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error   string            `json:"error"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}
