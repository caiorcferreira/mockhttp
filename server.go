package mockhttp

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"net"
	"net/http/httptest"
	"testing"
	"time"
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

func (ms *MockServer) Router() chi.Router {
	return ms.router
}

func (ms *MockServer) Server() *httptest.Server {
	return ms.server
}

func (ms *MockServer) Teardown() {
	ms.server.Close()
}
