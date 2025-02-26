// Package errors provides custom error types and error handling utilities.
package errors

import (
	"errors"
	"fmt"
)

// Standard errors that can be used for comparison.
var (
	// ErrInvalidInput is returned when user input is invalid.
	ErrInvalidInput = errors.New("Invalid input")

	// ErrUserAborted is returned when a user aborts an operation.
	ErrUserAborted = errors.New("Operation aborted by user")

	// ErrConfigurationInvalid is returned when application configuration is invalid.
	ErrConfigurationInvalid = errors.New("Configuration is invalid")
)

// ValidationError represents an error that occurs during validation.
type ValidationError struct {
	Field   string
	Message string
	Err     error
}

// Error returns the error message.
func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("Validation error for field '%s': %s", e.Field, e.Message)
	}
	return fmt.Sprintf("Validation error: %s", e.Message)
}

// Unwrap returns the underlying error.
func (e *ValidationError) Unwrap() error {
	return e.Err
}

// NewValidationError creates a new ValidationError.
func NewValidationError(field, message string, err error) error {
	return &ValidationError{
		Field:   field,
		Message: message,
		Err:     err,
	}
}

// UserInteractionError represents an error that occurs during user interaction.
type UserInteractionError struct {
	Operation string
	Message   string
	Err       error
}

// Error returns the error message.
func (e *UserInteractionError) Error() string {
	if e.Operation != "" {
		return fmt.Sprintf("User interaction error during %s: %s", e.Operation, e.Message)
	}
	return fmt.Sprintf("User interaction error: %s", e.Message)
}

// Unwrap returns the underlying error.
func (e *UserInteractionError) Unwrap() error {
	return e.Err
}

// NewUserInteractionError creates a new UserInteractionError.
func NewUserInteractionError(operation, message string, err error) error {
	return &UserInteractionError{
		Operation: operation,
		Message:   message,
		Err:       err,
	}
}

// ConfigurationError represents an error related to configuration.
type ConfigurationError struct {
	Component string
	Message   string
	Err       error
}

// Error returns the error message.
func (e *ConfigurationError) Error() string {
	if e.Component != "" {
		return fmt.Sprintf("Configuration error in %s: %s", e.Component, e.Message)
	}
	return fmt.Sprintf("Configuration error: %s", e.Message)
}

// Unwrap returns the underlying error.
func (e *ConfigurationError) Unwrap() error {
	return e.Err
}

// NewConfigurationError creates a new ConfigurationError.
func NewConfigurationError(component, message string, err error) error {
	return &ConfigurationError{
		Component: component,
		Message:   message,
		Err:       err,
	}
}

// IsValidationError returns true if the error is a ValidationError.
func IsValidationError(err error) bool {
	var valErr *ValidationError
	return errors.As(err, &valErr)
}

// IsUserInteractionError returns true if the error is a UserInteractionError.
func IsUserInteractionError(err error) bool {
	var uiErr *UserInteractionError
	return errors.As(err, &uiErr)
}

// IsConfigurationError returns true if the error is a ConfigurationError.
func IsConfigurationError(err error) bool {
	var confErr *ConfigurationError
	return errors.As(err, &confErr)
}

// IsErrUserAborted returns true if the error is or wraps ErrUserAborted.
func IsErrUserAborted(err error) bool {
	return errors.Is(err, ErrUserAborted)
}

// IsErrInvalidInput returns true if the error is or wraps ErrInvalidInput.
func IsErrInvalidInput(err error) bool {
	return errors.Is(err, ErrInvalidInput)
}

// IsErrConfigurationInvalid returns true if the error is or wraps ErrConfigurationInvalid.
func IsErrConfigurationInvalid(err error) bool {
	return errors.Is(err, ErrConfigurationInvalid)
}
