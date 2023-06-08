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

	port      int
	server    *httptest.Server
	router    chi.Router
	endpoints sync.Map
}

// NewMockServer creates a MockServer with the provided options.
func NewMockServer(opts ...Option) *MockServer {
	router := chi.NewRouter()
	mockServer := &MockServer{router: router}
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
// Important: All endpoint mocks MUST be defined before calling this method.
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

// AssertExpectations verifies that every registered endpoint was called at least once.
func (ms *MockServer) AssertExpectations() {
	var failedExpectations []string
	ms.endpoints.Range(func(key, value interface{}) bool {
		endpoint := key.(string)
		callCounter := value.(*int32)
		if atomic.LoadInt32(callCounter) == 0 {
			failedExpectations = append(failedExpectations, endpoint)
		}

		return true
	})

	for _, endpoint := range failedExpectations {
		ms.T.Errorf("expected endpoint was not called: %s", endpoint)
	}
}

// AssertNotCalled verifies that the given endpoint was never called.
func (ms *MockServer) AssertNotCalled(endpoint string) {
	result, found := ms.endpoints.Load(endpoint)
	if !found {
		ms.T.Errorf("unknwon endpoint endpoint: %s", endpoint)
		return
	}

	callCounter := result.(*int32)
	if atomic.LoadInt32(callCounter) > 0 {
		ms.T.Errorf("endpoint was called when not expected: %s", endpoint)
	}
}

// AssertTimesCalled verifies how many requests where made to the endpoint.
func (ms *MockServer) AssertTimesCalled(endpoint string, expected int) {
	actual := ms.TimesCalled(endpoint)
	if actual != expected {
		ms.T.Errorf("endpoint was called %d when expected was %d: %s", actual, expected, endpoint)
		return
	}
}

// TimesCalled returns how many requests where made to the endpoint.
func (ms *MockServer) TimesCalled(endpoint string) int {
	result, found := ms.endpoints.Load(endpoint)
	if !found {
		return 0
	}

	callCounter := result.(*int32)
	return int(atomic.LoadInt32(callCounter))
}

// Get creates a mock endpoint for a get request.
func (ms *MockServer) Get(pattern string, matchers ...Matcher) *Returner {
	endpoint := endpointName(http.MethodGet, pattern)

	returner := newReturner(endpoint)
	ms.router.Get(pattern, ms.newHandler(endpoint, returner, matchers))

	return returner
}

// Post creates a mock endpoint for a post request.
func (ms *MockServer) Post(pattern string, matchers ...Matcher) *Returner {
	endpoint := endpointName(http.MethodPost, pattern)

	returner := newReturner(endpoint)
	ms.router.Post(pattern, ms.newHandler(endpoint, returner, matchers))

	return returner
}

// Put creates a mock endpoint for a put request.
func (ms *MockServer) Put(pattern string, matchers ...Matcher) *Returner {
	endpoint := endpointName(http.MethodPut, pattern)

	returner := newReturner(endpoint)
	ms.router.Put(pattern, ms.newHandler(endpoint, returner, matchers))

	return returner
}

// Patch creates a mock endpoint for a patch request.
func (ms *MockServer) Patch(pattern string, matchers ...Matcher) *Returner {
	endpoint := endpointName(http.MethodPatch, pattern)

	returner := newReturner(endpoint)
	ms.router.Patch(pattern, ms.newHandler(endpoint, returner, matchers))

	return returner
}

// Delete creates a mock endpoint for a delete request.
func (ms *MockServer) Delete(pattern string, matchers ...Matcher) *Returner {
	endpoint := endpointName(http.MethodDelete, pattern)

	returner := newReturner(endpoint)
	ms.router.Delete(pattern, ms.newHandler(endpoint, returner, matchers))

	return returner
}

// Head creates a mock endpoint for a head request.
func (ms *MockServer) Head(pattern string, matchers ...Matcher) *Returner {
	endpoint := endpointName(http.MethodHead, pattern)

	returner := newReturner(endpoint)
	ms.router.Head(pattern, ms.newHandler(endpoint, returner, matchers))

	return returner
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

func (ms *MockServer) newHandler(endpoint string, returner *Returner, matchers []Matcher) http.HandlerFunc {
	var counter int32 = 0
	ms.endpoints.Store(endpoint, &counter)

	return func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&counter, 1)

		for _, m := range matchers {
			m(ms.T, r)
		}

		returner.write(w)
	}
}

func endpointName(m, p string) string {
	return m + " " + p
}
