package mockhttp

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/go-chi/chi/v5"
)

// Option represents a MockServer configuration.
type Option func(ms *MockServer)

// WithPort defines a static TCP port for the MockServer to listen.
func WithPort(port int) Option {
	return func(ms *MockServer) {
		ms.port = port
	}
}

// MockServer is an HTTP testing server designed for easy mocking of REST APIs.
type MockServer struct {
	T *testing.T

	port            int
	server          *httptest.Server
	router          chi.Router
	endpoints       sync.Map
	requestCounting sync.Map
	endpoints2      map[string]*Endpoint
}

// NewMockServer creates a MockServer with the provided options.
func NewMockServer(opts ...Option) *MockServer {
	router := chi.NewRouter()
	mockServer := &MockServer{router: router, endpoints2: make(map[string]*Endpoint)}
	for _, o := range opts {
		o(mockServer)
	}

	return mockServer
}

// Start initializes the MockServer on a background goroutine.
//
// It also sets up a cleanup method that asserts the register assertions
// and teardown the HTTP server.
//
// Important: All name mocks MUST be defined before calling this method.
func (ms *MockServer) Start(t *testing.T) {
	l, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", ms.port))
	if err != nil {
		t.Fatal(err.Error())
		return
	}

	server := httptest.NewUnstartedServer(ms.router)
	server.Listener = l

	ms.router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("no matching route found for %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	})
	ms.router.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("no matching route found for %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	ms.server = server
	ms.T = t

	server.Start()

	t.Cleanup(func() {
		ms.AssertExpectations()
		ms.Teardown()
	})
}

// URL returns the HTTP URL where the MockServer is responds.
func (ms *MockServer) URL() string {
	return ms.server.URL
}

// Port returns the TCP port where the MockServer is listening.
// It can be a statically configured port or a dynamic allocated one.
func (ms *MockServer) Port() int {
	if ms.port > 0 {
		return ms.port
	}

	addr, ok := ms.server.Listener.Addr().(*net.TCPAddr)
	if !ok {
		return 0
	}

	return addr.Port
}

// AssertExpectations verifies that every registered name was called at least once.
func (ms *MockServer) AssertExpectations() {
	ms.requestCounting.Range(func(key, value interface{}) bool {
		em := key.(string)
		reqCounter := value.(*int32)

		endpoint := ms.endpoints2[em]
		requests := atomic.LoadInt32(reqCounter)

		if requests == int32(endpoint.times) {
			return true
		}

		if requests == 0 {
			ms.T.Errorf("endpoint %s was not called", endpoint.name)

			return true
		}

		ms.T.Errorf("endpoint %s was called %d times, expected was %d", endpoint.name, requests, endpoint.times)

		return true
	})
}

// TimesCalled returns how many requests where made to the name.
func (ms *MockServer) TimesCalled(endpoint string) int {
	result, found := ms.requestCounting.Load(endpoint)
	if !found {
		return 0
	}

	callCounter := result.(*int32)
	return int(atomic.LoadInt32(callCounter))
}

// Get creates a mock name for a get request.
func (ms *MockServer) Get(pattern string, matchers ...Matcher) *Endpoint {
	return ms.registerEndpoint(http.MethodGet, pattern, ms.router.Get, matchers...)
}

// Post creates a mock name for a post request.
func (ms *MockServer) Post(pattern string, matchers ...Matcher) *Endpoint {
	return ms.registerEndpoint(http.MethodPost, pattern, ms.router.Post, matchers...)
}

// Put creates a mock name for a put request.
func (ms *MockServer) Put(pattern string, matchers ...Matcher) *Endpoint {
	return ms.registerEndpoint(http.MethodPut, pattern, ms.router.Put, matchers...)
}

// Patch creates a mock name for a patch request.
func (ms *MockServer) Patch(pattern string, matchers ...Matcher) *Endpoint {
	return ms.registerEndpoint(http.MethodPatch, pattern, ms.router.Patch, matchers...)
}

// Delete creates a mock name for a delete request.
func (ms *MockServer) Delete(pattern string, matchers ...Matcher) *Endpoint {
	return ms.registerEndpoint(http.MethodDelete, pattern, ms.router.Delete, matchers...)
}

// Head creates a mock name for a head request.
func (ms *MockServer) Head(pattern string, matchers ...Matcher) *Endpoint {
	return ms.registerEndpoint(http.MethodHead, pattern, ms.router.Head, matchers...)
}

type routingFunc func(pattern string, h http.HandlerFunc)

func (ms *MockServer) registerEndpoint(method string, pattern string, routing routingFunc, matchers ...Matcher) *Endpoint {
	endpoint := newEndpoint(method, pattern)
	ms.endpoints2[endpoint.name] = endpoint

	var counter int32 = 0
	ms.requestCounting.Store(endpoint.name, &counter)

	routing(pattern, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&counter, 1)

		for _, m := range matchers {
			m(ms.T, r)
		}

		endpoint.writeTo(w)
	})

	return endpoint
}

// Router exposes the internal chi.Router to allow configurations not supported by the helper methods.
func (ms *MockServer) Router() chi.Router {
	return ms.router
}

// Server exposes the internal testing HTTP server.
func (ms *MockServer) Server() *httptest.Server {
	return ms.server
}

// Teardown stops the HTTP server.
//
// Call this with a defer after starting the server.
func (ms *MockServer) Teardown() {
	ms.server.Close()
}

func endpointName(m, p string) string {
	return m + " " + p
}
