package mockhttp

import (
    "fmt"
    "net"
    "net/http"
    "net/http/httptest"
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
    endpoints map[string]*Endpoint
}

// NewMockServer creates a MockServer with the provided options.
func NewMockServer(opts ...Option) *MockServer {
    mockServer := &MockServer{endpoints: make(map[string]*Endpoint)}
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
    t.Helper()

    l, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", ms.port))
    if err != nil {
        t.Fatal(err.Error())
        return
    }

    router := chi.NewRouter()

    routingFuncs := map[string]routingFunc{
        http.MethodGet:     router.Get,
        http.MethodPost:    router.Post,
        http.MethodPut:     router.Put,
        http.MethodPatch:   router.Patch,
        http.MethodDelete:  router.Delete,
        http.MethodHead:    router.Head,
        http.MethodOptions: router.Options,
    }

    for _, endpoint := range ms.endpoints {
        routing := routingFuncs[endpoint.method]

        routing(endpoint.path, endpoint.Handler(t))
    }

    server := httptest.NewUnstartedServer(router)
    server.Listener = l

    router.NotFound(func(w http.ResponseWriter, r *http.Request) {
        t.Errorf("no matching route found for %s %s", r.Method, r.URL.Path)
        w.WriteHeader(http.StatusNotFound)
    })
    router.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
        t.Errorf("no matching route found for %s %s", r.Method, r.URL.Path)
        w.WriteHeader(http.StatusMethodNotAllowed)
    })

    ms.router = router
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
    return fmt.Sprintf("http://127.0.0.1:%d", ms.Port())
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
    for _, endpoint := range ms.endpoints {
        for _, scenario := range endpoint.scenarios {
            if int(scenario.executionCount) == scenario.times {
                break
            }

            if scenario.executionCount == 0 {
                ms.T.Errorf("endpoint %s was not called", endpoint.Name())

                break
            }

            ms.T.Errorf("endpoint %s was called %d times, expected was %d", endpoint.Name(), scenario.executionCount, scenario.times)
        }
    }
}

// Get creates a mock name for a get request.
func (ms *MockServer) Get(pattern string, matchers ...Matcher) *Scenario {
    return ms.registerEndpoint(http.MethodGet, pattern, matchers...)
}

// Post creates a mock name for a post request.
func (ms *MockServer) Post(pattern string, matchers ...Matcher) *Scenario {
    return ms.registerEndpoint(http.MethodPost, pattern, matchers...)
}

// Put creates a mock name for a put request.
func (ms *MockServer) Put(pattern string, matchers ...Matcher) *Scenario {
    return ms.registerEndpoint(http.MethodPut, pattern, matchers...)
}

// Patch creates a mock name for a patch request.
func (ms *MockServer) Patch(pattern string, matchers ...Matcher) *Scenario {
    return ms.registerEndpoint(http.MethodPatch, pattern, matchers...)
}

// Delete creates a mock name for a delete request.
func (ms *MockServer) Delete(pattern string, matchers ...Matcher) *Scenario {
    return ms.registerEndpoint(http.MethodDelete, pattern, matchers...)
}

func (ms *MockServer) getEndpoint(method, path string) *Endpoint {
    if e, found := ms.endpoints[endpointName(method, path)]; found {
        return e
    }

    newE := newEndpoint(method, path)
    ms.endpoints[newE.Name()] = newE

    return newE
}

// Head creates a mock name for a head request.
func (ms *MockServer) Head(pattern string, matchers ...Matcher) *Scenario {
    return ms.registerEndpoint(http.MethodHead, pattern, matchers...)
}

type routingFunc func(pattern string, h http.HandlerFunc)

func (ms *MockServer) registerEndpoint(method string, pattern string, matchers ...Matcher) *Scenario {
    endpoint := ms.getEndpoint(method, pattern)
    scenario := newScenario(matchers)

    endpoint.AddScenario(scenario)

    return scenario
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
