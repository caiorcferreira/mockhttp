package mockhttp

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

type Option func(ms *MockServer)

func WithPort(port int) Option {
	return func(ms *MockServer) {
		ms.port = port
	}
}

type MockServer struct {
	T *testing.T

	port         int
	server       *httptest.Server
	router       chi.Router
	expectations sync.Map
}

func NewMockServer(opts ...Option) *MockServer {
	router := chi.NewRouter()
	mockServer := &MockServer{router: router}
	for _, o := range opts {
		o(mockServer)
	}

	return mockServer
}

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
}

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

func (ms *MockServer) AssertExpectations() {
	var failedExpectations []string
	ms.expectations.Range(func(key, value interface{}) bool {
		expectation := key.(string)
		called := value.(bool)
		if !called {
			failedExpectations = append(failedExpectations, expectation)
		}

		return true
	})

	for _, expectation := range failedExpectations {
		ms.T.Errorf("expected endpoint was not called: %s", expectation)
	}
}

func (ms *MockServer) Get(pattern string, matchers ...Matcher) *Returner {
	expectation := expectationName(http.MethodGet, pattern)
	ms.expectations.Store(expectation, false)

	returner := &Returner{}
	ms.router.Get(pattern, ms.newHandler(expectation, returner, matchers))

	return returner
}

func (ms *MockServer) Post(pattern string, matchers ...Matcher) *Returner {
	expectation := expectationName(http.MethodPost, pattern)
	ms.expectations.Store(expectation, false)

	returner := &Returner{}
	ms.router.Post(pattern, ms.newHandler(expectation, returner, matchers))

	return returner
}

func (ms *MockServer) Put(pattern string, matchers ...Matcher) *Returner {
	expectation := expectationName(http.MethodPut, pattern)
	ms.expectations.Store(expectation, false)

	returner := &Returner{}
	ms.router.Put(pattern, ms.newHandler(expectation, returner, matchers))

	return returner
}

func (ms *MockServer) Patch(pattern string, matchers ...Matcher) *Returner {
	expectation := expectationName(http.MethodPatch, pattern)
	ms.expectations.Store(expectation, false)

	returner := &Returner{}
	ms.router.Patch(pattern, ms.newHandler(expectation, returner, matchers))

	return returner
}

func (ms *MockServer) Delete(pattern string, matchers ...Matcher) *Returner {
	expectation := expectationName(http.MethodDelete, pattern)
	ms.expectations.Store(expectation, false)

	returner := &Returner{}
	ms.router.Delete(pattern, ms.newHandler(expectation, returner, matchers))

	return returner
}

func (ms *MockServer) Head(pattern string, matchers ...Matcher) *Returner {
	expectation := expectationName(http.MethodHead, pattern)
	ms.expectations.Store(expectation, false)

	returner := &Returner{}
	ms.router.Head(pattern, ms.newHandler(expectation, returner, matchers))

	return returner
}

func (ms *MockServer) Router() chi.Router {
	return ms.router
}

func (ms *MockServer) Server() *httptest.Server {
	return ms.server
}

func (ms *MockServer) Teardown() {
	ms.server.Close()
}

func (ms *MockServer) newHandler(expectation string, returner *Returner, matchers []Matcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ms.expectations.Store(expectation, true)

		for _, m := range matchers {
			m(ms.T, r)
		}

		returner.write(w)
	}
}

func expectationName(m, p string) string {
	return m + " " + p
}
