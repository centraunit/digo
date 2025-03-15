package digo_test

import (
	"context"
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
	ctx := digo.NewContainerContext(context.Background())
	db := &mock.MockDB{}
	err := digo.BindTransient[mock.Database](db, ctx)
	s.NoError(err)

	instance1, err := digo.ResolveTransient[mock.Database]()
	s.NoError(err)
	instance2, err := digo.ResolveTransient[mock.Database]()
	s.NoError(err)
	s.Same(instance1, instance2)
	s.True(instance2.(*mock.MockDB).IsConnected())
}

func (s *ResourceTestSuite) TestRequestScope() {
	db := &mock.MockDB{}
	ctx := digo.NewContainerContext(context.Background()).WithValue("request_id", "req-1")

	err := digo.BindRequest[mock.Database](db, ctx)
	s.NoError(err)

	instance1, err := digo.ResolveRequest[mock.Database]()
	s.NoError(err)
	s.True(instance1.(*mock.MockDB).IsConnected(), "OnBoot should be called")

	instance2, err := digo.ResolveRequest[mock.Database]()
	s.NoError(err)
	s.Same(instance1, instance2)

	ctx2 := digo.NewContainerContext(context.Background()).WithValue("request_id", "req-2")
	db2 := &mock.MockDB{}
	err = digo.BindRequest[mock.Database](db2, ctx2)
	s.NoError(err)

	instance3, err := digo.ResolveRequest[mock.Database]()
	s.NoError(err)
	s.NotSame(instance1, instance3)
	s.True(instance3.(*mock.MockDB).IsConnected())
}

func (s *ResourceTestSuite) TestMemoryCleanup() {
	db := &mock.MockDB{}
	ctx := digo.NewContainerContext(context.Background()).WithValue("request_id", "req-1")

	err := digo.BindRequest[mock.Database](db, ctx)
	s.NoError(err)

	instance, err := digo.ResolveRequest[mock.Database]()
	s.NoError(err)
	s.NotNil(instance)

	digo.Shutdown(true)

	_, err = digo.ResolveRequest[mock.Database]()
	s.Error(err, "Should not be able to resolve after Reset")
}

func (s *ResourceTestSuite) TestLifecycleCleanup() {
	// Test regular shutdown (keeping singletons)
	s.Run("RegularShutdown", func() {
		// Create a singleton service
		singletonDB := &mock.MockDB{}

		singletonCtx := digo.NewContainerContext(context.Background()).
			WithValue("request_id", "singleton-test")
		err := digo.BindSingleton[mock.Database](singletonDB, singletonCtx)
		s.NoError(err)

		// Create a request-scoped service
		requestDB := &mock.MockDB{}
		requestCtx := digo.NewContainerContext(context.Background()).
			WithValue("request_id", "request-test")
		err = digo.BindRequest[mock.Database](requestDB, requestCtx)
		s.NoError(err)

		// Boot both digo
		err = digo.Boot()
		s.NoError(err)

		// Verify singleton is initialized
		instance, err := digo.ResolveSingleton[mock.Database]()
		s.NoError(err)
		s.Same(singletonDB, instance)
		s.True(instance.(*mock.MockDB).IsConnected())

		// Regular shutdown - should keep singletons
		err = digo.Shutdown(false)
		s.NoError(err)

		// Should still be able to resolve singleton
		instance, err = digo.ResolveSingleton[mock.Database]()
		s.NoError(err, "Singleton should still be resolvable after regular shutdown")
		s.Same(singletonDB, instance, "Should get the same singleton instance")
		s.True(instance.(*mock.MockDB).IsConnected(), "Singleton should still be initialized")
	})

	// Test complete shutdown (clearing everything)
	s.Run("CompleteShutdown", func() {
		// Create a singleton service
		singletonDB := &mock.MockDB{}
		singletonCtx := digo.NewContainerContext(context.Background()).
			WithValue("request_id", "singleton-test")
		err := digo.BindSingleton[mock.Database](singletonDB, singletonCtx)
		s.NoError(err)

		err = digo.Boot()
		s.NoError(err)

		// Complete shutdown - should clear everything
		err = digo.Shutdown(true)
		s.NoError(err)

		// Should not be able to resolve anything
		_, err = digo.ResolveSingleton[mock.Database]()
		s.Error(err, "Nothing should be resolvable after complete shutdown")
	})
}

func TestResourceSuite(t *testing.T) {
	suite.Run(t, new(ResourceTestSuite))
}
