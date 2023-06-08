package mockhttp

import "net/http"

// Endpoint represents an HTTP method + path with a mocked response
type Endpoint struct {
	name     string
	times    int
	builders []Responder
}

func newEndpoint(method, path string) *Endpoint {
	return &Endpoint{
		name:  endpointName(method, path),
		times: 1,
	}
}

// Times sets the how many requests it is expected to be received by this endpoint.
func (e *Endpoint) Times(n int) *Endpoint {
	e.times = n
	return e
}

// Respond set up a collection of Responders.
func (e *Endpoint) Respond(builders ...Responder) *Endpoint {
	e.builders = builders
	return e
}

// Name returns the endpoint name (method + path) that this Returner represents.
func (e *Endpoint) Name() string {
	return e.name
}

func (e *Endpoint) writeTo(w http.ResponseWriter) {
	mw := newMemoryResponseWriter()

	for _, b := range e.builders {
		b(mw)
	}

	mw.flush(w)
}

// memoryResponseWriter accumulates all response builders
// mutations such that the order they are used in test does not matter.
//
// This is necessary because if ResponseStatusCode is used after JSONResponseBody, the
// status will be fixed at 200 by the Write call to http.ResponseWriter.
type memoryResponseWriter struct {
	headers    http.Header
	body       []byte
	statusCode int
}

func newMemoryResponseWriter() *memoryResponseWriter {
	return &memoryResponseWriter{headers: make(http.Header)}
}

func (m *memoryResponseWriter) Header() http.Header {
	return m.headers
}

func (m *memoryResponseWriter) Write(bytes []byte) (int, error) {
	m.body = bytes
	return len(bytes), nil
}

func (m *memoryResponseWriter) WriteHeader(statusCode int) {
	m.statusCode = statusCode
}

func (m *memoryResponseWriter) flush(w http.ResponseWriter) {
	for k, values := range m.headers {
		for _, v := range values {
			w.Header().Add(k, v)
		}
	}

	if m.statusCode > 0 {
		w.WriteHeader(m.statusCode)
	}

	if len(m.body) > 0 {
		w.Write(m.body)
	}
}
