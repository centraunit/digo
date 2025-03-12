package services_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	services "github.com/centraunit/goallin_services"
	"github.com/centraunit/goallin_services/mock"
	"github.com/stretchr/testify/suite"
)

type EdgeCaseTestSuite struct {
	suite.Suite
}

func (s *EdgeCaseTestSuite) SetupTest() {
	services.Reset()

}

func (s *EdgeCaseTestSuite) TestContainerEdgeCases() {
	s.Run("ResetDuringResolution", func() {
		ctx := services.NewContainerContext(context.Background())
		db := &mock.MockDB{}
		err := services.BindTransient[mock.Database](db, ctx)
		s.NoError(err)

		var wg sync.WaitGroup
		errors := make(chan error, 2)

		// Start resolution
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := services.ResolveTransient[mock.Database]()
			if err != nil {
				errors <- err
			}
		}()

		// Reset during resolution
		wg.Add(1)
		go func() {
			defer wg.Done()
			services.Shutdown(true)
		}()

		wg.Wait()
		close(errors)

		for err := range errors {
			s.Error(err, "Should handle reset during resolution")
		}
	})

	s.Run("MultipleConcurrentResets", func() {
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				services.Shutdown(true)
			}()
		}
		wg.Wait()
	})

	s.Run("BootWithoutBindings", func() {
		err := services.Boot()
		s.NoError(err, "Boot should succeed with no bindings")
	})

	s.Run("MultipleBoots", func() {
		err := services.Boot()
		s.NoError(err)
		err = services.Boot()
		s.NoError(err, "Multiple boots should be safe")
	})

	s.Run("ShutdownWithoutBoot", func() {
		err := services.Shutdown(false)
		s.NoError(err, "Shutdown without boot should be safe")
	})

	s.Run("MultipleShutdowns", func() {
		err := services.Boot()
		s.NoError(err)
		err = services.Shutdown(false)
		s.NoError(err)
		err = services.Shutdown(false)
		s.NoError(err, "Multiple shutdowns should be safe")
	})
}

func (s *EdgeCaseTestSuite) TestContextEdgeCases() {
	s.Run("NilContextInBinding", func() {
		db := &mock.MockDB{}
		err := services.BindTransient[mock.Database](db, nil)
		s.NoError(err, "Should handle nil context")
	})

	s.Run("EmptyContextValues", func() {
		ctx := services.NewContainerContext(context.Background())
		db := &mock.MockDB{}
		err := services.BindRequest[mock.Database](db, ctx)
		s.NoError(err)
		_, err = services.ResolveRequest[mock.Database]()
		s.Error(err, "Should require request_id for request scope")
	})

	s.Run("ContextWithNilValues", func() {
		ctx := services.NewContainerContext(context.Background()).
			WithValue("key", nil)
		db := &mock.MockDB{}
		err := services.BindTransient[mock.Database](db, ctx)
		s.NoError(err, "Should handle nil values in context")
	})

	s.Run("ContextInheritanceWithNilParent", func() {
		var ctx1 *services.ContainerContext = nil
		ctx := services.NewContainerContext(ctx1)
		db := &mock.MockDB{}
		err := services.BindTransient[mock.Database](db, ctx)
		s.NoError(err, "Should handle nil parent context")
	})
}

// Move InvalidDB and its methods outside the test function
type InvalidDB struct{}

func (db *InvalidDB) OnBoot(ctx *services.ContainerContext) error {
	return nil
}

func (db *InvalidDB) OnShutdown(ctx *services.ContainerContext) error {
	return nil
}

// Add Connect method to satisfy mock.Database interface
func (db *InvalidDB) Connect() error {
	// This is an invalid implementation that always fails
	return fmt.Errorf("invalid database implementation")
}

func (s *EdgeCaseTestSuite) TestResolutionEdgeCases() {
	s.Run("ResolveNonExistent", func() {
		_, err := services.ResolveTransient[mock.Database]()
		var notFoundErr *services.BindingNotFoundError
		s.True(errors.As(err, &notFoundErr))
	})

	s.Run("ResolveDuringShutdown", func() {
		ctx := services.NewContainerContext(context.Background())
		db := &mock.MockDB{}
		err := services.BindTransient[mock.Database](db, ctx)
		s.NoError(err)

		var wg sync.WaitGroup
		errors := make(chan error, 2)

		// Start shutdown
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := services.Shutdown(true)
			if err != nil {
				errors <- err
			}
		}()

		// Try to resolve during shutdown
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := services.ResolveTransient[mock.Database]()
			if err != nil {
				errors <- err
			}
		}()

		wg.Wait()
		close(errors)

		for err := range errors {
			s.Error(err, "Should handle resolution during shutdown")
		}
	})
}

func (s *EdgeCaseTestSuite) TestResolveRequestEdgeCases() {
	ctx := services.NewContainerContext(context.Background())
	db := &mock.MockDB{}

	// Test resolving without request_id
	err := services.BindRequest[mock.Database](db, ctx)
	s.NoError(err)
	_, err = services.ResolveRequest[mock.Database]()
	s.Error(err)
	var missingErr *services.MissingContextValueError
	s.True(errors.As(err, &missingErr))
	s.Equal("request_id", missingErr.Key)
}

func (s *EdgeCaseTestSuite) TestResolveSingletonEdgeCases() {
	// Test initialization failure
	failingDB := &mock.FailingDB{ShouldFail: true}
	err := services.BindSingleton[mock.Database](failingDB)
	s.NoError(err)

	_, err = services.ResolveSingleton[mock.Database]()
	s.Error(err)
	s.Contains(err.Error(), "simulated boot failure")
}

func TestEdgeCaseTestSuite(t *testing.T) {
	suite.Run(t, new(EdgeCaseTestSuite))
}
