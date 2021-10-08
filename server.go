package mockhttp

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

type Option func(ms *MockServer)

func WithPort(port int) Option {
	return func(ms *MockServer) {
		ms.port = port
	}
}

type MockServer struct {
	port   int
	server *httptest.Server
	router chi.Router
	T      *testing.T
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

	ms.server = server

	ms.T = t

	ms.router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("no matching route found for %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	})
	ms.router.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("no matching route found for %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusMethodNotAllowed)
	})

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

func (ms *MockServer) Get(pattern string, matchers ...Matcher) *Returner {
	returner := &Returner{}
	ms.router.Get(pattern, func(w http.ResponseWriter, r *http.Request) {
		for _, m := range matchers {
			m(ms.T, r)
		}

		returner.write(w)
	})

	return returner
}

func (ms *MockServer) Post(pattern string, matchers ...Matcher) *Returner {
	returner := &Returner{}
	ms.router.Post(pattern, func(w http.ResponseWriter, r *http.Request) {
		for _, m := range matchers {
			m(ms.T, r)
		}

		returner.write(w)
	})

	return returner
}

func (ms *MockServer) Put(pattern string, f http.HandlerFunc) {
	ms.router.Put(pattern, f)
}

func (ms *MockServer) Patch(pattern string, f http.HandlerFunc) {
	ms.router.Patch(pattern, f)
}

func (ms *MockServer) Delete(pattern string, f http.HandlerFunc) {
	ms.router.Delete(pattern, f)
}

func (ms *MockServer) Head(pattern string, f http.HandlerFunc) {
	ms.router.Head(pattern, f)
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
