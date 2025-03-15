package digo

import (
	"context"
	"sync"
)

// ContainerContext extends the standard context.Context with container-specific functionality.
// It provides value inheritance and merging capabilities for service configuration.
type ContainerContext struct {
	context.Context
	values sync.Map
}

// NewContainerContext creates a new ContainerContext wrapping a standard context.Context.
// The new context inherits all values from the parent context.
func NewContainerContext(parent context.Context) *ContainerContext {
	if parent == nil {
		parent = context.Background()
	}
	return &ContainerContext{
		Context: parent,
	}
}

// WithValue returns a new ContainerContext with the provided key-value pair.
// The new context inherits all values from the parent context.
func (c *ContainerContext) WithValue(key, val interface{}) *ContainerContext {
	newCtx := &ContainerContext{
		Context: c.Context,
	}
	c.values.Range(func(k, v interface{}) bool {
		newCtx.values.Store(k, v)
		return true
	})
	newCtx.values.Store(key, val)
	return newCtx
}

func (c *ContainerContext) Parent() context.Context {
	return c.Context
}

func (c *ContainerContext) Value(key interface{}) interface{} {
	if c == nil {
		return nil
	}
	if val, ok := c.values.Load(key); ok {
		return val
	}
	if c.Context != nil {
		return c.Context.Value(key)
	}
	return nil
}

// Values returns the underlying sync.Map of values stored in the context.
func (c *ContainerContext) Values() *sync.Map {
	return &c.values
}

// MergeWith combines values from another ContainerContext.
// Values from the other context override existing values with the same key.
func (c *ContainerContext) MergeWith(other *ContainerContext) *ContainerContext {
	newCtx := NewContainerContext(c.Context)

	// First copy values from current context (base values)
	c.values.Range(func(k, v interface{}) bool {
		newCtx.values.Store(k, v)
		return true
	})

	// Then copy values from the other context (overriding values)
	if other != nil {
		other.values.Range(func(k, v interface{}) bool {
			newCtx.values.Store(k, v)
			return true
		})
	}

	return newCtx
}
