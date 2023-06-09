# mockhttp
[![Go Reference](https://pkg.go.dev/badge/github.com/caiorcferreira/mockhttp.svg)](https://pkg.go.dev/github.com/caiorcferreira/mockhttp)
[![Go Report Card](https://goreportcard.com/badge/github.com/caiorcferreira/mockhttp)](https://goreportcard.com/report/github.com/caiorcferreira/mockhttp)
[![Builds](https://github.com/caiorcferreira/mockhttp/actions/workflows/main.yaml/badge.svg)](https://github.com/caiorcferreira/mockhttp/actions/workflows/main.yaml)

A simple and expressive HTTP server mocking library for end-to-end and integration tests in Go.

## Installation
```
go get github.com/caiorcferreira/mockhttp
```

## Usage

`mockhttp` works by starting a local HTTP server that will handle your requests based on your configuration.

When the test ends it automatically verifies that every endpoint created using the method-based API was called with the correct inputs and the expected number of times.

### Examples

#### JSON Response
```go
func TestExample(t *testing.T) {
	bookInfo := `{"isbn": "9780345317988"}`

	mockServer := mockhttp.NewMockServer()
	mockServer.
		Get("/isbn").
		Respond(mockhttp.JSONResponseBody(bookInfo))

	mockServer.Start(t)

	response, err := http.Get(mockServer.URL() + "/book")
	if err != nil {
		t.Fatal(err.Error())
		return
	}

	if response.StatusCode != http.StatusOK {
		t.Errorf("unexpected status code: %d", response.StatusCode)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err.Error())
		return
	}

	if bookInfo != string(body) {
		t.Errorf("mismatched body, got: %s, expected: %s", string(body), bookInfo)
	}
}
```

#### Define server port
```go
func TestExample(t *testing.T) {
	bookInfo := `{"isbn": "9780345317988"}`

	mockServer := mockhttp.NewMockServer(mockhttp.WithPort(60000))
	mockServer.
		Get("/isbn").
		Respond(mockhttp.JSONResponseBody(bookInfo))

	mockServer.Start(t)

	response, err := http.Get(mockServer.URL() + "/book")
	if err != nil {
		t.Fatal(err.Error())
		return
	}

	if response.StatusCode != http.StatusOK {
		t.Errorf("unexpected status code: %d", response.StatusCode)
	}
}
```

#### Multiple HTTP methods
```go
func TestExample(t *testing.T) {
	bookInfo := `{"isbn": "9780345317988"}`

	mockServer := mockhttp.NewMockServer()
	mockServer.
		Get("/isbn").
		Respond(mockhttp.JSONResponseBody(bookInfo))
	mockServer.
		Post("/isbn").
		Respond(mockhttp.ResponseStatusCode(http.StatusMethodNotAllowed))
	mockServer.
		Patch("/isbn").
		Respond(mockhttp.ResponseStatusCode(http.StatusMethodNotAllowed))
	mockServer.
		Put("/isbn").
		Respond(mockhttp.ResponseStatusCode(http.StatusMethodNotAllowed))
	mockServer.
		Delete("/isbn").
		Respond(mockhttp.ResponseStatusCode(http.StatusMethodNotAllowed))
	mockServer.
		Head("/isbn").
		Respond(mockhttp.ResponseStatusCode(http.StatusMethodNotAllowed))

	mockServer.Start(t)

	response, err := http.Get(mockServer.URL() + "/book")
	if err != nil {
		t.Fatal(err.Error())
		return
	}

	if response.StatusCode != http.StatusOK {
		t.Errorf("unexpected status code: %d", response.StatusCode)
	}
	
	notAllowed := []string{
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodHead,
	}

	for _, method := range notAllowed {
		req, err := http.NewRequest(method, mockServer.URL()+"/isbn", http.NoBody)
		if err != nil {
			t.Fatal(err.Error())
			return
		}

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err.Error())
			return
		}

		if res.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("unexpected status code: %d", response.StatusCode)
		}
	}
}
```

#### Query Parameter matching
```go
func TestExample(t *testing.T) {
	book := "Foundation"
	expectedQueryParams := url.Values{"title": []string{book}}
	
	mockServer := mockhttp.NewMockServer()
	mockServer.
		Get("/isbn", mockhttp.MatchQueryParams(expectedQueryParams)).
		Respond(mockhttp.JSONResponseBody(`{"isbn": "9780345317988"}`))

	mockServer.Start(t)

	bookBody := strings.NewReader(`{"title": "Foundation"}`)
	targetURL := fmt.Sprintf("%s/book?title=%s", mockServer.URL(), book)
	
	response, err := http.Get(targetURL)
	if err != nil {
		t.Error(err.Error())
		return
	}

	if response.StatusCode != http.StatusOK {
		t.Errorf("unexpected status code: %d", response.StatusCode)
	}
}
```

#### Header matching
```go
func TestExample(t *testing.T) {
	token := "test-token"

	mockServer := mockhttp.NewMockServer()
	mockServer.
		Get("/isbn", mockhttp.MatchHeader(
			http.Header{"Authorization": []string{token}},
		)).
		Respond(mockhttp.ResponseStatusCode(http.StatusOK))

	mockServer.Start(t)

	req, err := http.NewRequest(http.MethodGet, mockServer.URL()+"/book", http.NoBody)
	if err != nil {
		t.Fatal(err.Error())
		return
	}

	req.Header.Add("Authorization", token)

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err.Error())
		return
	}

	if response.StatusCode != http.StatusOK {
		t.Errorf("unexpected status code: %d", response.StatusCode)
	}
}
```

#### Header response
```go
func TestExample(t *testing.T) {
	bookInfo := `{"isbn": "9780345317988"}`
	maxAge := "max-age=604800"

	mockServer := mockhttp.NewMockServer()
	mockServer.
		Get("/isbn").
		Respond(
			mockhttp.JSONResponseBody(bookInfo),
			mockhttp.ResponseHeaders(http.Header{
				"Cache-Control": []string{maxAge},
			}),
		)

	mockServer.Start(t)

	response, err := http.Get(mockServer.URL() + "/book")
	if err != nil {
		t.Fatal(err.Error())
		return
	}

	if response.StatusCode != http.StatusOK {
		t.Errorf("unexpected status code: %d", response.StatusCode)
	}

	cacheHeader := response.Header.Get("Cache-Control")
	if cacheHeader != maxAge {
		t.Errorf("unexpected cache control header: got %s, expected %s", cacheHeader, maxAge)
	}
}
```

#### Multiple calls matching
```go
func TestExample(t *testing.T) {
	bookInfo := `{"isbn": "9780345317988"}`

	mockServer := mockhttp.NewMockServer()
	mockServer.
		Get("/isbn").
		Times(3).
		Respond(mockhttp.JSONResponseBody(bookInfo))

	mockServer.Start(t)

	// all response should be the same and mockhttp validates the endpoint
	// was called the expected number of times.
	for i := 0; i < 3; i++ {
		response, err := http.Get(mockServer.URL() + "/book")
		if err != nil {
			t.Fatal(err.Error())
			return
		}

		if response.StatusCode != http.StatusOK {
			t.Errorf("unexpected status code: %d", response.StatusCode)
		}
	}
}
```

#### Custom mock handler

This example uses the internal `chi.Router` to add an endpoint handler that produces dynamic responses each time its called.

Only use this method when you need something the lib does not support, since **it will not assert** any input nor if it was called. This is manual mode.

```go
func TestExample(t *testing.T) {
	var called int32
	
	mockServer := mockhttp.NewMockServer()
	mockServer.Router().Get("/isbn", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&called)
		
		code := strconv.Itoa(rand.Int())
		w.Write([]byte(code))
	})

	mockServer.Start(t)

	response, err := http.Get(mockServer.URL() + "/book")
	if err != nil {
		t.Fatal(err.Error())
		return
	}

	if response.StatusCode != http.StatusOK {
		t.Errorf("unexpected status code: %d", response.StatusCode)
	}

    callCount := atomic.LoadInt32(&called)
	if callCount != 1 {
        t.Errorf("endpoint called wrong number of time: got %d, expected %d", calledCount, 1)
    }
}
```

## Contributing
Every help is always welcome. Feel free do throw us a pull request, we'll do our best to check it out as soon as possible. But before that, let us establish some guidelines:

1. This is an open source project so please do not add any proprietary code or infringe any copyright of any sort.
2. Avoid unnecessary dependencies or messing up go.mod file.
3. Be aware of golang coding style. Use a lint to help you out.
4.  Add tests to cover your contribution.
5. Use meaningful [messages](https://medium.com/@menuka/writing-meaningful-git-commit-messages-a62756b65c81) to your commits.
6. Use [pull requests](https://help.github.com/en/github/collaborating-with-issues-and-pull-requests/about-pull-requests).
7. At last, but also important, be kind and polite with the community.

Any submitted issue which disrespect one or more guidelines above, will be discarded and closed.


## License

Released under the [MIT License](LICENSE).