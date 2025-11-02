package domain

import "errors"

var (
	// Authentication errors
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrUserNotFound         = errors.New("user not found")
	ErrUserNotActive        = errors.New("user is not active")
	ErrInvalidToken         = errors.New("invalid token")
	ErrExpiredToken         = errors.New("token has expired")
	ErrTokenNotFound        = errors.New("token not found")

	// Registration errors
	ErrUsernameAlreadyExists = errors.New("username already exists")
	ErrEmailAlreadyExists    = errors.New("email already exists")
	ErrInvalidEmail          = errors.New("invalid email format")
	ErrInvalidPassword       = errors.New("password does not meet requirements")
	ErrRegistrationNotFound  = errors.New("registration request not found")
	ErrRegistrationNotPending = errors.New("registration request is not pending")

	// General errors
	ErrInternalServer = errors.New("internal server error")
)
