package digo_test

import (
	"testing"

	"github.com/centraunit/digo"
	"github.com/centraunit/digo/mock"
	"github.com/stretchr/testify/assert"
)

func TestContainerSingleton(t *testing.T) {
	t.Run("ContainerIsSingleton", func(t *testing.T) {
		digo.Reset()

		db := &mock.MockDB{}
		err := digo.BindSingleton[mock.Database](db)
		assert.NoError(t, err)

		instance1, err1 := digo.ResolveSingleton[mock.Database]()
		instance2, err2 := digo.ResolveSingleton[mock.Database]()

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.Same(t, instance1, instance2)
	})

	t.Run("SingletonStateConsistency", func(t *testing.T) {
		digo.Reset()

		service := &mock.SingletonTestService{}
		err := digo.BindSingleton[mock.Service](service)
		assert.NoError(t, err)

		instance1, err1 := digo.ResolveSingleton[mock.Service]()
		instance2, err2 := digo.ResolveSingleton[mock.Service]()

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.Same(t, instance1, instance2)
	})
}
