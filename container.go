package digo

import (
	"context"
	"fmt"
	"reflect"
	"sync"
)

type provider struct {
	scope   Scope
	typ     reflect.Type
	factory func(*ContainerContext) (Lifecycle, error)
}

type singletonState struct {
	provider *provider
	once     sync.Once
	inst     Lifecycle
	bootErr  error
	ctx      *ContainerContext
}

type managed struct {
	inst Lifecycle
	typ  reflect.Type
	ctx  *ContainerContext
}

// Container is a thread-safe DI container with singleton, request, and transient scopes.
type Container struct {
	mu sync.Mutex

	baseCtx *ContainerContext

	singletons map[string]*singletonState
	providers  map[string]*provider // key: scope:type for request/transient; singleton also registered here for lookup

	// requestID -> typeKey -> managed
	requests map[string]map[string]*managed

	// transient instances created outside request scope (cleaned by Shutdown)
	transients []*managed

	booted bool
}

var (
	defaultOnce sync.Once
	defaultCtr  *Container
)

// NewContainer creates an empty container.
func NewContainer() *Container {
	return &Container{
		baseCtx:    NewContainerContext(context.Background()),
		singletons: make(map[string]*singletonState),
		providers:  make(map[string]*provider),
		requests:   make(map[string]map[string]*managed),
	}
}

// getContainer returns the process default container.
func getContainer() *Container {
	defaultOnce.Do(func() {
		defaultCtr = NewContainer()
	})
	return defaultCtr
}

func typeKey(t reflect.Type) string {
	return t.String()
}

func providerKey(scope Scope, t reflect.Type) string {
	return string(scope) + ":" + typeKey(t)
}

