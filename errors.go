package digo

import "fmt"

// CircularDependencyError represents a circular dependency detection error.
type CircularDependencyError struct {
	Type string
}

func (e *CircularDependencyError) Error() string {
	return fmt.Sprintf("circular dependency detected for type: %s", e.Type)
}

// BindingNotFoundError represents a missing binding error.
type BindingNotFoundError struct {
	Type string
}

func (e *BindingNotFoundError) Error() string {
	return fmt.Sprintf("no binding found for type: %s", e.Type)
}

// NilServiceError represents an attempt to bind a nil service.
type NilServiceError struct {
	Type string
}

func (e *NilServiceError) Error() string {
	return fmt.Sprintf("nil service provided for type: %s", e.Type)
}

// InitializationError represents a service initialization failure.
type InitializationError struct {
	Type string
	Err  error
}

func (e *InitializationError) Error() string {
	return fmt.Sprintf("initialization failed for type %s: %v", e.Type, e.Err)
}

func (e *InitializationError) Unwrap() error {
	return e.Err
}

// MissingContextValueError represents a missing required context value.
type MissingContextValueError struct {
	Key string
}

func (e *MissingContextValueError) Error() string {
	return fmt.Sprintf("required context value not found: %s", e.Key)
}

// TypeMismatchError represents a type assertion failure.
type TypeMismatchError struct {
	Expected string
	Got      string
}

func (e *TypeMismatchError) Error() string {
	return fmt.Sprintf("type mismatch: expected %s, got %s", e.Expected, e.Got)
}

// ShutdownError represents a service shutdown failure.
type ShutdownError struct {
	Type string
	Err  error
}

func (e *ShutdownError) Error() string {
	return fmt.Sprintf("shutdown failed for type %s: %v", e.Type, e.Err)
}

func (e *ShutdownError) Unwrap() error {
	return e.Err
}

// PredicateError represents a predicate evaluation failure.
type PredicateError struct {
	Type string
	Err  error
}

func (e *PredicateError) Error() string {
	return fmt.Sprintf("predicate evaluation failed for type %s: %v", e.Type, e.Err)
}

func (e *PredicateError) Unwrap() error {
	return e.Err
}

// BootError represents a service boot failure.
type BootError struct {
	Type string
	Err  error
}

func (e *BootError) Error() string {
	return fmt.Sprintf("boot failed for type %s: %v", e.Type, e.Err)
}

func (e *BootError) Unwrap() error {
	return e.Err
}

// InvalidScopeError represents an invalid scope usage.
type InvalidScopeError struct {
	Type  string
	Scope string
}

func (e *InvalidScopeError) Error() string {
	return fmt.Sprintf("invalid scope %s for type %s", e.Scope, e.Type)
}
