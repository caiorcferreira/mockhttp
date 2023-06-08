package mockhttp

import (
	"net/http"
	"os"
	"testing"
)

// Responder configures a http.ResponseWriter to send data back.
type Responder func(w http.ResponseWriter)

// ResponseStatusCode is a Responder that defines the response status code.
func ResponseStatusCode(code int) Responder {
	return func(w http.ResponseWriter) {
		w.WriteHeader(code)
	}
}

// ResponseHeaders is a Responder that defines the response headers.
func ResponseHeaders(headers http.Header) Responder {
	return func(w http.ResponseWriter) {
		for k, v := range headers {
			for _, i := range v {
				w.Header().Add(k, i)
			}
		}
	}
}

// JSONResponseBody is a Responder that defines the response body as a JSON string.
func JSONResponseBody(jsonStr string) Responder {
	return func(w http.ResponseWriter) {
		w.Header().Add("Content-Type", "application/json")
		w.Write([]byte(jsonStr))
	}
}

// JSONFileResponseBody is a Responder that defines the response body as a JSON file.
func JSONFileResponseBody(t *testing.T, filePath string) Responder {
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read json file: %s", err.Error())
		return noop
	}

	return func(w http.ResponseWriter) {
		w.Header().Add("Content-Type", "application/json")
		w.Write(content)
	}
}

func StringResponseBody(b string) Responder {
	return func(w http.ResponseWriter) {
		w.Write([]byte(b))
	}
}

func noop(w http.ResponseWriter) {}

// Returner constructs the endpoint's response with a collection of Responders.
type Returner struct {
	endpoint string
	builders []Responder
}

func newReturner(endpoint string) *Returner {
	return &Returner{endpoint: endpoint}
}

// Response set up a collection of Responders.
func (r *Returner) Response(builders ...Responder) *Returner {
	r.builders = builders
	return r
}

// Endpoint returns the endpoint name (method + path) that this Returner represents.
func (r *Returner) Endpoint() string {
	return r.endpoint
}

func (r *Returner) write(w http.ResponseWriter) {
	mw := newMemoryResponseWriter()

	for _, b := range r.builders {
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
