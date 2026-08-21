package domain

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound        = errors.New("not found")
	ErrConflict        = errors.New("version conflict")
	ErrForbidden       = errors.New("forbidden")
	ErrInvalidState    = errors.New("invalid state transition")
	ErrInvalidArgument = errors.New("invalid argument")
	ErrCanceled        = errors.New("operation canceled")
	ErrAlreadyExists   = errors.New("already exists")
)

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ValidationError struct {
	Message string
	Fields  []FieldError
}

func (e *ValidationError) Error() string {
	return e.Message
}

func NewValidationError(message string, fields ...FieldError) error {
	return &ValidationError{Message: message, Fields: append([]FieldError(nil), fields...)}
}

type OperationError struct {
	Operation string
	Entity    string
	ID        string
	Err       error
}

func (e *OperationError) Error() string {
	if e.ID == "" {
		return fmt.Sprintf("%s %s: %v", e.Operation, e.Entity, e.Err)
	}
	return fmt.Sprintf("%s %s %s: %v", e.Operation, e.Entity, e.ID, e.Err)
}

func (e *OperationError) Unwrap() error {
	return e.Err
}

func Wrap(operation, entity, id string, err error) error {
	if err == nil {
		return nil
	}
	return &OperationError{Operation: operation, Entity: entity, ID: id, Err: err}
}

func IsRetryable(err error) bool {
	return errors.Is(err, ErrConflict) || errors.Is(err, ErrCanceled)
}
