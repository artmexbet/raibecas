package domain

import "errors"

var (
	// ErrNotFound is returned when a requested resource is not found
	ErrNotFound = errors.New("not found")

	// ErrInvalidInput is returned when input validation fails
	ErrInvalidInput = errors.New("invalid input")

	// ErrStorageFailure is returned when storage operation fails
	ErrStorageFailure = errors.New("storage failure")
)
