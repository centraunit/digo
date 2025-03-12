package mock

import (
	"fmt"

	services "github.com/centraunit/goallin_services"
)

// Core interfaces
type Database interface {
	services.Lifecycle
	Connect() error
	GetContextValue(key string) (interface{}, error)
}

type Cache interface {
	services.Lifecycle
	Get(key string) interface{}
}

// Mock implementations
type MockDB struct {
	isConnected bool
	ctx         *services.ContainerContext
	RequestID   string
}

func (m *MockDB) Connect() error {
	return nil
}

func (m *MockDB) OnBoot(ctx *services.ContainerContext) error {
	m.isConnected = true
	m.ctx = ctx

	// Handle nil request_id gracefully
	if reqID := ctx.Value("request_id"); reqID != nil {
		if str, ok := reqID.(string); ok {
			m.RequestID = str
		}
	}

	return nil
}
func (md *MockDB) GetContextValue(key string) (interface{}, error) {
	if md.ctx == nil {
		return nil, fmt.Errorf("context is nil")
	}
	return md.ctx.Value(key), nil
}

func (m *MockDB) OnShutdown(ctx *services.ContainerContext) error {
	m.isConnected = false
	m.ctx = nil
	return nil
}

func (m *MockDB) IsConnected() bool {
	return m.isConnected
}

type MockCache struct {
	db Database
}

func (m *MockCache) Get(key string) interface{} {
	return nil
}

func (m *MockCache) OnBoot(ctx *services.ContainerContext) error {
	db, err := services.ResolveTransient[Database]()
	if err != nil {
		return err
	}
	m.db = db
	return nil
}

func (m *MockCache) OnShutdown(ctx *services.ContainerContext) error {
	return nil
}

// Circular dependency test types
type CircularService1 interface {
	services.Lifecycle
	GetService2() CircularService2
}

type CircularService2 interface {
	services.Lifecycle
	GetService1() CircularService1
}

type CircularImpl1 struct {
	svc2 CircularService2
}

func (i *CircularImpl1) OnBoot(ctx *services.ContainerContext) error {
	var err error
	i.svc2, err = services.ResolveTransient[CircularService2]()
	return err
}

func (i *CircularImpl1) OnShutdown(ctx *services.ContainerContext) error { return nil }
func (i *CircularImpl1) GetService2() CircularService2                   { return i.svc2 }

type CircularImpl2 struct {
	svc1 CircularService1
}

func (i *CircularImpl2) OnBoot(ctx *services.ContainerContext) error {
	var err error
	i.svc1, err = services.ResolveTransient[CircularService1]()
	return err
}

func (i *CircularImpl2) OnShutdown(ctx *services.ContainerContext) error { return nil }
func (i *CircularImpl2) GetService1() CircularService1                   { return i.svc1 }

// Add FailingDB for testing initialization failures
type FailingDB struct {
	MockDB
	ShouldFail bool
}

func (f *FailingDB) OnBoot(ctx *services.ContainerContext) error {
	if f.ShouldFail {
		return fmt.Errorf("simulated boot failure")
	}
	return f.MockDB.OnBoot(ctx)
}

// Add these interfaces and implementations
type DeepService3 interface {
	services.Lifecycle
	GetValue() string
}

type DeepService2 interface {
	services.Lifecycle
	GetService3() DeepService3
}

type DeepService1 interface {
	services.Lifecycle
	GetService2() DeepService2
}

type DeepImpl3 struct {
	Value string
}

func (d *DeepImpl3) OnBoot(ctx *services.ContainerContext) error {
	d.Value = "deep"
	return nil
}

func (d *DeepImpl3) OnShutdown(ctx *services.ContainerContext) error {
	return nil
}

func (d *DeepImpl3) GetValue() string {
	return d.Value
}

type DeepImpl2 struct {
	svc3 DeepService3
}

func (d *DeepImpl2) OnBoot(ctx *services.ContainerContext) error {
	var err error
	d.svc3, err = services.ResolveTransient[DeepService3]()
	return err
}

func (d *DeepImpl2) OnShutdown(ctx *services.ContainerContext) error {
	return nil
}

func (d *DeepImpl2) SetService3(svc DeepService3) {
	d.svc3 = svc
}

func (d *DeepImpl2) GetService3() DeepService3 {
	return d.svc3
}

type DeepImpl1 struct {
	svc2 DeepService2
}

func (d *DeepImpl1) OnBoot(ctx *services.ContainerContext) error {
	var err error
	d.svc2, err = services.ResolveTransient[DeepService2]()
	return err
}

func (d *DeepImpl1) OnShutdown(ctx *services.ContainerContext) error {
	return nil
}

func (d *DeepImpl1) SetService2(svc DeepService2) {
	d.svc2 = svc
}

func (d *DeepImpl1) GetService2() DeepService2 {
	return d.svc2
}

// Add Service and SingletonTestService
type Service interface {
	services.Lifecycle
	IsInitialized() bool
}

type SingletonTestService struct {
	initialized bool
}

func (s *SingletonTestService) OnBoot(ctx *services.ContainerContext) error {
	s.initialized = true
	return nil
}

func (s *SingletonTestService) OnShutdown(ctx *services.ContainerContext) error {
	return nil
}

func (s *SingletonTestService) IsInitialized() bool {
	return s.initialized
}

// Add ComplexServiceInterface and ComplexService
type ComplexServiceInterface interface {
	services.Lifecycle
	GetDB() Database
	GetCache() Cache
}

type ComplexService struct {
	DB    Database
	Cache Cache
}

func (c *ComplexService) OnBoot(ctx *services.ContainerContext) error {
	var err error
	c.DB, err = services.ResolveTransient[Database]()
	if err != nil {
		return err
	}
	c.Cache, err = services.ResolveTransient[Cache]()
	return err
}

func (c *ComplexService) OnShutdown(ctx *services.ContainerContext) error {
	return nil
}

func (c *ComplexService) GetDB() Database {
	return c.DB
}

func (c *ComplexService) GetCache() Cache {
	return c.Cache
}
