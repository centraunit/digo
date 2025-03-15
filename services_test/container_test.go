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
	ctx := digo.NewContainerContext(context.Background())
	digo.BindTransient[mock.Database](&mock.MockDB{}, ctx)
	digo.BindTransient[mock.Cache](&mock.MockCache{}, ctx)

	db, err := digo.ResolveTransient[mock.Database]()
	s.NoError(err)
	s.NotNil(db)
	s.True(db.(*mock.MockDB).IsConnected(), "Database should be connected")
}

func (s *ContainerTestSuite) TestNestedDependencies() {
	ctx := digo.NewContainerContext(context.Background()).
		WithValue("request_id", "nested-test")

	// All digo should be bound with the same scope
	svc3 := &mock.DeepImpl3{Value: "deep"}
	svc2 := &mock.DeepImpl2{}
	svc1 := &mock.DeepImpl1{}

	err := digo.BindTransient[mock.DeepService3](svc3, ctx)
	s.NoError(err)
	err = digo.BindTransient[mock.DeepService2](svc2, ctx)
	s.NoError(err)
	err = digo.BindTransient[mock.DeepService1](svc1, ctx)
	s.NoError(err)

	resolved, err := digo.ResolveTransient[mock.DeepService1]()
	s.NoError(err)
	s.Equal("deep", resolved.GetService2().GetService3().GetValue())
}

func (s *ContainerTestSuite) TestComplexDependencyResolution() {
	ctx := digo.NewContainerContext(context.Background())
	digo.BindTransient[mock.Database](&mock.MockDB{}, ctx)
	digo.BindTransient[mock.Cache](&mock.MockCache{}, ctx)
	digo.BindTransient[mock.ComplexServiceInterface](&mock.ComplexService{}, ctx)

	service, err := digo.ResolveTransient[mock.ComplexServiceInterface]()
	s.NoError(err)

	complex := service.(*mock.ComplexService)
	s.NotNil(complex.DB)
	s.NotNil(complex.Cache)
}

func (s *ContainerTestSuite) TestDeepDependencyResolution() {
	s.Run("DeepResolution", func() {
		ctx := digo.NewContainerContext(context.Background())
		digo.BindTransient[mock.DeepService3](&mock.DeepImpl3{Value: "deep"}, ctx)
		digo.BindTransient[mock.DeepService2](&mock.DeepImpl2{}, ctx)
		digo.BindTransient[mock.DeepService1](&mock.DeepImpl1{}, ctx)

		svc1, err := digo.ResolveTransient[mock.DeepService1]()
		s.NoError(err)
		s.NotNil(svc1)
		s.NotNil(svc1.GetService2())
		s.NotNil(svc1.GetService2().GetService3())
		s.Equal("deep", svc1.GetService2().GetService3().GetValue())
	})

	s.Run("PartialResolutionFailure", func() {
		digo.Reset()
		ctx := digo.NewContainerContext(context.Background())

		digo.BindTransient[mock.DeepService1](&mock.DeepImpl1{}, ctx)
		digo.BindTransient[mock.DeepService2](&mock.DeepImpl2{}, ctx)

		_, err := digo.ResolveTransient[mock.DeepService1]()
		var initErr *digo.InitializationError
		s.True(errors.As(err, &initErr))
	})
}

func TestContainerSuite(t *testing.T) {
	suite.Run(t, new(ContainerTestSuite))
}
