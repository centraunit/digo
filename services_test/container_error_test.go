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
		ctx := digo.NewContainerContext(context.Background())
		db := &mock.MockDB{}
		err := digo.BindTransient[mock.Database](db, ctx)
		s.NoError(err)

		// Try to resolve - this should work
		_, err = digo.ResolveTransient[mock.Database]()
		s.NoError(err)

		// Reset and try to resolve - this should fail
		digo.Shutdown(true)
		_, err = digo.ResolveRequest[mock.Database]()
		s.Error(err)
		s.Contains(err.Error(), "no binding found")
	})

	s.Run("NilBinding", func() {
		var db *mock.MockDB
		err := digo.BindSingleton[mock.Database](db)
		var nilErr *digo.NilServiceError
		s.True(errors.As(err, &nilErr))
	})

	s.Run("MissingContextValues", func() {
		ctx := digo.NewContainerContext(context.Background())
		db := &mock.MockDB{}
		err := digo.BindRequest[mock.Database](db, ctx)
		s.NoError(err)

		// Resolve should fail because request_id is missing
		_, err = digo.ResolveRequest[mock.Database]()
		s.Error(err)
		var missingErr *digo.MissingContextValueError
		s.True(errors.As(err, &missingErr))
	})

	s.Run("RecoveryAfterFailedBoot", func() {
		failingDB := &mock.FailingDB{ShouldFail: true}
		ctx := digo.NewContainerContext(context.Background()).
			WithValue("request_id", "boot-test")

		err := digo.BindRequest[mock.Database](failingDB, ctx)
		s.NoError(err)

		// Boot should fail
		err = digo.Boot()
		s.Error(err)
		s.Contains(err.Error(), "simulated boot failure")

		// Reset and try again with working DB
		digo.Shutdown(true)
		workingDB := &mock.MockDB{}
		err = digo.BindRequest[mock.Database](workingDB, ctx)
		s.NoError(err)
		err = digo.Boot()
		s.NoError(err)
	})

	s.Run("CircularDependency", func() {
		ctx := digo.NewContainerContext(context.Background())
		err := digo.BindTransient[mock.CircularService1](&mock.CircularImpl1{}, ctx)
		s.NoError(err)
		err = digo.BindTransient[mock.CircularService2](&mock.CircularImpl2{}, ctx)
		s.NoError(err)

		// Try to resolve - should detect circular dependency
		_, err = digo.ResolveTransient[mock.CircularService1]()
		s.Error(err)
		s.Contains(err.Error(), "circular dependency")
	})
}

func TestErrorSuite(t *testing.T) {
	suite.Run(t, new(ErrorTestSuite))
}
