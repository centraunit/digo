package digo_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/centraunit/digo"
	"github.com/centraunit/digo/mock"
	"github.com/stretchr/testify/suite"
)

type ContextTestSuite struct {
	suite.Suite
}

func (s *ContextTestSuite) SetupTest() {
	digo.Shutdown(true)
}

func (s *ContextTestSuite) TestContextInheritance() {
	s.Run("ValueOverriding", func() {
		// Create a DB with global context value
		globalCtx := digo.NewContainerContext(context.Background()).
			WithValue("shared", "base-value").
			WithValue("request_id", "req-1")
		db1 := &mock.MockDB{}
		digo.BindRequest[mock.Database](db1, globalCtx)

		// Create a DB with local context that overrides global value
		localCtx := digo.NewContainerContext(context.Background()).
			WithValue("shared", "override-value").
			WithValue("request_id", "req-2")
		db2 := &mock.MockDB{}
		digo.BindRequest[mock.Database](db2, localCtx)

		// Verify context values are preserved during OnBoot
		instance, err := digo.ResolveRequest[mock.Database]()
		s.NoError(err)
		s.NotNil(instance)
		val, err := instance.(*mock.MockDB).GetContextValue("shared")
		s.NoError(err)
		s.Equal("override-value", val)
	})

	s.Run("ConditionalBindingWithContext", func() {
		ctx := digo.NewContainerContext(context.Background()).
			WithValue("env", "prod").
			WithValue("request_id", "req-1")

		prodDB := &mock.MockDB{}

		digo.BindTransient[mock.Database](prodDB, ctx, func(resolveCtx *digo.ContainerContext) (digo.Lifecycle, error) {
			val := resolveCtx.Value("env")
			if val != nil && val.(string) == "prod" {
				return prodDB, nil
			}
			return nil, fmt.Errorf("condition not met")
		})

		instance, err := digo.ResolveTransient[mock.Database]()
		s.NoError(err)
		s.NotNil(instance)
		val, err := instance.(*mock.MockDB).GetContextValue("env")
		s.NoError(err)
		s.Equal("prod", val)
	})

	s.Run("MissingRequestID", func() {
		ctx := digo.NewContainerContext(context.Background())
		db := &mock.MockDB{}
		digo.BindRequest[mock.Database](db, ctx)
		_, err := digo.ResolveRequest[mock.Database]()
		s.Error(err)
		var missingErr *digo.MissingContextValueError
		s.True(errors.As(err, &missingErr))
		s.Equal("request_id", missingErr.Key)
	})
}

func (s *ContextTestSuite) TestParent() {
	parentCtx := context.Background()
	ctx := digo.NewContainerContext(parentCtx)
	s.Equal(parentCtx, ctx.Parent(), "Parent context should be preserved")
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

	// Test merge with nil
	merged = ctx1.MergeWith(nil)
	s.Equal("value1", merged.Value("key1"))
}

func TestContextSuite(t *testing.T) {
	suite.Run(t, new(ContextTestSuite))
}
