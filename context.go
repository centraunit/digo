package digo

import (
	"context"
	"sync"
)

type ctxKeyRequestID struct{}
type ctxKeyResolveStack struct{}
type ctxKeyContainer struct{}

// ContainerContext is a context.Context with an extra value bag used at resolve time.
type ContainerContext struct {
	context.Context
	values sync.Map
}

// NewContainerContext wraps parent (or Background).
func NewContainerContext(parent context.Context) *ContainerContext {
	if parent == nil {
		parent = context.Background()
	}
	return &ContainerContext{Context: parent}
}

// AsContainerContext returns ctx as *ContainerContext, wrapping if needed.
func AsContainerContext(ctx context.Context) *ContainerContext {
	if ctx == nil {
		return NewContainerContext(context.Background())
	}
	if cc, ok := ctx.(*ContainerContext); ok {
		return cc
	}
	return NewContainerContext(ctx)
}

// WithValue returns a child with an extra value (receiver is not mutated).
func (c *ContainerContext) WithValue(key, val interface{}) *ContainerContext {
	newCtx := &ContainerContext{Context: c.Context}
	c.values.Range(func(k, v interface{}) bool {
		newCtx.values.Store(k, v)
		return true
	})
	newCtx.values.Store(key, val)
	return newCtx
}

// Value looks up bag values first, then the parent context.
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

// MergeWith copies values from c then other; other's keys win on conflict.
func (c *ContainerContext) MergeWith(other *ContainerContext) *ContainerContext {
	parent := context.Background()
	if c != nil && c.Context != nil {
		parent = c.Context
	}
	newCtx := NewContainerContext(parent)
	if c != nil {
		c.values.Range(func(k, v interface{}) bool {
			newCtx.values.Store(k, v)
			return true
		})
	}
	if other != nil {
		other.values.Range(func(k, v interface{}) bool {
			newCtx.values.Store(k, v)
			return true
		})
		if other.Context != nil {
			newCtx.Context = other.Context
		}
	}
	return newCtx
}

// WithRequestID stores a non-empty request id on the context.
func WithRequestID(ctx context.Context, id string) context.Context {
	return AsContainerContext(ctx).WithValue(ctxKeyRequestID{}, id)
}

// RequestID returns the request id from ctx, if any.
func RequestID(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	if v := ctx.Value(ctxKeyRequestID{}); v != nil {
		if s, ok := v.(string); ok && s != "" {
			return s, true
		}
	}
	return "", false
}

func withResolveStack(ctx context.Context, stack map[string]struct{}) context.Context {
	return context.WithValue(AsContainerContext(ctx), ctxKeyResolveStack{}, stack)
}

func resolveStackFrom(ctx context.Context) map[string]struct{} {
	if ctx == nil {
		return nil
	}
	if v := ctx.Value(ctxKeyResolveStack{}); v != nil {
		if s, ok := v.(map[string]struct{}); ok {
			return s
		}
	}
	return nil
}

func withContainer(ctx context.Context, c *Container) context.Context {
	return context.WithValue(AsContainerContext(ctx), ctxKeyContainer{}, c)
}

func containerFrom(ctx context.Context) *Container {
	if ctx == nil {
		return nil
	}
	if v := ctx.Value(ctxKeyContainer{}); v != nil {
		if c, ok := v.(*Container); ok {
			return c
		}
	}
	return nil
}
