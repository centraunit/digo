package digo

import "fmt"

// CircularDependencyError is returned when a resolve chain re-enters the same binding.
type CircularDependencyError struct {
	Type string
}

func (e *CircularDependencyError) Error() string {
	return fmt.Sprintf("digo: circular dependency: %s", e.Type)
}

// BindingNotFoundError is returned when no provider is registered.
type BindingNotFoundError struct {
	Type string
}

func (e *BindingNotFoundError) Error() string {
	return fmt.Sprintf("digo: no binding for %s", e.Type)
}

// NilServiceError is returned when binding a nil instance.
type NilServiceError struct {
	Type string
}

func (e *NilServiceError) Error() string {
	return fmt.Sprintf("digo: nil service for %s", e.Type)
}

// InitializationError wraps OnBoot failures.
type InitializationError struct {
	Type string
	Err  error
}

func (e *InitializationError) Error() string {
	return fmt.Sprintf("digo: OnBoot %s: %v", e.Type, e.Err)
}

func (e *InitializationError) Unwrap() error { return e.Err }

// MissingContextValueError is returned when a required context value is absent.
type MissingContextValueError struct {
	Key string
}

func (e *MissingContextValueError) Error() string {
	return fmt.Sprintf("digo: missing %s in context", e.Key)
}

// TypeMismatchError is returned when a factory yields the wrong type.
type TypeMismatchError struct {
	Expected string
	Got      string
}

func (e *TypeMismatchError) Error() string {
	return fmt.Sprintf("digo: type mismatch: want %s, got %s", e.Expected, e.Got)
}

// ShutdownError wraps OnShutdown failures.
type ShutdownError struct {
	Type string
	Err  error
}

func (e *ShutdownError) Error() string {
	return fmt.Sprintf("digo: OnShutdown %s: %v", e.Type, e.Err)
}

func (e *ShutdownError) Unwrap() error { return e.Err }
