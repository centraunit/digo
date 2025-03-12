package services_test

import (
	"context"
	"testing"

	services "github.com/centraunit/goallin_services"
	"github.com/centraunit/goallin_services/mock"
	"github.com/stretchr/testify/assert"
)

func TestContainerSingleton(t *testing.T) {
	t.Run("ContainerIsSingleton", func(t *testing.T) {
		services.Shutdown(true)

		db := &mock.MockDB{}
		ctx := services.NewContainerContext(context.Background()).
			WithValue("request_id", "app-singleton")

		err := services.BindSingleton[mock.Database](db, ctx)
		assert.NoError(t, err)

		instance1, err1 := services.ResolveSingleton[mock.Database]()
		instance2, err2 := services.ResolveSingleton[mock.Database]()

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.Same(t, instance1, instance2, "Singleton should return same instance")
	})

	t.Run("SingletonStateConsistency", func(t *testing.T) {
		services.Shutdown(true)

		service := &mock.SingletonTestService{}
		ctx := services.NewContainerContext(context.Background()).
			WithValue("request_id", "app-singleton")

		err := services.BindSingleton[mock.Service](service, ctx)
		assert.NoError(t, err)

		instance1, err1 := services.ResolveSingleton[mock.Service]()
		instance2, err2 := services.ResolveSingleton[mock.Service]()

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.Same(t, instance1, instance2, "Singleton should maintain state")
	})
}
