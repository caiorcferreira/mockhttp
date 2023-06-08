package mockhttp

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestName(t *testing.T) {
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

	t.Run("fail if not mapped route is called", func(t *testing.T) {
		mockT := new(testing.T)

		ms := NewMockServer(WithPort(60000))

		ms.Get("/get").Return(StatusCode(http.StatusNoContent))

		ms.Start(mockT)
		defer ms.Teardown()

		_, _ = http.Post("http://localhost:60000/foo", "", nil)

		require.True(t, mockT.Failed())
	})

	t.Run("mock request with return builder", func(t *testing.T) {
		ms := NewMockServer(WithPort(60000))

		ms.Get("/get").Return(StatusCode(http.StatusNoContent))

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

		ms.Post("/post").Return(
			JSONBody(`{"result": true}`),
			StatusCode(http.StatusCreated),
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
		ms.Get("/get").Return(
			StatusCode(http.StatusNoContent),
			Headers(headers),
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

	t.Run("mock request with query param matcher", func(t *testing.T) {
		ms := NewMockServer(WithPort(60000))

		ms.Get(
			"/get",
			MatchQueryParams(url.Values{"foo": []string{"bar"}}),
		).Return(
			StatusCode(http.StatusNoContent),
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
		).Return(
			StatusCode(http.StatusNoContent),
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
		).Return(
			StatusCode(http.StatusCreated),
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
}
