package digo_test

import (
	"context"
	"errors"
	"testing"

	"github.com/centraunit/digo"
	"github.com/centraunit/digo/mock"
	"github.com/stretchr/testify/suite"
)

type ContextTestSuite struct {
	suite.Suite
}

func (s *ContextTestSuite) SetupTest() {
	digo.Reset()
}

func (s *ContextTestSuite) TestContextInheritance() {
	s.Run("ValueOverriding", func() {
		s.Require().NoError(digo.ProvideRequest[mock.Database](func(ctx *digo.ContainerContext) (mock.Database, error) {
			return &mock.MockDB{}, nil
		}))

		ctx := digo.AsContainerContext(digo.WithRequestID(context.Background(), "req-2")).
			WithValue("shared", "override-value")

		instance, err := digo.ResolveRequest[mock.Database](ctx)
		s.NoError(err)
		s.NotNil(instance)
		val, err := instance.(*mock.MockDB).GetContextValue("shared")
		s.NoError(err)
		s.Equal("override-value", val)
	})

	s.Run("FactorySeesResolveContext", func() {
		s.Require().NoError(digo.ProvideTransient[mock.Database](func(ctx *digo.ContainerContext) (mock.Database, error) {
			val := ctx.Value("env")
			if val != nil && val.(string) == "prod" {
				return &mock.MockDB{}, nil
			}
			return nil, errors.New("condition not met")
		}))

		ctx := digo.NewContainerContext(context.Background()).WithValue("env", "prod")
		instance, err := digo.ResolveTransient[mock.Database](ctx)
		s.NoError(err)
		s.NotNil(instance)
		val, err := instance.(*mock.MockDB).GetContextValue("env")
		s.NoError(err)
		s.Equal("prod", val)
	})

	s.Run("MissingRequestID", func() {
		s.Require().NoError(digo.ProvideRequest[mock.Database](func(ctx *digo.ContainerContext) (mock.Database, error) {
			return &mock.MockDB{}, nil
		}))
		_, err := digo.ResolveRequest[mock.Database](context.Background())
		s.Error(err)
		var missingErr *digo.MissingContextValueError
		s.True(errors.As(err, &missingErr))
		s.Equal("request_id", missingErr.Key)
	})
}

func (s *ContextTestSuite) TestEmbeddedContext() {
	parentCtx := context.Background()
	ctx := digo.NewContainerContext(parentCtx)
	s.Equal(parentCtx, ctx.Context)
}

func (s *ContextTestSuite) TestMergeWith() {
	ctx1 := digo.NewContainerContext(context.Background()).
		WithValue("key1", "value1").
		WithValue("shared", "value1")

	ctx2 := digo.NewContainerContext(context.Background()).
		WithValue("key2", "value2").
		WithValue("shared", "value2")

	merged := ctx1.MergeWith(ctx2)
	s.Equal("value1", ctx1.Value("key1"))
	s.Equal("value2", ctx2.Value("key2"))
	s.Equal("value2", merged.Value("shared"), "Later context should override shared keys")

	merged = ctx1.MergeWith(nil)
	s.Equal("value1", merged.Value("key1"))
}

func TestContextSuite(t *testing.T) {
	suite.Run(t, new(ContextTestSuite))
}
