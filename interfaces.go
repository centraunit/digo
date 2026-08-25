package digo

// Lifecycle is implemented by services that need boot and shutdown hooks.
type Lifecycle interface {
	OnBoot(ctx *ContainerContext) error
	OnShutdown(ctx *ContainerContext) error
}

// NoopLifecycle embeds into types that need Lifecycle but have nothing to do.
type NoopLifecycle struct{}

func (NoopLifecycle) OnBoot(*ContainerContext) error     { return nil }
func (NoopLifecycle) OnShutdown(*ContainerContext) error { return nil }

// Scope is the lifetime of a binding.
type Scope string

const (
	ScopeTransient Scope = "transient"
	ScopeRequest   Scope = "request"
	ScopeSingleton Scope = "singleton"
)

// Factory creates a service instance for the given context.
type Factory[T Lifecycle] func(ctx *ContainerContext) (T, error)
