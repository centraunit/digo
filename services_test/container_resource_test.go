package digo_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/centraunit/digo"
	"github.com/centraunit/digo/mock"
	"github.com/stretchr/testify/suite"
)

type ResourceTestSuite struct {
	suite.Suite
}

func (s *ResourceTestSuite) SetupTest() {
	digo.Reset()
}

func (s *ResourceTestSuite) TestTransientScope() {
	s.Require().NoError(digo.ProvideTransient[mock.Database](func(ctx *digo.ContainerContext) (mock.Database, error) {
		return &mock.MockDB{}, nil
	}))

	instance1, err := digo.ResolveTransient[mock.Database](context.Background())
	s.NoError(err)
	instance2, err := digo.ResolveTransient[mock.Database](context.Background())
	s.NoError(err)
	s.NotSame(instance1, instance2)
	s.True(instance1.(*mock.MockDB).IsConnected())
	s.True(instance2.(*mock.MockDB).IsConnected())
}

func (s *ResourceTestSuite) TestRequestScope() {
	s.Require().NoError(digo.ProvideRequest[mock.Database](func(ctx *digo.ContainerContext) (mock.Database, error) {
		return &mock.MockDB{}, nil
	}))

	ctx1 := digo.WithRequestID(context.Background(), "req-1")
	instance1, err := digo.ResolveRequest[mock.Database](ctx1)
	s.NoError(err)
	s.True(instance1.(*mock.MockDB).IsConnected())

	instance2, err := digo.ResolveRequest[mock.Database](ctx1)
	s.NoError(err)
	s.Same(instance1, instance2)

	ctx2 := digo.WithRequestID(context.Background(), "req-2")
	instance3, err := digo.ResolveRequest[mock.Database](ctx2)
	s.NoError(err)
	s.NotSame(instance1, instance3)
	s.True(instance3.(*mock.MockDB).IsConnected())
}

func (s *ResourceTestSuite) TestConcurrentSingletonBootOnce() {
	var boots atomic.Int64
	s.Require().NoError(digo.ProvideSingleton[mock.Database](func(ctx *digo.ContainerContext) (mock.Database, error) {
		boots.Add(1)
		return &mock.MockDB{}, nil
	}))

	var wg sync.WaitGroup
	const n = 32
	instances := make([]mock.Database, n)
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			inst, err := digo.ResolveSingleton[mock.Database]()
			s.NoError(err)
			instances[i] = inst
		}(i)
	}
	wg.Wait()
	s.Equal(int64(1), boots.Load())
	for i := 1; i < n; i++ {
		s.Same(instances[0], instances[i])
	}
}

func (s *ResourceTestSuite) TestMemoryCleanup() {
	s.Require().NoError(digo.ProvideRequest[mock.Database](func(ctx *digo.ContainerContext) (mock.Database, error) {
		return &mock.MockDB{}, nil
	}))

	ctx := digo.WithRequestID(context.Background(), "req-1")
	instance, err := digo.ResolveRequest[mock.Database](ctx)
	s.NoError(err)
	s.NotNil(instance)

	digo.Shutdown(true)

	_, err = digo.ResolveRequest[mock.Database](ctx)
	s.Error(err)
}

func (s *ResourceTestSuite) TestLifecycleCleanup() {
	s.Run("RegularShutdown", func() {
		singletonDB := &mock.MockDB{}
		err := digo.BindSingleton[mock.Database](singletonDB)
		s.NoError(err)

		s.Require().NoError(digo.ProvideRequest[mock.Database](func(ctx *digo.ContainerContext) (mock.Database, error) {
			return &mock.MockDB{}, nil
		}))

		err = digo.Boot()
		s.NoError(err)

		instance, err := digo.ResolveSingleton[mock.Database]()
		s.NoError(err)
		s.Same(singletonDB, instance)
		s.True(instance.(*mock.MockDB).IsConnected())

		reqCtx := digo.WithRequestID(context.Background(), "request-test")
		_, err = digo.ResolveRequest[mock.Database](reqCtx)
		s.NoError(err)

		err = digo.Shutdown(false)
		s.NoError(err)

		instance, err = digo.ResolveSingleton[mock.Database]()
		s.NoError(err)
		s.Same(singletonDB, instance)
		s.True(instance.(*mock.MockDB).IsConnected())
	})

	s.Run("CompleteShutdown", func() {
		digo.Reset()
		singletonDB := &mock.MockDB{}
		err := digo.BindSingleton[mock.Database](singletonDB)
		s.NoError(err)

		err = digo.Boot()
		s.NoError(err)

		err = digo.Shutdown(true)
		s.NoError(err)

		_, err = digo.ResolveSingleton[mock.Database]()
		s.Error(err)
	})
}

func TestResourceSuite(t *testing.T) {
	suite.Run(t, new(ResourceTestSuite))
}
