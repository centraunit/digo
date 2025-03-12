# Go Dependency Injection Container

A high-performance, thread-safe dependency injection container with lifecycle management and context awareness.

[![Go Tests](https://github.com/centraunit/goallin_services/actions/workflows/tests.yml/badge.svg)](https://github.com/centraunit/goallin_services/actions/workflows/tests.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/centraunit/goallin_services)](https://goreportcard.com/report/github.com/centraunit/goallin_services)
[![GoDoc](https://godoc.org/github.com/centraunit/goallin_services?status.svg)](https://godoc.org/github.com/centraunit/goallin_services)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Performance Benchmarks

| Operation | Time (ns/op) | Memory (B/op) | Allocs/op |
|-----------|-------------|---------------|-----------|
| Transient Binding | 473.3 | 920 | 5 |
| Request Binding | 853.7 | 1232 | 11 |
| Singleton Binding | 476.8 | 920 | 5 |
| Transient Resolution | 19767 | 584 | 12 |
| Request Resolution | 20122 | 584 | 12 |
| Singleton Resolution | 19745 | 584 | 12 |
| Deep Dependency Chain | 56734 | 1360 | 26 |
| Concurrent Resolution | 98388 | 6695 | 124 |

## Key Features

- Three scoping modes (Singleton, Request, Transient)
- Full lifecycle management (Boot/Shutdown)
- Context awareness and inheritance
- Circular dependency detection
- Thread-safe operations
- Generic type support
- Comprehensive error handling

## Installation

```
go get github.com/centraunit/goallin_services
```

## Quick Start

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/centraunit/goallin_services"
)

// Define a service interface
type Database interface {
	services.Lifecycle
	Connect() error
}

// Implement the service
type PostgresDB struct {
	connected bool
}

func (db *PostgresDB) OnBoot(ctx *services.ContainerContext) error {
	fmt.Println("Database booting up...")
	db.connected = true
	return nil
}

func (db *PostgresDB) OnShutdown(ctx *services.ContainerContext) error {
	fmt.Println("Database shutting down...")
	db.connected = false
	return nil
}

func (db *PostgresDB) Connect() error {
	if !db.connected {
		return fmt.Errorf("database not connected")
	}
	return nil
}

func main() {
	// Create a context
	ctx := services.NewContainerContext(context.Background())
	
	// Register a service
	db := &PostgresDB{}
	err := services.BindSingleton[Database](db)
	if err != nil {
		log.Fatalf("Failed to bind service: %v", err)
	}
	
	// Boot the container
	if err := services.Boot(); err != nil {
		log.Fatalf("Failed to boot container: %v", err)
	}
	
	// Resolve the service
	dbInstance, err := services.ResolveSingleton[Database]()
	if err != nil {
		log.Fatalf("Failed to resolve service: %v", err)
	}
	
	// Use the service
	if err := dbInstance.Connect(); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	
	fmt.Println("Database connected successfully")
	
	// Shutdown when done
	if err := services.Shutdown(true); err != nil {
		log.Fatalf("Failed to shutdown container: %v", err)
	}
}
```

## Service Scopes

### Singleton

Services that should be instantiated only once and shared across the entire application.

```go
// Define a singleton service
type GlobalConfig struct {
	DatabaseURL string
}

func (c *GlobalConfig) OnBoot(ctx *services.ContainerContext) error {
	c.DatabaseURL = ctx.Value("db_url").(string)
	return nil
}

func (c *GlobalConfig) OnShutdown(ctx *services.ContainerContext) error {
	return nil
}

// Bind singleton
config := &GlobalConfig{}
services.BindSingleton[GlobalConfig](config)

// Use anywhere in the application
config, _ := services.ResolveSingleton[GlobalConfig]()
```

### Request

Services that should be instantiated once per request context.

```go
// Define a request-scoped service
type RequestLogger struct {
	RequestID string
}

func (l *RequestLogger) OnBoot(ctx *services.ContainerContext) error {
	l.RequestID = ctx.Value("request_id").(string)
	return nil
}

func (l *RequestLogger) OnShutdown(ctx *services.ContainerContext) error {
	return nil
}

// In HTTP middleware
ctx := services.NewContainerContext(r.Context()).
	WithValue("request_id", "unique-id")
logger := &RequestLogger{}
services.BindRequest[RequestLogger](logger, ctx)

// Use within the same request
logger, _ := services.ResolveRequest[RequestLogger]()
```

### Transient

Services that should be reinitialized on every resolution.

```go
// Define a transient service
type DatabaseConnection struct {
	conn *sql.DB
}

func (db *DatabaseConnection) OnBoot(ctx *services.ContainerContext) error {
	db.conn, _ = sql.Open("postgres", ctx.Value("db_url").(string))
	return nil
}

func (db *DatabaseConnection) OnShutdown(ctx *services.ContainerContext) error {
	return db.conn.Close()
}

// Each resolution gets a new instance
dbConn := &DatabaseConnection{}
services.BindTransient[DatabaseConnection](dbConn, ctx)
conn1, _ := services.ResolveTransient[DatabaseConnection]()
conn2, _ := services.ResolveTransient[DatabaseConnection]() // Different instance
```

## Lifecycle Management

Services implement the `Lifecycle` interface with `OnBoot` and `OnShutdown` methods:

```go
type Service struct{}

func (s *Service) OnBoot(ctx *services.ContainerContext) error {
	// Initialize service
	return nil
}

func (s *Service) OnShutdown(ctx *services.ContainerContext) error {
	// Cleanup resources
	return nil
}

// Boot all services
services.Boot()

// Shutdown with options
services.Shutdown(false) // Keep singletons
services.Shutdown(true)  // Clear everything
```

## Context Awareness

The container provides a context-aware system for passing configuration and request data:

```go
// Create context with values
ctx := services.NewContainerContext(context.Background()).
	WithValue("environment", "production").
	WithValue("region", "us-west")

// Bind with context
service := &MyService{}
services.BindTransient[MyService](service, ctx)

// Context inheritance
childCtx := ctx.WithValue("feature", "enabled")
childCtx.Value("environment") // Returns "production"

// Context merging
ctx1 := services.NewContainerContext(context.Background()).
	WithValue("key1", "value1")
ctx2 := services.NewContainerContext(context.Background()).
	WithValue("key2", "value2")
merged := ctx1.MergeWith(ctx2)
```

## Thread Safety

All operations are thread-safe and can be used in concurrent environments:

```go
// Safe for concurrent access
var wg sync.WaitGroup
wg.Add(2)

go func() {
	defer wg.Done()
	logger, _ := services.ResolveRequest[Logger]()
	// Use logger
}()

go func() {
	defer wg.Done()
	db, _ := services.ResolveTransient[Database]()
	// Use database
}()

wg.Wait()
```

## Conditional Binding

Services can be conditionally bound based on context values:

```go
// Define a predicate function
predicate := func(ctx *services.ContainerContext) (services.Lifecycle, error) {
	env := ctx.Value("environment")
	if env == "production" {
		return &ProductionService{}, nil
	}
	return &DevelopmentService{}, nil
}

// Bind with predicate
ctx := services.NewContainerContext(context.Background()).
	WithValue("environment", "production")
services.BindTransient[Service](nil, ctx, predicate)

// Resolve will return the appropriate implementation
service, _ := services.ResolveTransient[Service]()
```

## Error Handling

The container provides typed errors for better error handling:

```go
// Binding a nil service
var service *MyService
err := services.BindTransient[MyService](service, ctx)
var nilErr *services.NilServiceError
if errors.As(err, &nilErr) {
	log.Printf("Nil service error: %v", err)
}

// Circular dependency detection
_, err = services.ResolveSingleton[ServiceA]()
var circErr *services.CircularDependencyError
if errors.As(err, &circErr) {
	log.Printf("Circular dependency: %v", err)
}

// Missing request context
_, err = services.ResolveRequest[Logger]()
var missingCtxErr *services.MissingContextValueError
if errors.As(err, &missingCtxErr) {
	log.Printf("Missing context value: %v", err)
}

// Service initialization failure
_, err = services.ResolveSingleton[FailingService]()
var initErr *services.InitializationError
if errors.As(err, &initErr) {
	log.Printf("Initialization failed: %v", initErr.Unwrap())
}
```

## Web Framework Integration

The container can be easily integrated with web frameworks like Gin, Echo, or standard net/http:

```go
// Example with standard net/http
func containerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create request context with unique ID
		ctx := services.NewContainerContext(r.Context()).
			WithValue("request_id", r.Header.Get("X-Request-ID"))

		// Boot container before request
		if err := services.Boot(); err != nil {
			http.Error(w, "Container boot failed", http.StatusInternalServerError)
			return
		}

		// Call next handler with context
		next.ServeHTTP(w, r.WithContext(ctx))

		// Shutdown after request (keep singletons)
		if err := services.Shutdown(false); err != nil {
			http.Error(w, "Container shutdown failed", http.StatusInternalServerError)
			return
		}
	})
}

