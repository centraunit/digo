package digo

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"sync"
)

// Package digo provides a high-performance dependency injection container.

// bindingDefinition represents a service binding in the container.
// It holds the concrete implementation, scope, and associated context.
type bindingDefinition struct {
	scope       Scope
	concrete    Lifecycle
	abstract    reflect.Type
	initialized bool
	ctx         *ContainerContext
	predicate   ContextPredicate
}

type resolutionState struct {
	chain    map[string]bool
	mu       sync.Mutex
	keyCache []string
}

// container manages service bindings and their lifecycle.
// It provides thread-safe access to digo and handles dependency resolution.
type container struct {
	bindings        map[string]bindingDefinition
	ctx             *ContainerContext
	mu              sync.RWMutex
	booted          bool
	bootOnce        sync.Once
	resolutionState sync.Map
	resolutionMu    sync.RWMutex
	statePool       sync.Pool
	goidCache       sync.Map
}

var (
	once             sync.Once
	defaultContainer *container
	typeStringCache  sync.Map
)

func makeBindingKey(scope Scope, serviceType reflect.Type) string {
	if cached, ok := typeStringCache.Load(serviceType); ok {
		return string(scope) + ":" + cached.(string)
	}
	typeStr := serviceType.String()
	typeStringCache.Store(serviceType, typeStr)
	return string(scope) + ":" + typeStr
}

// GetContainer returns the singleton container instance.
// The container is initialized on first access with default configuration.
func GetContainer() *container {
	once.Do(func() {
		defaultContainer = &container{
			bindings:        make(map[string]bindingDefinition, 32),
			ctx:             NewContainerContext(context.Background()),
			resolutionState: sync.Map{},
			statePool: sync.Pool{
				New: func() interface{} {
					return &resolutionState{
						chain:    make(map[string]bool, 8),
						mu:       sync.Mutex{},
						keyCache: make([]string, 0, 8),
					}
				},
			},
			goidCache: sync.Map{},
		}
	})
	return defaultContainer
}

// Boot initializes all singleton digo in the container.
// It ensures each singleton is initialized exactly once and handles initialization errors.
// Returns an error if any service fails to initialize.
func Boot() error {
	instance := GetContainer()
	var bootErr error

	instance.bootOnce.Do(func() {
		instance.mu.Lock()
		if instance.booted {
			instance.mu.Unlock()
			return
		}

		for _, binding := range instance.bindings {
			if !binding.initialized && binding.scope == ScopeSingleton {
				if err := binding.concrete.OnBoot(binding.ctx); err != nil {
					bootErr = err
					break
				}
				binding.initialized = true
			}
			if binding.scope == ScopeRequest {
				err := binding.concrete.OnBoot(binding.ctx)
				if err != nil {
					bootErr = err
					break
				}
				binding.initialized = true
			}
		}
		instance.mu.Unlock()
	})

	return bootErr
}

// Shutdown gracefully shuts down digo in the container.
// If clearSingletons is true, it also removes singleton digo from the container.
// Returns an error if any service fails to shut down properly.
func Shutdown(clearSingletons bool) error {
	instance := GetContainer()
	instance.mu.Lock()
	defer instance.mu.Unlock()

	// First collect all digo to shutdown
	toShutdown := make([]bindingDefinition, 0)

	for _, binding := range instance.bindings {
		if binding.scope != ScopeSingleton || clearSingletons {
			toShutdown = append(toShutdown, binding)
		}
	}

	// Shutdown digo
	for _, binding := range toShutdown {
		if err := binding.concrete.OnShutdown(binding.ctx); err != nil {
			return &ShutdownError{
				Type: reflect.TypeOf(binding.concrete).String(),
				Err:  err,
			}
		}
	}

	// Clear bindings under lock
	if clearSingletons {
		instance.resolutionMu.Lock()
		instance.bindings = make(map[string]bindingDefinition)
		instance.booted = false
		instance.bootOnce = sync.Once{}
		instance.resolutionState = sync.Map{}
		instance.resolutionMu.Unlock()
	} else {
		// Only remove non-singleton bindings
		for key, binding := range instance.bindings {
			if binding.scope != ScopeSingleton {
				delete(instance.bindings, key)
			}
		}
	}

	return nil
}

// BindTransient registers a service with transient scope.
// Each resolution creates a new instance of the service.
// Returns NilServiceError if the service is nil.
func BindTransient[T Lifecycle](service T, ctx *ContainerContext, predicate ...ContextPredicate) error {
	serviceType := reflect.TypeOf((*T)(nil)).Elem()
	return GetContainer().bind(service, serviceType, ScopeTransient, ctx, predicate...)
}

