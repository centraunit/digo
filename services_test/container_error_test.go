package digo_test

import (
	"context"
	"errors"
	"testing"

	"github.com/centraunit/digo"
	"github.com/centraunit/digo/mock"
	"github.com/stretchr/testify/suite"
)

type ErrorTestSuite struct {
	suite.Suite
}

func (s *ErrorTestSuite) SetupTest() {
	digo.Reset()

}

func (s *ErrorTestSuite) TestErrorCases() {
	s.Run("InvalidScope", func() {
		_ = digo.NewContainerContext(context.Background())
		db := &mock.MockDB{}
		err := digo.ProvideTransient[mock.Database](func(_ *digo.ContainerContext) (mock.Database, error) { return db, nil })
		s.NoError(err)

		// Try to resolve - this should work
		_, err = digo.ResolveTransient[mock.Database](context.Background())
		s.NoError(err)

		// Reset and try to resolve - this should fail
		digo.Shutdown(true)
		_, err = digo.ResolveTransient[mock.Database](context.Background())
		s.Error(err)
		s.Contains(err.Error(), "digo: no binding")
	})

	s.Run("NilBinding", func() {
		var db *mock.MockDB
		err := digo.BindSingleton[mock.Database](db)
		var nilErr *digo.NilServiceError
		s.True(errors.As(err, &nilErr))
	})

	s.Run("MissingContextValues", func() {
		_ = digo.NewContainerContext(context.Background())
		db := &mock.MockDB{}
		err := digo.ProvideRequest[mock.Database](func(_ *digo.ContainerContext) (mock.Database, error) { return db, nil })
		s.NoError(err)

		// Resolve should fail because request_id is missing
		_, err = digo.ResolveRequest[mock.Database](context.Background())
		s.Error(err)
		var missingErr *digo.MissingContextValueError
		s.True(errors.As(err, &missingErr))
	})

	s.Run("RecoveryAfterFailedBoot", func() {
		failingDB := &mock.FailingDB{ShouldFail: true}
		err := digo.BindSingleton[mock.Database](failingDB)
		s.NoError(err)

		err = digo.Boot()
		s.Error(err)
		s.Contains(err.Error(), "simulated boot failure")

		digo.Reset()
		workingDB := &mock.MockDB{}
		err = digo.BindSingleton[mock.Database](workingDB)
		s.NoError(err)
		err = digo.Boot()
		s.NoError(err)
	})

	s.Run("CircularDependency", func() {
		_ = digo.NewContainerContext(context.Background())
		err := digo.ProvideTransient[mock.CircularService1](func(_ *digo.ContainerContext) (mock.CircularService1, error) { return &mock.CircularImpl1{}, nil })
		s.NoError(err)
		err = digo.ProvideTransient[mock.CircularService2](func(_ *digo.ContainerContext) (mock.CircularService2, error) { return &mock.CircularImpl2{}, nil })
		s.NoError(err)

		// Try to resolve - should detect digo: circular dependency
		_, err = digo.ResolveTransient[mock.CircularService1](context.Background())
		s.Error(err)
		s.Contains(err.Error(), "digo: circular dependency")
	})
}

func TestErrorSuite(t *testing.T) {
	suite.Run(t, new(ErrorTestSuite))
}