// Use middleware
http.Handle("/", containerMiddleware(yourHandler))
```

## Advanced Usage

### Deep Dependency Chains

```go
// Define service interfaces
type Service1 interface {
	services.Lifecycle
	GetService2() Service2
}

type Service2 interface {
	services.Lifecycle
	GetService3() Service3
}

type Service3 interface {
	services.Lifecycle
	GetValue() string
}

// Implement services
type Impl1 struct {
	svc2 Service2
}

func (s *Impl1) OnBoot(ctx *services.ContainerContext) error {
	var err error
	s.svc2, err = services.ResolveTransient[Service2]()
	return err
}

func (s *Impl1) OnShutdown(ctx *services.ContainerContext) error {
	return nil
}

func (s *Impl1) GetService2() Service2 {
	return s.svc2
}

// Similar implementations for Impl2 and Impl3...

// Bind services
services.BindTransient[Service3](&Impl3{Value: "deep"}, ctx)
services.BindTransient[Service2](&Impl2{}, ctx)
services.BindTransient[Service1](&Impl1{}, ctx)

// Resolve top-level service
svc1, _ := services.ResolveTransient[Service1]()
value := svc1.GetService2().GetService3().GetValue() // "deep"
```

### Complex Service Resolution

```go
// Define complex service with multiple dependencies
type ComplexService struct {
	DB    Database
	Cache Cache
	Logger Logger
}

func (s *ComplexService) OnBoot(ctx *services.ContainerContext) error {
	var err error
	
	s.DB, err = services.ResolveTransient[Database]()
	if err != nil {
		return err
	}
	
	s.Cache, err = services.ResolveTransient[Cache]()
	if err != nil {
		return err
	}
	
	s.Logger, err = services.ResolveTransient[Logger]()
	if err != nil {
		return err
	}
	
	return nil
}

func (s *ComplexService) OnShutdown(ctx *services.ContainerContext) error {
	return nil
}

// Bind dependencies
services.BindTransient[Database](&PostgresDB{}, ctx)
services.BindTransient[Cache](&RedisCache{}, ctx)
services.BindTransient[Logger](&FileLogger{}, ctx)

// Bind complex service
services.BindTransient[ComplexService](&ComplexService{}, ctx)

// Resolve complex service
service, _ := services.ResolveTransient[ComplexService]()
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request. See [CONTRIBUTING.md](CONTRIBUTING.md) for more details.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details. 