func isNilService(service any) bool {
	if service == nil {
		return true
	}
	v := reflect.ValueOf(service)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

func (c *Container) registerProvider(scope Scope, t reflect.Type, factory func(*ContainerContext) (Lifecycle, error)) {
	key := providerKey(scope, t)
	c.mu.Lock()
	defer c.mu.Unlock()
	p := &provider{scope: scope, typ: t, factory: factory}
	c.providers[key] = p
	if scope == ScopeSingleton {
		c.singletons[typeKey(t)] = &singletonState{provider: p}
	}
}

// BindSingleton registers a concrete singleton instance.
func BindSingleton[T Lifecycle](service T, ctx ...*ContainerContext) error {
	return getContainer().BindSingleton(service, ctx...)
}

// BindSingleton registers a concrete singleton instance on c.
func (c *Container) BindSingleton[T Lifecycle](service T, ctx ...*ContainerContext) error {
	t := reflect.TypeOf((*T)(nil)).Elem()
	if isNilService(service) {
		return &NilServiceError{Type: t.String()}
	}
	var bindCtx *ContainerContext
	if len(ctx) > 0 && ctx[0] != nil {
		bindCtx = ctx[0]
	}
	c.registerProvider(ScopeSingleton, t, func(cc *ContainerContext) (Lifecycle, error) {
		return service, nil
	})
	if bindCtx != nil {
		c.mu.Lock()
		if st := c.singletons[typeKey(t)]; st != nil {
			st.ctx = bindCtx
		}
		c.mu.Unlock()
	}
	return nil
}

// ProvideSingleton registers a singleton factory.
func ProvideSingleton[T Lifecycle](factory Factory[T]) error {
	return getContainer().ProvideSingleton(factory)
}

// ProvideSingleton registers a singleton factory on c.
func (c *Container) ProvideSingleton[T Lifecycle](factory Factory[T]) error {
	t := reflect.TypeOf((*T)(nil)).Elem()
	if factory == nil {
		return &NilServiceError{Type: t.String()}
	}
	c.registerProvider(ScopeSingleton, t, func(cc *ContainerContext) (Lifecycle, error) {
		return factory(cc)
	})
	return nil
}

// ProvideRequest registers a request-scoped factory.
func ProvideRequest[T Lifecycle](factory Factory[T]) error {
	return getContainer().ProvideRequest(factory)
}

// ProvideRequest registers a request-scoped factory on c.
func (c *Container) ProvideRequest[T Lifecycle](factory Factory[T]) error {
	t := reflect.TypeOf((*T)(nil)).Elem()
	if factory == nil {
		return &NilServiceError{Type: t.String()}
	}
	c.registerProvider(ScopeRequest, t, func(cc *ContainerContext) (Lifecycle, error) {
		return factory(cc)
	})
	return nil
}

// ProvideTransient registers a transient factory (new instance every resolve).
func ProvideTransient[T Lifecycle](factory Factory[T]) error {
	return getContainer().ProvideTransient(factory)
}

// ProvideTransient registers a transient factory on c.
func (c *Container) ProvideTransient[T Lifecycle](factory Factory[T]) error {
	t := reflect.TypeOf((*T)(nil)).Elem()
	if factory == nil {
		return &NilServiceError{Type: t.String()}
	}
	c.registerProvider(ScopeTransient, t, func(cc *ContainerContext) (Lifecycle, error) {
		return factory(cc)
	})
	return nil
}

func (c *Container) resolveCtx(ctx context.Context) *ContainerContext {
	cc := AsContainerContext(ctx)
	c.mu.Lock()
	base := c.baseCtx
	c.mu.Unlock()
	return base.MergeWith(cc)
}

func cloneStack(in map[string]struct{}) map[string]struct{} {
	out := make(map[string]struct{}, len(in)+1)
	for k := range in {
		out[k] = struct{}{}
	}
	return out
}

func (c *Container) beginResolve(ctx context.Context, key string) (context.Context, error) {
	stack := resolveStackFrom(ctx)
	if stack != nil {
		if _, ok := stack[key]; ok {
			return ctx, &CircularDependencyError{Type: key}
		}
		stack = cloneStack(stack)
	} else {
		stack = make(map[string]struct{})
	}
	stack[key] = struct{}{}
	ctx = withResolveStack(ctx, stack)
	ctx = withContainer(ctx, c)
	return ctx, nil
}

func bootInstance(inst Lifecycle, cc *ContainerContext) error {
	if err := inst.OnBoot(cc); err != nil {
		return &InitializationError{Type: reflect.TypeOf(inst).String(), Err: err}
	}
	return nil
}

func shutdownInstance(inst Lifecycle, cc *ContainerContext) error {
	if err := inst.OnShutdown(cc); err != nil {
		return &ShutdownError{Type: reflect.TypeOf(inst).String(), Err: err}
	}
	return nil
}

// ResolveSingleton resolves a singleton from the default container.
func ResolveSingleton[T Lifecycle]() (T, error) {
	return getContainer().ResolveSingleton[T]()
}

// ResolveSingleton resolves a singleton from c.
func (c *Container) ResolveSingleton[T Lifecycle]() (T, error) {
	var zero T
	t := reflect.TypeOf((*T)(nil)).Elem()
	tk := typeKey(t)
	pk := providerKey(ScopeSingleton, t)

	c.mu.Lock()
	st, ok := c.singletons[tk]
	c.mu.Unlock()
	if !ok || st == nil || st.provider == nil {
		return zero, &BindingNotFoundError{Type: t.String()}
	}

	ctx, err := c.beginResolve(context.Background(), pk)
	if err != nil {
		return zero, err
	}
	cc := c.resolveCtx(ctx)
	if st.ctx != nil {
		cc = st.ctx.MergeWith(cc)
	}

	st.once.Do(func() {
		inst, ferr := st.provider.factory(cc)
		if ferr != nil {
			st.bootErr = ferr
			return
		}
		if err := bootInstance(inst, cc); err != nil {
			st.bootErr = err
			return
		}
		st.inst = inst
		st.ctx = cc
	})

	if st.bootErr != nil {
		return zero, st.bootErr
	}
	typed, ok := st.inst.(T)
	if !ok {
		return zero, &TypeMismatchError{Expected: t.String(), Got: reflect.TypeOf(st.inst).String()}
	}
	return typed, nil
}

// ResolveTransient resolves a new transient from the default container.
func ResolveTransient[T Lifecycle](ctx context.Context) (T, error) {
	return getContainer().ResolveTransient[T](ctx)
}

// ResolveTransient creates a new transient instance.
func (c *Container) ResolveTransient[T Lifecycle](ctx context.Context) (T, error) {
	var zero T
	t := reflect.TypeOf((*T)(nil)).Elem()
	pk := providerKey(ScopeTransient, t)

	c.mu.Lock()
	p, ok := c.providers[pk]
	c.mu.Unlock()
	if !ok || p == nil {
		return zero, &BindingNotFoundError{Type: t.String()}
	}

	rctx, err := c.beginResolve(ctx, pk)
	if err != nil {
		return zero, err
	}
	cc := c.resolveCtx(rctx)

	inst, err := p.factory(cc)
	if err != nil {
		return zero, err
	}
	if err := bootInstance(inst, cc); err != nil {
		return zero, err
	}

	typed, ok := inst.(T)
	if !ok {
		_ = shutdownInstance(inst, cc)
		return zero, &TypeMismatchError{Expected: t.String(), Got: reflect.TypeOf(inst).String()}
	}

	m := &managed{inst: inst, typ: t, ctx: cc}
	if reqID, ok := RequestID(rctx); ok {
		c.mu.Lock()
		if c.requests[reqID] == nil {
			c.requests[reqID] = make(map[string]*managed)
		}
		// allow multiple transients of same type per request via unique key
		uniq := fmt.Sprintf("%s#%p", typeKey(t), inst)
		c.requests[reqID][uniq] = m
		c.mu.Unlock()
	} else {
		c.mu.Lock()
		c.transients = append(c.transients, m)
		c.mu.Unlock()
	}
	return typed, nil
}

// ResolveRequest resolves a request-scoped service from the default container.
func ResolveRequest[T Lifecycle](ctx context.Context) (T, error) {
	return getContainer().ResolveRequest[T](ctx)
}

// ResolveRequest resolves (or creates) the request-scoped instance for request_id in ctx.
func (c *Container) ResolveRequest[T Lifecycle](ctx context.Context) (T, error) {
	var zero T
	t := reflect.TypeOf((*T)(nil)).Elem()
	tk := typeKey(t)
	pk := providerKey(ScopeRequest, t)

	reqID, ok := RequestID(ctx)
	if !ok {
		return zero, &MissingContextValueError{Key: "request_id"}
	}

	c.mu.Lock()
	if byType, ok := c.requests[reqID]; ok {
		if m, ok := byType[tk]; ok && m != nil {
			c.mu.Unlock()
			typed, ok := m.inst.(T)
			if !ok {
				return zero, &TypeMismatchError{Expected: t.String(), Got: reflect.TypeOf(m.inst).String()}
			}
			return typed, nil
		}
	}
	p, ok := c.providers[pk]
	c.mu.Unlock()
	if !ok || p == nil {
		return zero, &BindingNotFoundError{Type: t.String()}
	}

	rctx, err := c.beginResolve(ctx, pk)
	if err != nil {
		return zero, err
	}
	cc := c.resolveCtx(rctx)

	// Double-check under lock after factory? Create outside lock, then publish.
	inst, err := p.factory(cc)
	if err != nil {
		return zero, err
	}
	if err := bootInstance(inst, cc); err != nil {
		return zero, err
	}

	c.mu.Lock()
	if byType, ok := c.requests[reqID]; ok {
		if m, ok := byType[tk]; ok && m != nil {
			c.mu.Unlock()
			_ = shutdownInstance(inst, cc) // lost the race; discard duplicate
			typed, ok := m.inst.(T)
			if !ok {
				return zero, &TypeMismatchError{Expected: t.String(), Got: reflect.TypeOf(m.inst).String()}
			}
			return typed, nil
		}
	} else {
		c.requests[reqID] = make(map[string]*managed)
	}
	c.requests[reqID][tk] = &managed{inst: inst, typ: t, ctx: cc}
	c.mu.Unlock()

	typed, ok := inst.(T)
	if !ok {
		return zero, &TypeMismatchError{Expected: t.String(), Got: reflect.TypeOf(inst).String()}
	}
	return typed, nil
}

// Boot initializes all singleton providers that have not been resolved yet.
func Boot() error { return getContainer().Boot() }

// Boot initializes all singleton providers on c.
func (c *Container) Boot() error {
	c.mu.Lock()
	if c.booted {
		c.mu.Unlock()
		return nil
	}
	states := make([]*singletonState, 0, len(c.singletons))
	for _, st := range c.singletons {
		states = append(states, st)
	}
	c.mu.Unlock()

	var firstErr error
	booted := make([]*singletonState, 0, len(states))
	var failed *singletonState
	for _, st := range states {
		t := st.provider.typ
		_, err := c.resolveSingletonState(st, providerKey(ScopeSingleton, t))
		if err != nil {
			firstErr = err
			failed = st
			break
		}
		booted = append(booted, st)
	}
	if firstErr != nil {
		for _, st := range booted {
			if st.inst != nil {
				cc := st.ctx
				if cc == nil {
					cc = c.baseCtx
				}
				_ = shutdownInstance(st.inst, cc)
				st.inst = nil
				st.bootErr = nil
				st.once = sync.Once{}
			}
		}
		if failed != nil {
			failed.inst = nil
			failed.bootErr = nil
			failed.once = sync.Once{}
		}
		return firstErr
	}
	c.mu.Lock()
	c.booted = true
	c.mu.Unlock()
	return nil
}

func (c *Container) resolveSingletonState(st *singletonState, pk string) (Lifecycle, error) {
	ctx, err := c.beginResolve(context.Background(), pk)
	if err != nil {
		return nil, err
	}
	cc := c.resolveCtx(ctx)
	if st.ctx != nil {
		cc = st.ctx.MergeWith(cc)
	}
	st.once.Do(func() {
		inst, ferr := st.provider.factory(cc)
		if ferr != nil {
			st.bootErr = ferr
			return
		}
		if err := bootInstance(inst, cc); err != nil {
			st.bootErr = err
			return
		}
		st.inst = inst
		st.ctx = cc
	})
	if st.bootErr != nil {
		return nil, st.bootErr
	}
	return st.inst, nil
}

// Shutdown shuts down request/transient instances; singletons only if clearSingletons.
func Shutdown(clearSingletons bool) error { return getContainer().Shutdown(clearSingletons) }

// Shutdown shuts down managed instances on c.
func (c *Container) Shutdown(clearSingletons bool) error {
	c.mu.Lock()
	var toStop []*managed
	for _, byType := range c.requests {
		for _, m := range byType {
			toStop = append(toStop, m)
		}
	}
	toStop = append(toStop, c.transients...)
	c.requests = make(map[string]map[string]*managed)
	c.transients = nil

	var singletons []*singletonState
	if clearSingletons {
		for _, st := range c.singletons {
			singletons = append(singletons, st)
		}
		c.booted = false
	}
	c.mu.Unlock()

	var errs []error
	for _, m := range toStop {
		if m == nil || m.inst == nil {
			continue
		}
		cc := m.ctx
		if cc == nil {
			cc = c.baseCtx
		}
		if err := shutdownInstance(m.inst, cc); err != nil {
			errs = append(errs, err)
		}
	}
	if clearSingletons {
		for _, st := range singletons {
			if st.inst == nil {
				continue
			}
			cc := st.ctx
			if cc == nil {
				cc = c.baseCtx
			}
			inst := st.inst
			if err := shutdownInstance(inst, cc); err != nil {
				errs = append(errs, err)
			}
		}
		c.mu.Lock()
		c.singletons = make(map[string]*singletonState)
		c.providers = make(map[string]*provider)
		c.mu.Unlock()
	}

	if len(errs) == 0 {
		return nil
	}
	return errs[0]
}

// Reset shuts everything down and clears providers (test helper).
func Reset() { _ = getContainer().Reset() }

// Reset clears all providers and instances on c.
func (c *Container) Reset() error {
	err := c.Shutdown(true)
	c.mu.Lock()
	c.providers = make(map[string]*provider)
	c.singletons = make(map[string]*singletonState)
	c.requests = make(map[string]map[string]*managed)
	c.transients = nil
	c.booted = false
	c.baseCtx = NewContainerContext(context.Background())
	c.mu.Unlock()
	return err
}

// EndRequest shuts down all instances associated with requestID.
func (c *Container) EndRequest(requestID string) error {
	if requestID == "" {
		return &MissingContextValueError{Key: "request_id"}
	}
	c.mu.Lock()
	byType := c.requests[requestID]
	delete(c.requests, requestID)
	c.mu.Unlock()

	var first error
	for _, m := range byType {
		if m == nil || m.inst == nil {
			continue
		}
		cc := m.ctx
		if cc == nil {
			cc = c.baseCtx
		}
		if err := shutdownInstance(m.inst, cc); err != nil && first == nil {
			first = err
		}
	}
	return first
}
