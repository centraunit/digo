package digo_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/centraunit/digo"
	"github.com/centraunit/digo/mock"
	"github.com/stretchr/testify/suite"
)

type HTTPTestSuite struct {
	suite.Suite
}

func (s *HTTPTestSuite) SetupTest() {
	digo.Reset()
}

// Middleware to handle container lifecycle
func containerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create request context with unique ID
		ctx := digo.NewContainerContext(r.Context()).
			WithValue("request_id", r.Header.Get("X-Request-ID"))

		// Boot container before request
		if err := digo.Boot(); err != nil {
			http.Error(w, "Container boot failed", http.StatusInternalServerError)
			return
		}

		// Call next handler
		next.ServeHTTP(w, r.WithContext(ctx))

		// Shutdown after request (keep singletons)
		if err := digo.Shutdown(false); err != nil {
			http.Error(w, "Container shutdown failed", http.StatusInternalServerError)
			return
		}
	})
}

func (s *HTTPTestSuite) TestRequestScopeLifecycle() {
	// Create handlers that use different scopes
	handler1 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Bind and resolve request-scoped service
		db := &mock.MockDB{}
		err := digo.BindRequest[mock.Database](db, r.Context().(*digo.ContainerContext))
		s.NoError(err)

		instance, err := digo.ResolveRequest[mock.Database]()
		s.NoError(err)
		s.True(instance.(*mock.MockDB).IsConnected())

		w.WriteHeader(http.StatusOK)
	})

	handler2 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to resolve the same request-scoped service (should be different instance)
		instance, err := digo.ResolveRequest[mock.Database]()
		s.Error(err) // Should fail as it's a new request
		s.Nil(instance)

		w.WriteHeader(http.StatusOK)
	})

	// Create test server with middleware
	server := httptest.NewServer(containerMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/handler1" {
			handler1.ServeHTTP(w, r)
		} else {
			handler2.ServeHTTP(w, r)
		}
	})))
	defer server.Close()

	// Make first request
	req1, _ := http.NewRequest("GET", server.URL+"/handler1", nil)
	req1.Header.Set("X-Request-ID", "req-1")
	resp1, err := http.DefaultClient.Do(req1)
	s.NoError(err)
	s.Equal(http.StatusOK, resp1.StatusCode)

	// Make second request
	req2, _ := http.NewRequest("GET", server.URL+"/handler2", nil)
	req2.Header.Set("X-Request-ID", "req-2")
	resp2, err := http.DefaultClient.Do(req2)
	s.NoError(err)
	s.Equal(http.StatusOK, resp2.StatusCode)
}

func (s *HTTPTestSuite) TestTransientScopeLifecycle() {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Bind transient service
		db := &mock.MockDB{}
		err := digo.BindTransient[mock.Database](db, r.Context().(*digo.ContainerContext))
		s.NoError(err)

		// Resolve multiple times - should be same instance but reinitialized
		instance1, err := digo.ResolveTransient[mock.Database]()
		s.NoError(err)
		s.True(instance1.(*mock.MockDB).IsConnected())

		instance2, err := digo.ResolveTransient[mock.Database]()
		s.NoError(err)
		s.Same(instance1, instance2)
		s.True(instance2.(*mock.MockDB).IsConnected())

		w.WriteHeader(http.StatusOK)
	})

	server := httptest.NewServer(containerMiddleware(handler))
	defer server.Close()

	req, _ := http.NewRequest("GET", server.URL, nil)
	req.Header.Set("X-Request-ID", "req-1")
	resp, err := http.DefaultClient.Do(req)
	s.NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode)
}

func (s *HTTPTestSuite) TestSingletonScopeLifecycle() {
	var globalInstance mock.Database

	handler1 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Bind singleton in first request
		db := &mock.MockDB{}
		err := digo.BindSingleton[mock.Database](db)
		s.NoError(err)

		instance, err := digo.ResolveSingleton[mock.Database]()
		s.NoError(err)
		globalInstance = instance
		s.True(instance.(*mock.MockDB).IsConnected())

		w.WriteHeader(http.StatusOK)
	})

	handler2 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Resolve singleton in second request - should be same instance
		instance, err := digo.ResolveSingleton[mock.Database]()
		s.NoError(err)
		s.Same(globalInstance, instance)
		s.True(instance.(*mock.MockDB).IsConnected())

		w.WriteHeader(http.StatusOK)
	})

	server := httptest.NewServer(containerMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/handler1" {
			handler1.ServeHTTP(w, r)
		} else {
			handler2.ServeHTTP(w, r)
		}
	})))
	defer server.Close()

	// First request binds singleton
	req1, _ := http.NewRequest("GET", server.URL+"/handler1", nil)
	req1.Header.Set("X-Request-ID", "req-1")
	resp1, err := http.DefaultClient.Do(req1)
	s.NoError(err)
	s.Equal(http.StatusOK, resp1.StatusCode)

	// Second request uses same singleton
	req2, _ := http.NewRequest("GET", server.URL+"/handler2", nil)
	req2.Header.Set("X-Request-ID", "req-2")
	resp2, err := http.DefaultClient.Do(req2)
	s.NoError(err)
	s.Equal(http.StatusOK, resp2.StatusCode)
}

func TestHTTPSuite(t *testing.T) {
	suite.Run(t, new(HTTPTestSuite))
}
