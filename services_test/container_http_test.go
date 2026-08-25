package digo_test

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
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

func requestMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parent := r.Context()
		if id := r.Header.Get("X-Request-ID"); id != "" {
			parent = digo.WithRequestID(parent, id)
		}
		ctx, end, err := digo.RequestScope(parent)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer func() { _ = end() }()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *HTTPTestSuite) TestRequestScopeLifecycle() {
	s.Require().NoError(digo.ProvideRequest[mock.Database](func(ctx *digo.ContainerContext) (mock.Database, error) {
		return &mock.MockDB{}, nil
	}))

	handler1 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		instance, err := digo.ResolveRequest[mock.Database](r.Context())
		s.NoError(err)
		s.True(instance.(*mock.MockDB).IsConnected())
		w.WriteHeader(http.StatusOK)
	})

	handler2 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		instance, err := digo.ResolveRequest[mock.Database](r.Context())
		s.NoError(err)
		s.True(instance.(*mock.MockDB).IsConnected())
		w.WriteHeader(http.StatusOK)
	})

	server := httptest.NewServer(requestMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/handler1" {
			handler1.ServeHTTP(w, r)
		} else {
			handler2.ServeHTTP(w, r)
		}
	})))
	defer server.Close()

	req1, _ := http.NewRequest("GET", server.URL+"/handler1", nil)
	req1.Header.Set("X-Request-ID", "req-1")
	resp1, err := http.DefaultClient.Do(req1)
	s.NoError(err)
	s.Equal(http.StatusOK, resp1.StatusCode)

	req2, _ := http.NewRequest("GET", server.URL+"/handler2", nil)
	req2.Header.Set("X-Request-ID", "req-2")
	resp2, err := http.DefaultClient.Do(req2)
	s.NoError(err)
	s.Equal(http.StatusOK, resp2.StatusCode)
}

func (s *HTTPTestSuite) TestConcurrentRequestIsolation() {
	var boots atomic.Int64
	s.Require().NoError(digo.ProvideRequest[mock.Database](func(ctx *digo.ContainerContext) (mock.Database, error) {
		boots.Add(1)
		return &mock.MockDB{}, nil
	}))

	var (
		mu    sync.Mutex
		seen  = map[string]mock.Database{}
		start = make(chan struct{})
		wg    sync.WaitGroup
	)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-start
		inst, err := digo.ResolveRequest[mock.Database](r.Context())
		s.NoError(err)
		id, ok := digo.RequestID(r.Context())
		s.True(ok)
		mu.Lock()
		seen[id] = inst
		mu.Unlock()
		// hold instance briefly while peer requests also resolve
		inst2, err := digo.ResolveRequest[mock.Database](r.Context())
		s.NoError(err)
		s.Same(inst, inst2)
		w.WriteHeader(http.StatusOK)
	})

	server := httptest.NewServer(requestMiddleware(handler))
	defer server.Close()

	const n = 8
	wg.Add(n)
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			req, _ := http.NewRequest("GET", server.URL, nil)
			req.Header.Set("X-Request-ID", "req-"+string(rune('a'+i)))
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				errs <- err
				return
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				errs <- err
			}
		}(i)
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		s.NoError(err)
	}

	mu.Lock()
	defer mu.Unlock()
	s.Equal(n, len(seen))
	s.Equal(int64(n), boots.Load())
	ptrs := map[mock.Database]struct{}{}
	for _, inst := range seen {
		ptrs[inst] = struct{}{}
	}
	s.Equal(n, len(ptrs), "each request must get its own instance")
}

func (s *HTTPTestSuite) TestTransientScopeLifecycle() {
	s.Require().NoError(digo.ProvideTransient[mock.Database](func(ctx *digo.ContainerContext) (mock.Database, error) {
		return &mock.MockDB{}, nil
	}))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		instance1, err := digo.ResolveTransient[mock.Database](r.Context())
		s.NoError(err)
		s.True(instance1.(*mock.MockDB).IsConnected())

		instance2, err := digo.ResolveTransient[mock.Database](r.Context())
		s.NoError(err)
		s.NotSame(instance1, instance2)
		s.True(instance2.(*mock.MockDB).IsConnected())

		w.WriteHeader(http.StatusOK)
	})

	server := httptest.NewServer(requestMiddleware(handler))
	defer server.Close()

	req, _ := http.NewRequest("GET", server.URL, nil)
	req.Header.Set("X-Request-ID", "req-1")
	resp, err := http.DefaultClient.Do(req)
	s.NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode)
}

func (s *HTTPTestSuite) TestSingletonScopeLifecycle() {
	var globalInstance mock.Database

	s.Require().NoError(digo.ProvideSingleton[mock.Database](func(ctx *digo.ContainerContext) (mock.Database, error) {
		return &mock.MockDB{}, nil
	}))

	handler1 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		instance, err := digo.ResolveSingleton[mock.Database]()
		s.NoError(err)
		globalInstance = instance
		s.True(instance.(*mock.MockDB).IsConnected())
		w.WriteHeader(http.StatusOK)
	})

	handler2 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		instance, err := digo.ResolveSingleton[mock.Database]()
		s.NoError(err)
		s.Same(globalInstance, instance)
		s.True(instance.(*mock.MockDB).IsConnected())
		w.WriteHeader(http.StatusOK)
	})

	server := httptest.NewServer(requestMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/handler1" {
			handler1.ServeHTTP(w, r)
		} else {
			handler2.ServeHTTP(w, r)
		}
	})))
	defer server.Close()

	req1, _ := http.NewRequest("GET", server.URL+"/handler1", nil)
	req1.Header.Set("X-Request-ID", "req-1")
	resp1, err := http.DefaultClient.Do(req1)
	s.NoError(err)
	s.Equal(http.StatusOK, resp1.StatusCode)

	req2, _ := http.NewRequest("GET", server.URL+"/handler2", nil)
	req2.Header.Set("X-Request-ID", "req-2")
	resp2, err := http.DefaultClient.Do(req2)
	s.NoError(err)
	s.Equal(http.StatusOK, resp2.StatusCode)
}

func TestHTTPSuite(t *testing.T) {
	suite.Run(t, new(HTTPTestSuite))
}
