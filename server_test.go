package mockhttp

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMockServer(t *testing.T) {
	t.Run("start mock server at specify port", func(t *testing.T) {
		ms := NewMockServer(WithPort(60000))
		ms.Start(t)
		defer ms.Teardown()

		addr := "localhost:60000"
		require.Eventually(t, func() bool {
			_, err := net.Dial("tcp", addr)
			return err == nil
		}, 2*time.Second, 200*time.Millisecond)
	})

	t.Run("get mock server URL at specify port", func(t *testing.T) {
		ms := NewMockServer(WithPort(60000))
		ms.Start(t)
		defer ms.Teardown()

		addr := "localhost:60000"
		require.Eventually(t, func() bool {
			_, err := net.Dial("tcp", addr)
			return err == nil
		}, 2*time.Second, 200*time.Millisecond)

		require.Equal(t, "http://127.0.0.1:60000", ms.URL())
	})

	t.Run("start mock server at any port available", func(t *testing.T) {
		ms := NewMockServer()
		ms.Start(t)
		defer ms.Teardown()

		addr := fmt.Sprintf("localhost:%d", ms.Port())
		require.Eventually(t, func() bool {
			_, err := net.Dial("tcp", addr)
			return err == nil
		}, 2*time.Second, 200*time.Millisecond)
	})

	t.Run("fail if unmapped route is called", func(t *testing.T) {
		mockT := new(testing.T)

		ms := NewMockServer(WithPort(60000))

		ms.Get("/get").Response(ResponseStatusCode(http.StatusNoContent))

		ms.Start(mockT)
		defer ms.Teardown()

		_, _ = http.Post("http://localhost:60000/foo", "", nil)

		require.True(t, mockT.Failed())
	})

	t.Run("mock request with return builder", func(t *testing.T) {
		ms := NewMockServer(WithPort(60000))

		ms.Get("/get").Response(ResponseStatusCode(http.StatusNoContent))

		ms.Start(t)
		defer ms.Teardown()

		var response *http.Response
		require.Eventually(t, func() bool {
			r, err := http.Get("http://localhost:60000/get")
			if err != nil {
				return false
			}
			response = r
			return true
		}, 2*time.Second, 200*time.Millisecond)

		require.Equal(t, http.StatusNoContent, response.StatusCode)
	})

	t.Run("mock request with body and status return builder in wrong order", func(t *testing.T) {
		ms := NewMockServer(WithPort(60000))

		ms.Post("/post").Response(
			JSONResponseBody(`{"result": true}`),
			ResponseStatusCode(http.StatusCreated),
		)

		ms.Start(t)
		defer ms.Teardown()

		var response *http.Response
		require.Eventually(t, func() bool {
			r, err := http.Post("http://localhost:60000/post", "text/html", nil)
			if err != nil {
				return false
			}
			response = r
			return true
		}, 2*time.Second, 200*time.Millisecond)

		require.Equal(t, http.StatusCreated, response.StatusCode)

		body, err := io.ReadAll(response.Body)
		require.NoError(t, err)

		require.JSONEq(t, `{"result": true}`, string(body))
	})

	t.Run("mock request with header return builder", func(t *testing.T) {
		ms := NewMockServer(WithPort(60000))

		headers := http.Header{
			"X-Foo": []string{"bar"},
		}
		ms.Get("/get").Response(
			ResponseStatusCode(http.StatusNoContent),
			ResponseHeaders(headers),
		)

		ms.Start(t)
		defer ms.Teardown()

		var response *http.Response
		require.Eventually(t, func() bool {
			r, err := http.Get("http://localhost:60000/get")
			if err != nil {
				return false
			}
			response = r
			return true
		}, 2*time.Second, 200*time.Millisecond)

		require.Equal(t, http.StatusNoContent, response.StatusCode)
		for k, v := range headers {
			require.Equal(t, v, response.Header[k])
		}
	})

	t.Run("mock request with json file return builder", func(t *testing.T) {
		ms := NewMockServer(WithPort(60000))

		ms.Get("/get").Response(
			ResponseStatusCode(http.StatusOK),
			JSONFileResponseBody(t, "./fixtures/body.json"),
		)

		ms.Start(t)
		defer ms.Teardown()

		var response *http.Response
		require.Eventually(t, func() bool {
			r, err := http.Get("http://localhost:60000/get")
			if err != nil {
				return false
			}
			response = r
			return true
		}, 2*time.Second, 200*time.Millisecond)

		require.Equal(t, http.StatusOK, response.StatusCode)

		body, err := io.ReadAll(response.Body)
		require.NoError(t, err)

		require.JSONEq(t, `{"result": true}`, string(body))
	})

	t.Run("mock request with string response body", func(t *testing.T) {
		ms := NewMockServer(WithPort(60000))

		ms.Post("/post").Response(
			StringResponseBody(`success`),
		)

		ms.Start(t)
		defer ms.Teardown()

		var response *http.Response
		require.Eventually(t, func() bool {
			r, err := http.Post("http://localhost:60000/post", "text/html", nil)
			if err != nil {
				return false
			}
			response = r
			return true
		}, 2*time.Second, 200*time.Millisecond)

		require.Equal(t, http.StatusOK, response.StatusCode)

		body, err := io.ReadAll(response.Body)
		require.NoError(t, err)

		require.Equal(t, `success`, string(body))
	})

	t.Run("mock request with query param matcher", func(t *testing.T) {
		ms := NewMockServer(WithPort(60000))

		ms.Get(
			"/get",
			MatchQueryParams(url.Values{"foo": []string{"bar"}}),
		).Response(
			ResponseStatusCode(http.StatusNoContent),
		)

		ms.Start(t)
		defer ms.Teardown()

		var response *http.Response
		require.Eventually(t, func() bool {
			r, err := http.Get("http://localhost:60000/get?foo=bar")
			if err != nil {
				return false
			}
			response = r
			return true
		}, 2*time.Second, 200*time.Millisecond)

		require.Equal(t, http.StatusNoContent, response.StatusCode)
	})

	t.Run("mock request with header matcher", func(t *testing.T) {
		ms := NewMockServer(WithPort(60000))

		ms.Get(
			"/get",
			MatchHeader(http.Header{"X-App": []string{"foo"}}),
		).Response(
			ResponseStatusCode(http.StatusNoContent),
		)

		ms.Start(t)
		defer ms.Teardown()

		request, err := http.NewRequest(http.MethodGet, "http://localhost:60000/get", nil)
		require.NoError(t, err)

		request.Header.Set("X-App", "foo")

		var response *http.Response
		require.Eventually(t, func() bool {
			r, err := http.DefaultClient.Do(request)
			if err != nil {
				return false
			}
			response = r
			return true
		}, 2*time.Second, 200*time.Millisecond)

		require.Equal(t, http.StatusNoContent, response.StatusCode)
	})

	t.Run("mock request with json body matcher", func(t *testing.T) {
		ms := NewMockServer(WithPort(60000))

		jsonBody := `{"result": true}`
		ms.Post(
			"/post",
			MatchJSONBody(jsonBody),
		).Response(
			ResponseStatusCode(http.StatusCreated),
		)

		ms.Start(t)
		defer ms.Teardown()

		bodyReader := strings.NewReader(jsonBody)
		request, err := http.NewRequest(http.MethodPost, "http://localhost:60000/post", bodyReader)
		require.NoError(t, err)

		var response *http.Response
		require.Eventually(t, func() bool {
			r, err := http.DefaultClient.Do(request)
			if err != nil {
				return false
			}
			response = r
			return true
		}, 2*time.Second, 200*time.Millisecond)

		require.Equal(t, http.StatusCreated, response.StatusCode)
	})

	t.Run("mock request with no return builder", func(t *testing.T) {
		ms := NewMockServer(WithPort(60000))

		ms.Get("/get")

		ms.Start(t)
		defer ms.Teardown()

		var response *http.Response
		require.Eventually(t, func() bool {
			r, err := http.Get("http://localhost:60000/get")
			if err != nil {
				return false
			}
			response = r
			return true
		}, 2*time.Second, 200*time.Millisecond)

		require.Equal(t, http.StatusOK, response.StatusCode)

		body, err := io.ReadAll(response.Body)
		require.NoError(t, err)

		require.Empty(t, body)
	})

	t.Run("verifies that all mocked endpoint where called", func(t *testing.T) {
		mockT := new(testing.T)

		ms := NewMockServer(WithPort(60000))

		ms.Get("/get").Response(ResponseStatusCode(http.StatusNoContent))
		ms.Post("/post").Response(ResponseStatusCode(http.StatusOK))

		ms.Start(mockT)
		defer ms.Teardown()

		var response *http.Response
		require.Eventually(t, func() bool {
			r, err := http.Get("http://localhost:60000/get")
			if err != nil {
				return false
			}
			response = r
			return true
		}, 2*time.Second, 200*time.Millisecond)

		require.Equal(t, http.StatusNoContent, response.StatusCode)

		ms.AssertExpectations()
		require.True(t, mockT.Failed())
	})

	t.Run("verifies that an endpoint was not called", func(t *testing.T) {
		mockT := new(testing.T)

		ms := NewMockServer(WithPort(60000))

		getEndpoint := ms.Get("/get").Response(ResponseStatusCode(http.StatusNoContent)).Endpoint()
		postEndpoint := ms.Post("/post").Response(ResponseStatusCode(http.StatusOK)).Endpoint()

		ms.Start(mockT)
		defer ms.Teardown()

		var response *http.Response
		require.Eventually(t, func() bool {
			r, err := http.Get("http://localhost:60000/get")
			if err != nil {
				return false
			}
			response = r
			return true
		}, 2*time.Second, 200*time.Millisecond)

		require.Equal(t, http.StatusNoContent, response.StatusCode)

		ms.AssertNotCalled(postEndpoint)
		require.False(t, mockT.Failed())

		ms.AssertNotCalled(getEndpoint)
		require.True(t, mockT.Failed())
	})

	t.Run("get number of times mocked endpoint was called", func(t *testing.T) {
		mockT := new(testing.T)

		ms := NewMockServer(WithPort(60000))

		endpoint := ms.Get("/get").Response(ResponseStatusCode(http.StatusNoContent)).Endpoint()

		ms.Start(mockT)
		defer ms.Teardown()

		var response *http.Response
		require.Eventually(t, func() bool {
			r, err := http.Get("http://localhost:60000/get")
			if err != nil {
				return false
			}
			response = r
			return true
		}, 2*time.Second, 200*time.Millisecond)

		require.Equal(t, http.StatusNoContent, response.StatusCode)

		require.Equal(t, 1, ms.TimesCalled(endpoint))
	})

	t.Run("verfies number of times mocked endpoint was called", func(t *testing.T) {
		mockT := new(testing.T)

		ms := NewMockServer(WithPort(60000))

		endpoint := ms.Get("/get").Response(ResponseStatusCode(http.StatusNoContent)).Endpoint()

		ms.Start(mockT)
		defer ms.Teardown()

		var response *http.Response
		require.Eventually(t, func() bool {
			r, err := http.Get("http://localhost:60000/get")
			if err != nil {
				return false
			}
			response = r
			return true
		}, 2*time.Second, 200*time.Millisecond)

		require.Equal(t, http.StatusNoContent, response.StatusCode)

		ms.AssertTimesCalled(endpoint, 2)
		require.True(t, mockT.Failed())
	})
}

// This uses the built-in cleanup to perform
// a integration test similar to what the lib user
// should write.
func TestMockServer_Cleanup(t *testing.T) {
	ms := NewMockServer()

	ms.Get("/get").Response(ResponseStatusCode(http.StatusNoContent))

	ms.Start(t)

	var response *http.Response
	require.Eventually(t, func() bool {
		r, err := http.Get(ms.URL() + "/get")
		if err != nil {
			return false
		}
		response = r
		return true
	}, 2*time.Second, 200*time.Millisecond)

	require.Equal(t, http.StatusNoContent, response.StatusCode)
}
