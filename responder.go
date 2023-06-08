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
