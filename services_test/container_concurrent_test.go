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

	s.Require().NoError(digo.ProvideTransient[mock.Database](func(ctx *digo.ContainerContext) (mock.Database, error) {
		return &mock.MockDB{}, nil
	}))

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			instance, err := digo.ResolveTransient[mock.Database](context.Background())
			if err != nil {
				errors <- err
				return
			}
			s.True(instance.(*mock.MockDB).IsConnected())
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		s.NoError(err)
	}
}

func (s *ConcurrentTestSuite) TestProviderOverwrite() {
	db1 := &mock.MockDB{}
	db2 := &mock.MockDB{}

	s.Require().NoError(digo.ProvideTransient[mock.Database](func(ctx *digo.ContainerContext) (mock.Database, error) {
		return db1, nil
	}))
	s.Require().NoError(digo.ProvideTransient[mock.Database](func(ctx *digo.ContainerContext) (mock.Database, error) {
		return db2, nil
	}))

	instance, err := digo.ResolveTransient[mock.Database](context.Background())
	s.NoError(err)
	s.Same(db2, instance)
}

func (s *ConcurrentTestSuite) TestConcurrentSameProvider() {
	var wg sync.WaitGroup
	errors := make(chan error, 10)

	db := &mock.MockDB{}
	s.Require().NoError(digo.ProvideTransient[mock.Database](func(ctx *digo.ContainerContext) (mock.Database, error) {
		return db, nil
	}))

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			instance, err := digo.ResolveTransient[mock.Database](context.Background())
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
}

func TestConcurrentSuite(t *testing.T) {
	suite.Run(t, new(ConcurrentTestSuite))
}
