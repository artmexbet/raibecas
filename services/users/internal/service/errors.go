package service

import "errors"

var (
	// ErrNotFound is returned when a requested resource is not found
	ErrNotFound = errors.New("not found")

	// ErrInvalidStatus is returned when an invalid status is provided
	ErrInvalidStatus = errors.New("invalid status")

	// ErrInvalidUserID is returned when user ID is invalid (nil)
	ErrInvalidUserID = errors.New("invalid user id")

	// ErrRegistrationRequestNil is returned when registration request is nil
	ErrRegistrationRequestNil = errors.New("registration request cannot be nil")

	// ErrMissingRequiredFields is returned when required fields are missing
	ErrMissingRequiredFields = errors.New("missing required fields")

	// ErrInvalidRequestOrApproverID is returned when request ID or approver ID is invalid
	ErrInvalidRequestOrApproverID = errors.New("invalid request or approver id")
)
