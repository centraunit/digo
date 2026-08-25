package digo_test

import (
	"context"
	"errors"
	"testing"

	"github.com/centraunit/digo"
	"github.com/centraunit/digo/mock"
	"github.com/stretchr/testify/suite"
)

type ContainerTestSuite struct {
	suite.Suite
}

func (s *ContainerTestSuite) SetupTest() {
	digo.Reset()

}

func (s *ContainerTestSuite) TestBasicInitialization() {
	_ = digo.NewContainerContext(context.Background())
	digo.ProvideTransient[mock.Database](func(_ *digo.ContainerContext) (mock.Database, error) { return &mock.MockDB{}, nil })
	digo.ProvideTransient[mock.Cache](func(_ *digo.ContainerContext) (mock.Cache, error) { return &mock.MockCache{}, nil })

	db, err := digo.ResolveTransient[mock.Database](context.Background())
	s.NoError(err)
	s.NotNil(db)
	s.True(db.(*mock.MockDB).IsConnected(), "Database should be connected")
}

func (s *ContainerTestSuite) TestNestedDependencies() {
	_ = digo.NewContainerContext(context.Background()).
		WithValue("request_id", "nested-test")

	// All digo should be bound with the same scope
	svc3 := &mock.DeepImpl3{Value: "deep"}
	svc2 := &mock.DeepImpl2{}
	svc1 := &mock.DeepImpl1{}

	err := digo.ProvideTransient[mock.DeepService3](func(_ *digo.ContainerContext) (mock.DeepService3, error) { return svc3, nil })
	s.NoError(err)
	err = digo.ProvideTransient[mock.DeepService2](func(_ *digo.ContainerContext) (mock.DeepService2, error) { return svc2, nil })
	s.NoError(err)
	err = digo.ProvideTransient[mock.DeepService1](func(_ *digo.ContainerContext) (mock.DeepService1, error) { return svc1, nil })
	s.NoError(err)

	resolved, err := digo.ResolveTransient[mock.DeepService1](context.Background())
	s.NoError(err)
	s.Equal("deep", resolved.GetService2().GetService3().GetValue())
}

func (s *ContainerTestSuite) TestComplexDependencyResolution() {
	_ = digo.NewContainerContext(context.Background())
	digo.ProvideTransient[mock.Database](func(_ *digo.ContainerContext) (mock.Database, error) { return &mock.MockDB{}, nil })
	digo.ProvideTransient[mock.Cache](func(_ *digo.ContainerContext) (mock.Cache, error) { return &mock.MockCache{}, nil })
	digo.ProvideTransient[mock.ComplexServiceInterface](func(_ *digo.ContainerContext) (mock.ComplexServiceInterface, error) { return &mock.ComplexService{}, nil })

	service, err := digo.ResolveTransient[mock.ComplexServiceInterface](context.Background())
	s.NoError(err)

	complex := service.(*mock.ComplexService)
	s.NotNil(complex.DB)
	s.NotNil(complex.Cache)
}

func (s *ContainerTestSuite) TestDeepDependencyResolution() {
	s.Run("DeepResolution", func() {
		_ = digo.NewContainerContext(context.Background())
		digo.ProvideTransient[mock.DeepService3](func(_ *digo.ContainerContext) (mock.DeepService3, error) { return &mock.DeepImpl3{Value: "deep"}, nil })
		digo.ProvideTransient[mock.DeepService2](func(_ *digo.ContainerContext) (mock.DeepService2, error) { return &mock.DeepImpl2{}, nil })
		digo.ProvideTransient[mock.DeepService1](func(_ *digo.ContainerContext) (mock.DeepService1, error) { return &mock.DeepImpl1{}, nil })

		svc1, err := digo.ResolveTransient[mock.DeepService1](context.Background())
		s.NoError(err)
		s.NotNil(svc1)
		s.NotNil(svc1.GetService2())
		s.NotNil(svc1.GetService2().GetService3())
		s.Equal("deep", svc1.GetService2().GetService3().GetValue())
	})

	s.Run("PartialResolutionFailure", func() {
		digo.Reset()
		_ = digo.NewContainerContext(context.Background())

		digo.ProvideTransient[mock.DeepService1](func(_ *digo.ContainerContext) (mock.DeepService1, error) { return &mock.DeepImpl1{}, nil })
		digo.ProvideTransient[mock.DeepService2](func(_ *digo.ContainerContext) (mock.DeepService2, error) { return &mock.DeepImpl2{}, nil })

		_, err := digo.ResolveTransient[mock.DeepService1](context.Background())
		var initErr *digo.InitializationError
		s.True(errors.As(err, &initErr))
	})
}

func TestContainerSuite(t *testing.T) {
	suite.Run(t, new(ContainerTestSuite))
}
