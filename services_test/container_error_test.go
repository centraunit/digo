package services_test

import (
	"context"
	"errors"
	"testing"

	services "github.com/centraunit/goallin_services"
	"github.com/centraunit/goallin_services/mock"
	"github.com/stretchr/testify/suite"
)

type ErrorTestSuite struct {
	suite.Suite
}

func (s *ErrorTestSuite) SetupTest() {
	services.Reset()

}

func (s *ErrorTestSuite) TestErrorCases() {
	s.Run("InvalidScope", func() {
		ctx := services.NewContainerContext(context.Background())
		db := &mock.MockDB{}
		err := services.BindTransient[mock.Database](db, ctx)
		s.NoError(err)

		// Try to resolve - this should work
		_, err = services.ResolveTransient[mock.Database]()
		s.NoError(err)

		// Reset and try to resolve - this should fail
		services.Shutdown(true)
		_, err = services.ResolveRequest[mock.Database]()
		s.Error(err)
		s.Contains(err.Error(), "no binding found")
	})

	s.Run("NilBinding", func() {
		var db *mock.MockDB
		err := services.BindSingleton[mock.Database](db)
		var nilErr *services.NilServiceError
		s.True(errors.As(err, &nilErr))
	})

	s.Run("MissingContextValues", func() {
		ctx := services.NewContainerContext(context.Background())
		db := &mock.MockDB{}
		err := services.BindRequest[mock.Database](db, ctx)
		s.NoError(err)

		// Resolve should fail because request_id is missing
		_, err = services.ResolveRequest[mock.Database]()
		s.Error(err)
		var missingErr *services.MissingContextValueError
		s.True(errors.As(err, &missingErr))
	})

	s.Run("RecoveryAfterFailedBoot", func() {
		failingDB := &mock.FailingDB{ShouldFail: true}
		ctx := services.NewContainerContext(context.Background()).
			WithValue("request_id", "boot-test")

		err := services.BindRequest[mock.Database](failingDB, ctx)
		s.NoError(err)

		// Boot should fail
		err = services.Boot()
		s.Error(err)
		s.Contains(err.Error(), "simulated boot failure")

		// Reset and try again with working DB
		services.Shutdown(true)
		workingDB := &mock.MockDB{}
		err = services.BindRequest[mock.Database](workingDB, ctx)
		s.NoError(err)
		err = services.Boot()
		s.NoError(err)
	})

	s.Run("CircularDependency", func() {
		ctx := services.NewContainerContext(context.Background())
		err := services.BindTransient[mock.CircularService1](&mock.CircularImpl1{}, ctx)
		s.NoError(err)
		err = services.BindTransient[mock.CircularService2](&mock.CircularImpl2{}, ctx)
		s.NoError(err)

		// Try to resolve - should detect circular dependency
		_, err = services.ResolveTransient[mock.CircularService1]()
		s.Error(err)
		s.Contains(err.Error(), "circular dependency")
	})
}

func TestErrorSuite(t *testing.T) {
	suite.Run(t, new(ErrorTestSuite))
}
