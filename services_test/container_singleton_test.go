package digo_test

import (
	"context"
	"testing"

	"github.com/centraunit/digo"
	"github.com/centraunit/digo/mock"
	"github.com/stretchr/testify/assert"
)

func TestContainerSingleton(t *testing.T) {
	t.Run("ContainerIsSingleton", func(t *testing.T) {
		digo.Shutdown(true)

		db := &mock.MockDB{}
		ctx := digo.NewContainerContext(context.Background()).
			WithValue("request_id", "app-singleton")

		err := digo.BindSingleton[mock.Database](db, ctx)
		assert.NoError(t, err)

		instance1, err1 := digo.ResolveSingleton[mock.Database]()
		instance2, err2 := digo.ResolveSingleton[mock.Database]()

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.Same(t, instance1, instance2, "Singleton should return same instance")
	})

	t.Run("SingletonStateConsistency", func(t *testing.T) {
		digo.Shutdown(true)

		service := &mock.SingletonTestService{}
		ctx := digo.NewContainerContext(context.Background()).
			WithValue("request_id", "app-singleton")

		err := digo.BindSingleton[mock.Service](service, ctx)
		assert.NoError(t, err)

		instance1, err1 := digo.ResolveSingleton[mock.Service]()
		instance2, err2 := digo.ResolveSingleton[mock.Service]()

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.Same(t, instance1, instance2, "Singleton should maintain state")
	})
}
