package services

// Package services provides interfaces for dependency injection and lifecycle management.

// Lifecycle defines the interface for services that require initialization and cleanup.
type Lifecycle interface {
	// OnBoot is called when the service is being initialized.
	// It receives a ContainerContext for configuration access.
	OnBoot(ctx *ContainerContext) error

	// OnShutdown is called when the service is being terminated.
	// It should clean up any resources held by the service.
	OnShutdown(ctx *ContainerContext) error
}

// ConditionalBinding allows for context-based service resolution.
type ConditionalBinding interface {
	// When evaluates a predicate to determine the appropriate service implementation.
	When(predicate ContextPredicate) Lifecycle
}

// ContextPredicate evaluates context conditions to determine service binding.
// Returns a service instance and any error that occurred during evaluation.
type ContextPredicate func(ctx *ContainerContext) (Lifecycle, error)

// Scope defines the lifetime and sharing behavior of a service.
type Scope string

// Available service scopes
const (
	// ScopeTransient creates a new instance for each resolution
	ScopeTransient Scope = "transient"
	// ScopeRequest shares an instance within a request context
	ScopeRequest Scope = "request"
	// ScopeSingleton shares a single instance across the application
	ScopeSingleton Scope = "singleton"
)
