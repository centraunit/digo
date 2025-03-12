package services_test

import (
	"context"
	"errors"
	"testing"

	services "github.com/centraunit/goallin_services"
	"github.com/centraunit/goallin_services/mock"
	"github.com/stretchr/testify/suite"
)

type ContainerTestSuite struct {
	suite.Suite
}

func (s *ContainerTestSuite) SetupTest() {
	services.Reset()

}

func (s *ContainerTestSuite) TestBasicInitialization() {
	ctx := services.NewContainerContext(context.Background())
	services.BindTransient[mock.Database](&mock.MockDB{}, ctx)
	services.BindTransient[mock.Cache](&mock.MockCache{}, ctx)

	db, err := services.ResolveTransient[mock.Database]()
	s.NoError(err)
	s.NotNil(db)
	s.True(db.(*mock.MockDB).IsConnected(), "Database should be connected")
}

func (s *ContainerTestSuite) TestNestedDependencies() {
	ctx := services.NewContainerContext(context.Background()).
		WithValue("request_id", "nested-test")

	// All services should be bound with the same scope
	svc3 := &mock.DeepImpl3{Value: "deep"}
	svc2 := &mock.DeepImpl2{}
	svc1 := &mock.DeepImpl1{}

	err := services.BindTransient[mock.DeepService3](svc3, ctx)
	s.NoError(err)
	err = services.BindTransient[mock.DeepService2](svc2, ctx)
	s.NoError(err)
	err = services.BindTransient[mock.DeepService1](svc1, ctx)
	s.NoError(err)

	resolved, err := services.ResolveTransient[mock.DeepService1]()
	s.NoError(err)
	s.Equal("deep", resolved.GetService2().GetService3().GetValue())
}

func (s *ContainerTestSuite) TestComplexDependencyResolution() {
	ctx := services.NewContainerContext(context.Background())
	services.BindTransient[mock.Database](&mock.MockDB{}, ctx)
	services.BindTransient[mock.Cache](&mock.MockCache{}, ctx)
	services.BindTransient[mock.ComplexServiceInterface](&mock.ComplexService{}, ctx)

	service, err := services.ResolveTransient[mock.ComplexServiceInterface]()
	s.NoError(err)

	complex := service.(*mock.ComplexService)
	s.NotNil(complex.DB)
	s.NotNil(complex.Cache)
}

func (s *ContainerTestSuite) TestDeepDependencyResolution() {
	s.Run("DeepResolution", func() {
		ctx := services.NewContainerContext(context.Background())
		services.BindTransient[mock.DeepService3](&mock.DeepImpl3{Value: "deep"}, ctx)
		services.BindTransient[mock.DeepService2](&mock.DeepImpl2{}, ctx)
		services.BindTransient[mock.DeepService1](&mock.DeepImpl1{}, ctx)

		svc1, err := services.ResolveTransient[mock.DeepService1]()
		s.NoError(err)
		s.NotNil(svc1)
		s.NotNil(svc1.GetService2())
		s.NotNil(svc1.GetService2().GetService3())
		s.Equal("deep", svc1.GetService2().GetService3().GetValue())
	})

	s.Run("PartialResolutionFailure", func() {
		services.Reset()
		ctx := services.NewContainerContext(context.Background())

		services.BindTransient[mock.DeepService1](&mock.DeepImpl1{}, ctx)
		services.BindTransient[mock.DeepService2](&mock.DeepImpl2{}, ctx)

		_, err := services.ResolveTransient[mock.DeepService1]()
		var initErr *services.InitializationError
		s.True(errors.As(err, &initErr))
	})
}

func TestContainerSuite(t *testing.T) {
	suite.Run(t, new(ContainerTestSuite))
}
