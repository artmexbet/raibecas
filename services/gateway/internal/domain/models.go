package domain

import (
	"time"

	"github.com/google/uuid"
)

//go:generate easyjson -all models.go

type Role string

const (
	RoleSuperAdmin Role = "super_admin"
	RoleAdmin      Role = "admin"
)

type Additional struct {
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
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
	FullName     string    `json:"fullName" validate:"required"`
	RegisteredAt time.Time `json:"registeredAt" validate:"required"`
	LastLoginAt  time.Time `json:"lastLoginAt" validate:"required"`
	IsActive     bool      `json:"isActive"`
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

type Document struct {
	ID          uuid.UUID `json:"id" validate:"required,uuid"`
	Title       string    `json:"title" validate:"required"`
	Description *string   `json:"description" validate:"-"`
	Author      Author    `json:"author" validate:"dive"`

	Category        Category  `json:"category" validate:"dive"`
	PublicationDate time.Time `json:"publicationDate" validate:"required"`
	Tags            []Tag     `json:"tags" validate:"dive"`
	Additional
}
