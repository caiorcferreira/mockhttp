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

//nolint:gocognit // test function, complexity does not apply
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

		ms.Get("/get").Respond(ResponseStatusCode(http.StatusNoContent))

		ms.Start(mockT)
		defer ms.Teardown()

		_, _ = http.Post("http://localhost:60000/foo", "", nil)

		require.True(t, mockT.Failed())
	})

	t.Run("mock multiple responses to same endpoint", func(t *testing.T) {
		ms := NewMockServer(WithPort(60000))

		ms.Delete(
			"/delete",
			MatchQueryParams(url.Values{
				"context": []string{"1"},
			}),
		).Respond(ResponseStatusCode(http.StatusForbidden))
		ms.Delete(
			"/delete",
			MatchQueryParams(url.Values{
				"context": []string{"2"},
			}),
		).Respond(ResponseStatusCode(http.StatusOK))

		ms.Start(t)
		defer ms.Teardown()

		req1, err := http.NewRequest(http.MethodDelete, ms.URL()+"/delete?context=1", http.NoBody)
		require.NoError(t, err)

		first, err := http.DefaultClient.Do(req1)
		require.NoError(t, err)

		require.Equal(t, http.StatusForbidden, first.StatusCode)

		req2, err := http.NewRequest(http.MethodDelete, ms.URL()+"/delete?context=2", http.NoBody)
		require.NoError(t, err)

		second, err := http.DefaultClient.Do(req2)
		require.NoError(t, err)

		require.Equal(t, http.StatusOK, second.StatusCode)
	})

	t.Run("mock repeatable responses to same endpoint", func(t *testing.T) {
		ms := NewMockServer(WithPort(60000))

		ms.Delete(
			"/delete",
			MatchQueryParams(url.Values{
				"context": []string{"1"},
			}),
		).Times(2).Respond(ResponseStatusCode(http.StatusForbidden))
		ms.Delete(
			"/delete",
			MatchQueryParams(url.Values{
				"context": []string{"2"},
			}),
		).Times(3).Respond(ResponseStatusCode(http.StatusOK))

		ms.Start(t)
		defer ms.Teardown()

		for i := 0; i < 2; i++ {
			req1, err := http.NewRequest(http.MethodDelete, ms.URL()+"/delete?context=1", http.NoBody)
			require.NoError(t, err)

			first, err := http.DefaultClient.Do(req1)
			require.NoError(t, err)

			require.Equalf(t, http.StatusForbidden, first.StatusCode, "request %d was wrong", i)
		}

		for i := 0; i < 3; i++ {
			req2, err := http.NewRequest(http.MethodDelete, ms.URL()+"/delete?context=2", http.NoBody)
			require.NoError(t, err)

			second, err := http.DefaultClient.Do(req2)
			require.NoError(t, err)

			require.Equalf(t, http.StatusOK, second.StatusCode, "request %d was wrong", i)
		}
	})

	t.Run("mock request with return builder", func(t *testing.T) {
		ms := NewMockServer(WithPort(60000))

		ms.Get("/get").Respond(ResponseStatusCode(http.StatusNoContent))

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

		ms.Post("/post").Respond(
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
		ms.Get("/get").Respond(
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

		ms.Get("/get").Respond(
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

		ms.Post("/post").Respond(
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
		).Respond(
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
		).Respond(
			ResponseStatusCode(http.StatusNoContent),
		)

		ms.Start(t)
		defer ms.Teardown()

		request, err := http.NewRequest(http.MethodGet, "http://localhost:60000/get", nil)
		require.NoError(t, err)

		request.Header.Set("X-App", "foo")

		var response *http.Response
		require.Eventually(t, func() bool {
			r, doErr := http.DefaultClient.Do(request)
			if doErr != nil {
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
		).Respond(
			ResponseStatusCode(http.StatusCreated),
		)

		ms.Start(t)
		defer ms.Teardown()

		bodyReader := strings.NewReader(jsonBody)
		request, err := http.NewRequest(http.MethodPost, "http://localhost:60000/post", bodyReader)
		require.NoError(t, err)

		var response *http.Response
		require.Eventually(t, func() bool {
			r, doErr := http.DefaultClient.Do(request)
			if doErr != nil {
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

	t.Run("fails assertion when not all mocked endpoints where called", func(t *testing.T) {
		mockT := new(testing.T)

		ms := NewMockServer(WithPort(60000))

		ms.Get("/get").Respond(ResponseStatusCode(http.StatusNoContent))
		ms.Get("/post").Respond(ResponseStatusCode(http.StatusNoContent))

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

	t.Run("get number of times mocked endpoint was called", func(t *testing.T) {
		mockT := new(testing.T)

		ms := NewMockServer(WithPort(60000))

		scenario := ms.Get("/get").Respond(ResponseStatusCode(http.StatusNoContent))

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

		require.Equal(t, 1, scenario.TimesCalled())
	})

	t.Run("verifies number of times mocked name was called", func(t *testing.T) {
		mockT := new(testing.T)

		ms := NewMockServer(WithPort(60000))

		ms.Get("/get").Times(1).Respond(ResponseStatusCode(http.StatusNoContent))

		ms.Start(mockT)
		defer ms.Teardown()

		getURL := ms.URL() + "/get"

		_, err := http.Get(getURL)
		require.NoError(t, err)

		r, err := http.Get(getURL)
		require.NoError(t, err)

		require.Equal(t, http.StatusNoContent, r.StatusCode)

		ms.AssertExpectations()
		require.True(t, mockT.Failed())
	})
}

// This uses the built-in cleanup to perform
// a integration test similar to what the lib user
// should write.
func TestMockServer_Cleanup(t *testing.T) {
	ms := NewMockServer()

	ms.Get("/get").Respond(ResponseStatusCode(http.StatusNoContent))

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
