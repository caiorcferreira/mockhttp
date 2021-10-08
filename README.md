# mockhttp
[![Go Reference](https://pkg.go.dev/badge/github.com/caiorcferreira/mockhttp.svg)](https://pkg.go.dev/github.com/caiorcferreira/mockhttp)
[![Go Report Card](https://goreportcard.com/badge/github.com/caiorcferreira/mockhttp)](https://goreportcard.com/report/github.com/caiorcferreira/mockhttp)

A simple and expressive HTTP server mocking library for end-to-end tests in Go.

## Installation
```
go get github.com/caiorcferreira/mockhttp
```

## Example

The following example shows a Book API as the application under test, which calls an external ISBN database exposed with an REST interface.

Since `mockhttp` only starts the server, you must have a way to inject the address of the mocked server into your application. For example, the Book API may use an environment variable `BOOK_API_ISBN_URL` that is set to `http://localhost:12000/` during the end-to-end test execution.

```go
package e2e

import (
	"net/url"
	"strings"
	"testing"
	"github.com/caiorcferreira/mockhttp"
	"net/http"
)

func TestCreateBook(t *testing.T) {
	mockServer := mockhttp.NewMockServer(mockhttp.WithPort(12000))
	mockServer.
		Get("/isbn", mockhttp.MatchQueryParams(url.Values{"title": []string{"Foundation"}})).
		Return(mockhttp.JSONBody(`{"isbn": "9780345317988"}`))

	mockServer.Start(t)
	defer mockServer.Teardown()

	bookBody := strings.NewReader(`{"title": "Foundation"}`)
	response, err := http.Post("http://localhost:9000/book", "application/json", bookBody)
	if err != nil {
		t.Error(err.Error())
		return
	}

	if response.StatusCode != http.StatusOK {
		t.Errorf("unexpected status code: %d", response.StatusCode)
	}
	
	mockServer.AssertExpectations()
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