// BindRequest registers a service with request scope.
// Service instance is shared within a single request context.
// Returns NilServiceError if the service is nil.
func BindRequest[T Lifecycle](service T, ctx *ContainerContext, predicate ...ContextPredicate) error {
	serviceType := reflect.TypeOf((*T)(nil)).Elem()
	return GetContainer().bind(service, serviceType, ScopeRequest, ctx, predicate...)
}

// BindSingleton registers a service with singleton scope.
// Service instance is shared across the entire application.
// Returns NilServiceError if the service is nil.
func BindSingleton[T Lifecycle](service T, ctx ...*ContainerContext) error {
	serviceType := reflect.TypeOf((*T)(nil)).Elem()
	var bindingCtx *ContainerContext
	if len(ctx) > 0 && ctx[0] != nil {
		bindingCtx = ctx[0]
	}
	return GetContainer().bind(service, serviceType, ScopeSingleton, bindingCtx)
}

// ResolveTransient resolves a service with transient scope.
// Returns a new instance on each resolution.
// Returns BindingNotFoundError if service is not registered.
// Returns InitializationError if service fails to initialize.
func ResolveTransient[T Lifecycle]() (T, error) {
	instance := GetContainer()
	var zero T
	serviceType := reflect.TypeOf((*T)(nil)).Elem()
	key := makeBindingKey(ScopeTransient, serviceType)

	if err := instance.startResolving(key); err != nil {
		return zero, err
	}
	defer instance.finishResolving(key)

	instance.mu.Lock()
	binding, ok := instance.bindings[key]
	if !ok {
		instance.mu.Unlock()
		return zero, &BindingNotFoundError{Type: serviceType.String()}
	}

	// For transient scope, we need to shutdown before reuse
	if binding.initialized {
		if err := binding.concrete.OnShutdown(binding.ctx); err != nil {
			instance.mu.Unlock()
			return zero, &ShutdownError{Type: serviceType.String(), Err: err}
		}
		binding.initialized = false
	}

	// Handle predicate
	if binding.predicate != nil {
		instance.mu.Unlock()
		result, err := binding.predicate(binding.ctx)
		if err != nil {
			return zero, &PredicateError{Type: serviceType.String(), Err: err}
		}
		if typed, ok := result.(T); ok {
			if err := typed.OnBoot(binding.ctx); err != nil {
				return zero, &InitializationError{Type: serviceType.String(), Err: err}
			}
			return typed, nil
		}
		return zero, &PredicateError{Type: serviceType.String(), Err: fmt.Errorf("predicate returned invalid type")}
	}

	concrete := binding.concrete
	instance.mu.Unlock()

	if typed, ok := concrete.(T); ok {
		if err := typed.OnBoot(binding.ctx); err != nil {
			return zero, &InitializationError{Type: serviceType.String(), Err: err}
		}

		instance.mu.Lock()
		binding.initialized = true
		instance.bindings[key] = binding
		instance.mu.Unlock()

		return typed, nil
	}

	return zero, &TypeMismatchError{Expected: serviceType.String(), Got: reflect.TypeOf(binding.concrete).String()}
}

// ResolveRequest resolves a service with request scope.
// Returns the same instance within a request context.
// Returns MissingContextValueError if request_id is not in context.
// Returns BindingNotFoundError if service is not registered.
func ResolveRequest[T Lifecycle]() (T, error) {
	instance := GetContainer()
	var zero T
	serviceType := reflect.TypeOf((*T)(nil)).Elem()

	// Create composite key for resolution chain
	key := makeBindingKey(ScopeRequest, serviceType)

	// Check for circular dependency
	if err := instance.startResolving(key); err != nil {
		return zero, err
	}
	defer instance.finishResolving(key)
	instance.mu.RLock()
	binding, ok := instance.bindings[key]
	if !ok {
		instance.mu.RUnlock()
		return zero, &BindingNotFoundError{Type: serviceType.String()}
	}
	requestID := binding.ctx.Value("request_id")
	if requestID == nil {
		instance.mu.RUnlock()

		return zero, &MissingContextValueError{Key: "request_id"}
	}

	// Check if already initialized
	if binding.initialized {
		if typed, ok := binding.concrete.(T); ok {
			instance.mu.RUnlock()
			return typed, nil
		}
		instance.mu.RUnlock()
		return zero, &TypeMismatchError{Expected: serviceType.String(), Got: reflect.TypeOf(binding.concrete).String()}
	}
	instance.mu.RUnlock()

	if binding.predicate != nil {
		result, err := binding.predicate(binding.ctx)
		if err != nil {
			return zero, &PredicateError{Type: serviceType.String(), Err: err}
		}
		binding.concrete = result.(T)
	}
	if err := binding.concrete.OnBoot(binding.ctx); err != nil {
		return zero, &InitializationError{Type: serviceType.String(), Err: err}
	}

	// Update binding under lock
	instance.mu.Lock()
	binding.initialized = true
	instance.bindings[key] = binding
	instance.mu.Unlock()

	return binding.concrete.(T), nil
}

