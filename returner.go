package mockhttp

import "net/http"

type Responder func(w http.ResponseWriter)

func StatusCode(code int) Responder {
	return func(w http.ResponseWriter) {
		w.WriteHeader(code)
	}
}

func Headers(headers http.Header) Responder {
	return func(w http.ResponseWriter) {
		for k, v := range headers {
			for _, i := range v {
				w.Header().Add(k, i)
			}
		}
	}
}

func JSONBody(jsonStr string) Responder {
	return func(w http.ResponseWriter) {
		w.Header().Add("Content-Type", "application/json")
		w.Write([]byte(jsonStr))
	}
}

type Returner struct {
	builders []Responder
}

func (r *Returner) Return(builders ...Responder) {
	r.builders = builders
}

func (r *Returner) write(w http.ResponseWriter) {
	mw := newMemoryResponseWriter()

	for _, b := range r.builders {
		b(mw)
	}

	mw.flush(w)
}

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

	w.WriteHeader(m.statusCode)
	w.Write(m.body)
}
