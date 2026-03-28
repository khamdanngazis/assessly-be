package domain

import "fmt"

// ErrNotFound represents a resource not found error
type ErrNotFound struct {
	Resource string
	ID       string
}

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("%s not found: %s", e.Resource, e.ID)
}

// ErrUnauthorized represents an authorization error
type ErrUnauthorized struct {
	Message string
}

func (e ErrUnauthorized) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "unauthorized access"
}

// ErrValidation represents a validation error
type ErrValidation struct {
	Field   string
	Message string
}

func (e ErrValidation) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation error on field '%s': %s", e.Field, e.Message)
	}
	return fmt.Sprintf("validation error: %s", e.Message)
}

// ErrConflict represents a conflict error (e.g., duplicate resource)
type ErrConflict struct {
	Resource string
	Message  string
}

func (e ErrConflict) Error() string {
	return fmt.Sprintf("conflict on %s: %s", e.Resource, e.Message)
}

// ErrInternal represents an internal server error
type ErrInternal struct {
	Message string
	Err     error
}

func (e ErrInternal) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("internal error: %s: %v", e.Message, e.Err)
	}
	return fmt.Sprintf("internal error: %s", e.Message)
}

func (e ErrInternal) Unwrap() error {
	return e.Err
}