// ResolveSingleton resolves a service with singleton scope.
// Returns the same instance for the entire application.
// Returns BindingNotFoundError if service is not registered.
// Returns InitializationError if service fails to initialize.
func ResolveSingleton[T Lifecycle]() (T, error) {
	var zero T
	instance := GetContainer()
	serviceType := reflect.TypeOf((*T)(nil)).Elem()
	key := makeBindingKey(ScopeSingleton, serviceType)
	instance.mu.RLock()
	binding, ok := instance.bindings[key]
	instance.mu.RUnlock()
	if !ok {
		return zero, &BindingNotFoundError{Type: serviceType.String()}
	}
	// Check for circular dependency
	if err := instance.startResolving(key); err != nil {
		return zero, err
	}
	defer instance.finishResolving(key)

	// Check if already initialized
	if binding.initialized {
		if typed, ok := binding.concrete.(T); ok {
			return typed, nil
		}
		return zero, &TypeMismatchError{Expected: serviceType.String(), Got: reflect.TypeOf(binding.concrete).String()}
	}

	// Initialize outside of lock
	if err := binding.concrete.OnBoot(binding.ctx); err != nil {
		return zero, &InitializationError{Type: serviceType.String(), Err: err}
	}

	// Update binding under lock
	binding.initialized = true
	instance.bindings[key] = binding

	return binding.concrete.(T), nil
}

// Reset clears all container state.
// This function is intended for testing purposes only.
// It removes all bindings and resets the container to its initial state.
func Reset() {
	instance := GetContainer()
	instance.mu.Lock()
	instance.resolutionMu.Lock()

	instance.bindings = make(map[string]bindingDefinition)
	instance.resolutionState = sync.Map{}
	instance.booted = false
	instance.bootOnce = sync.Once{}

	instance.resolutionMu.Unlock()
	instance.mu.Unlock()
}

func (c *container) bind(service Lifecycle, serviceType reflect.Type, scope Scope, ctx *ContainerContext, predicate ...ContextPredicate) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if reflect.ValueOf(service).IsNil() {
		return &NilServiceError{Type: serviceType.String()}
	}

	bindingCtx := ctx
	if bindingCtx == nil {
		bindingCtx = c.ctx
	}
	bindingCtx = bindingCtx.MergeWith(c.ctx)

	var pred ContextPredicate
	if len(predicate) > 0 {
		pred = predicate[0]
	}

	key := makeBindingKey(scope, serviceType)
	c.bindings[key] = bindingDefinition{
		scope:       scope,
		concrete:    service,
		abstract:    serviceType,
		initialized: false,
		ctx:         bindingCtx,
		predicate:   pred,
	}
	return nil
}

// Add methods to track resolution chain
func (c *container) getResolutionState() *resolutionState {
	id := c.getGoroutineID() // Get ID first to minimize lock time

	// Fast path with read lock
	c.resolutionMu.RLock()
	state, ok := c.resolutionState.Load(id)
	c.resolutionMu.RUnlock()
	if ok {
		return state.(*resolutionState)
	}

	// Slow path with write lock
	c.resolutionMu.Lock()
	defer c.resolutionMu.Unlock()

	// Double-check after acquiring write lock
	if state, ok := c.resolutionState.Load(id); ok {
		return state.(*resolutionState)
	}

	state = c.statePool.Get()
	c.resolutionState.Store(id, state)
	return state.(*resolutionState)
}

func (c *container) startResolving(key string) error {
	state := c.getResolutionState()
	state.mu.Lock()
	defer state.mu.Unlock()

	if state.chain[key] {
		return &CircularDependencyError{Type: key}
	}
	state.chain[key] = true
	state.keyCache = append(state.keyCache, key)
	return nil
}

func (c *container) finishResolving(key string) {
	state := c.getResolutionState()
	state.mu.Lock()
	delete(state.chain, key)
	isEmpty := len(state.chain) == 0
	state.mu.Unlock()

	if isEmpty {
		c.resolutionMu.Lock()
		id := c.getGoroutineID()
		if s, ok := c.resolutionState.Load(id); ok {
			c.resolutionState.Delete(id)
			rs := s.(*resolutionState)
			for _, k := range rs.keyCache {
				delete(rs.chain, k)
			}
			rs.keyCache = rs.keyCache[:0]
			c.statePool.Put(rs)
		}
		c.resolutionMu.Unlock()
	}
}

func (c *container) getGoroutineID() string {
	id := goid()
	if cached, ok := c.goidCache.Load(id); ok {
		return cached.(string)
	}
	strID := strconv.FormatInt(id, 10)
	c.goidCache.Store(id, strID)
	return strID
}
