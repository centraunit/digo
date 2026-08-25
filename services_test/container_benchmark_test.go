package digo_test

import (
	"context"
	"sync"
	"testing"

	"github.com/centraunit/digo"
	"github.com/centraunit/digo/mock"
)

func BenchmarkBinding(b *testing.B) {
	b.Run("TransientBinding", func(b *testing.B) {
		_ = digo.NewContainerContext(context.Background())
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			digo.Reset()
			db := &mock.MockDB{}
			_ = digo.ProvideTransient[mock.Database](func(_ *digo.ContainerContext) (mock.Database, error) { return db, nil })
		}
	})

	b.Run("RequestBinding", func(b *testing.B) {
		_ = digo.NewContainerContext(context.Background()).
			WithValue("request_id", "bench-1")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			digo.Reset()
			db := &mock.MockDB{}
			_ = digo.ProvideRequest[mock.Database](func(_ *digo.ContainerContext) (mock.Database, error) { return db, nil })
		}
	})

	b.Run("SingletonBinding", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			digo.Reset()
			db := &mock.MockDB{}
			_ = digo.BindSingleton[mock.Database](db)
		}
	})
}

func BenchmarkResolution(b *testing.B) {
	b.Run("TransientResolution", func(b *testing.B) {
		_ = digo.NewContainerContext(context.Background())
		db := &mock.MockDB{}
		_ = digo.ProvideTransient[mock.Database](func(_ *digo.ContainerContext) (mock.Database, error) { return db, nil })
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = digo.ResolveTransient[mock.Database](context.Background())
		}
	})

	b.Run("RequestResolution", func(b *testing.B) {
		digo.Reset()
		_ = digo.ProvideRequest[mock.Database](func(ctx *digo.ContainerContext) (mock.Database, error) {
			return &mock.MockDB{}, nil
		})
		ctx := digo.WithRequestID(context.Background(), "bench-1")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = digo.ResolveRequest[mock.Database](ctx)
		}
	})

	b.Run("SingletonResolution", func(b *testing.B) {
		db := &mock.MockDB{}
		_ = digo.BindSingleton[mock.Database](db)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = digo.ResolveSingleton[mock.Database]()
		}
	})
}

func BenchmarkComplexResolution(b *testing.B) {
	b.Run("DeepDependencyChain", func(b *testing.B) {
		_ = digo.NewContainerContext(context.Background())
		digo.ProvideTransient[mock.DeepService3](func(_ *digo.ContainerContext) (mock.DeepService3, error) { return &mock.DeepImpl3{}, nil })
		digo.ProvideTransient[mock.DeepService2](func(_ *digo.ContainerContext) (mock.DeepService2, error) { return &mock.DeepImpl2{}, nil })
		digo.ProvideTransient[mock.DeepService1](func(_ *digo.ContainerContext) (mock.DeepService1, error) { return &mock.DeepImpl1{}, nil })
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = digo.ResolveTransient[mock.DeepService1](context.Background())
		}
	})

	b.Run("ComplexServiceResolution", func(b *testing.B) {
		_ = digo.NewContainerContext(context.Background())
		digo.ProvideTransient[mock.Database](func(_ *digo.ContainerContext) (mock.Database, error) { return &mock.MockDB{}, nil })
		digo.ProvideTransient[mock.Cache](func(_ *digo.ContainerContext) (mock.Cache, error) { return &mock.MockCache{}, nil })
		digo.ProvideTransient[mock.ComplexServiceInterface](func(_ *digo.ContainerContext) (mock.ComplexServiceInterface, error) { return &mock.ComplexService{}, nil })
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = digo.ResolveTransient[mock.ComplexServiceInterface](context.Background())
		}
	})
}

func BenchmarkConcurrentOperations(b *testing.B) {
	b.Run("ConcurrentResolution", func(b *testing.B) {
		_ = digo.NewContainerContext(context.Background())
		db := &mock.MockDB{}
		_ = digo.ProvideTransient[mock.Database](func(_ *digo.ContainerContext) (mock.Database, error) { return db, nil })
		var wg sync.WaitGroup
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			wg.Add(5)
			for j := 0; j < 5; j++ {
				go func() {
					defer wg.Done()
					_, _ = digo.ResolveTransient[mock.Database](context.Background())
				}()
			}
			wg.Wait()
		}
	})

	b.Run("ConcurrentMixedOperations", func(b *testing.B) {
		_ = digo.NewContainerContext(context.Background())
		var wg sync.WaitGroup
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			wg.Add(5)
			// Mix of binding and resolution operations
			go func() {
				defer wg.Done()
				db := &mock.MockDB{}
				_ = digo.ProvideTransient[mock.Database](func(_ *digo.ContainerContext) (mock.Database, error) { return db, nil })
			}()
			go func() {
				defer wg.Done()
				_, _ = digo.ResolveTransient[mock.Database](context.Background())
			}()
			go func() {
				defer wg.Done()
				cache := &mock.MockCache{}
				_ = digo.ProvideTransient[mock.Cache](func(_ *digo.ContainerContext) (mock.Cache, error) { return cache, nil })
			}()
			go func() {
				defer wg.Done()
				_, _ = digo.ResolveTransient[mock.Cache](context.Background())
			}()
			go func() {
				defer wg.Done()
				digo.Reset()
			}()
			wg.Wait()
		}
	})
}

func BenchmarkContextOperations(b *testing.B) {
	b.Run("ContextCreation", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = digo.NewContainerContext(context.Background())
		}
	})

	b.Run("ContextWithValue", func(b *testing.B) {
		ctx := digo.NewContainerContext(context.Background())
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = ctx.WithValue("key", "value")
		}
	})

	b.Run("ContextMerge", func(b *testing.B) {
		ctx1 := digo.NewContainerContext(context.Background()).
			WithValue("key1", "value1")
		ctx2 := digo.NewContainerContext(context.Background()).
			WithValue("key2", "value2")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = ctx1.MergeWith(ctx2)
		}
	})
}

func BenchmarkLifecycleOperations(b *testing.B) {
	b.Run("ContainerBoot", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			digo.Reset()
			db := &mock.MockDB{}
			_ = digo.BindSingleton[mock.Database](db)
			_ = digo.Boot()
		}
	})

	b.Run("ContainerShutdown", func(b *testing.B) {
		_ = digo.NewContainerContext(context.Background())
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			db := &mock.MockDB{}
			_ = digo.ProvideTransient[mock.Database](func(_ *digo.ContainerContext) (mock.Database, error) { return db, nil })
			_ = digo.Boot()
			_ = digo.Shutdown(true)
		}
	})
}
