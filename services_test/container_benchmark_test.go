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
		ctx := digo.NewContainerContext(context.Background())
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			digo.Reset()
			db := &mock.MockDB{}
			_ = digo.BindTransient[mock.Database](db, ctx)
		}
	})

	b.Run("RequestBinding", func(b *testing.B) {
		ctx := digo.NewContainerContext(context.Background()).
			WithValue("request_id", "bench-1")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			digo.Reset()
			db := &mock.MockDB{}
			_ = digo.BindRequest[mock.Database](db, ctx)
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
		ctx := digo.NewContainerContext(context.Background())
		db := &mock.MockDB{}
		_ = digo.BindTransient[mock.Database](db, ctx)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = digo.ResolveTransient[mock.Database]()
		}
	})

	b.Run("RequestResolution", func(b *testing.B) {
		ctx := digo.NewContainerContext(context.Background()).
			WithValue("request_id", "bench-1")
		db := &mock.MockDB{}
		_ = digo.BindRequest[mock.Database](db, ctx)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = digo.ResolveRequest[mock.Database]()
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
		ctx := digo.NewContainerContext(context.Background())
		digo.BindTransient[mock.DeepService3](&mock.DeepImpl3{}, ctx)
		digo.BindTransient[mock.DeepService2](&mock.DeepImpl2{}, ctx)
		digo.BindTransient[mock.DeepService1](&mock.DeepImpl1{}, ctx)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = digo.ResolveTransient[mock.DeepService1]()
		}
	})

	b.Run("ComplexServiceResolution", func(b *testing.B) {
		ctx := digo.NewContainerContext(context.Background())
		digo.BindTransient[mock.Database](&mock.MockDB{}, ctx)
		digo.BindTransient[mock.Cache](&mock.MockCache{}, ctx)
		digo.BindTransient[mock.ComplexServiceInterface](&mock.ComplexService{}, ctx)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = digo.ResolveTransient[mock.ComplexServiceInterface]()
		}
	})
}

func BenchmarkConcurrentOperations(b *testing.B) {
	b.Run("ConcurrentResolution", func(b *testing.B) {
		ctx := digo.NewContainerContext(context.Background())
		db := &mock.MockDB{}
		_ = digo.BindTransient[mock.Database](db, ctx)
		var wg sync.WaitGroup
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			wg.Add(5)
			for j := 0; j < 5; j++ {
				go func() {
					defer wg.Done()
					_, _ = digo.ResolveTransient[mock.Database]()
				}()
			}
			wg.Wait()
		}
	})

	b.Run("ConcurrentMixedOperations", func(b *testing.B) {
		ctx := digo.NewContainerContext(context.Background())
		var wg sync.WaitGroup
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			wg.Add(5)
			// Mix of binding and resolution operations
			go func() {
				defer wg.Done()
				db := &mock.MockDB{}
				_ = digo.BindTransient[mock.Database](db, ctx)
			}()
			go func() {
				defer wg.Done()
				_, _ = digo.ResolveTransient[mock.Database]()
			}()
			go func() {
				defer wg.Done()
				cache := &mock.MockCache{}
				_ = digo.BindTransient[mock.Cache](cache, ctx)
			}()
			go func() {
				defer wg.Done()
				_, _ = digo.ResolveTransient[mock.Cache]()
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
		ctx := digo.NewContainerContext(context.Background())
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			db := &mock.MockDB{}
			_ = digo.BindTransient[mock.Database](db, ctx)
			_ = digo.Boot()
			_ = digo.Shutdown(true)
		}
	})
}
