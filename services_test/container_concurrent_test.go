package digo_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/centraunit/digo"
	"github.com/centraunit/digo/mock"
	"github.com/stretchr/testify/suite"
)

type ConcurrentTestSuite struct {
	suite.Suite
}

func (s *ConcurrentTestSuite) SetupTest() {
	digo.Reset()
}

func (s *ConcurrentTestSuite) TestConcurrentAccess() {
	var wg sync.WaitGroup
	errors := make(chan error, 10)

	// Bind service once before starting goroutines
	ctx := digo.NewContainerContext(context.Background())
	db := &mock.MockDB{}
	err := digo.BindTransient[mock.Database](db, ctx)
	s.NoError(err)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			// Each goroutine just resolves the service
			instance, err := digo.ResolveTransient[mock.Database]()
			if err != nil {
				errors <- err
				return
			}
			// Verify the instance is working
			s.True(instance.(*mock.MockDB).IsConnected())
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		s.NoError(err)
	}
}

func (s *ConcurrentTestSuite) TestConcurrentConditionalBindings() {
	// Test that conditional binding gets overwritten
	s.Run("BindingOverwrite", func() {
		ctx := digo.NewContainerContext(context.Background()).
			WithValue("key", "value-1").
			WithValue("request_id", "req-1")
		ctx2 := digo.NewContainerContext(context.Background()).
			WithValue("key", "value-1").
			WithValue("request_id", "req-2")

		db1 := &mock.MockDB{}
		db2 := &mock.MockDB{}

		// Register first conditional binding
		digo.BindTransient[mock.Database](db1, ctx, func(resolveCtx *digo.ContainerContext) (digo.Lifecycle, error) {
			return db1, nil
		})

		// This should overwrite the previous binding
		digo.BindTransient[mock.Database](db2, ctx2, func(resolveCtx *digo.ContainerContext) (digo.Lifecycle, error) {
			return db2, nil
		})

		// Resolve should return db2, not db1
		instance, err := digo.ResolveTransient[mock.Database]()
		s.NoError(err)
		reqId, err := instance.GetContextValue("request_id")
		s.NoError(err)
		s.Equal(db2.RequestID, reqId)
	})

	// Test concurrent access to conditional binding
	s.Run("ConcurrentAccess", func() {
		var wg sync.WaitGroup
		errors := make(chan error, 10)

		ctx := digo.NewContainerContext(context.Background()).
			WithValue("key", "test-value").
			WithValue("request_id", "req-1")

		db := &mock.MockDB{}

		fmt.Println("binding db")
		digo.BindTransient[mock.Database](db, ctx, func(resolveCtx *digo.ContainerContext) (digo.Lifecycle, error) {
			val := resolveCtx.Value("key")
			if val != nil && val.(string) == "test-value" {
				return db, nil
			}
			return nil, fmt.Errorf("condition not met")
		})

		// Test concurrent resolutions
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				fmt.Printf("resolving db %d\n", id)
				instance, err := digo.ResolveTransient[mock.Database]()
				if err != nil {
					errors <- err
					return
				}

				if instance != db {
					errors <- fmt.Errorf("wrong instance returned")
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		for err := range errors {
			s.NoError(err)
		}
	})

}

func TestConcurrentSuite(t *testing.T) {
	suite.Run(t, new(ConcurrentTestSuite))
}
