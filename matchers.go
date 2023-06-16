package mockhttp

import (
    "io"
    "net/http"
    "net/url"
    "testing"

    "github.com/google/go-cmp/cmp"
    "github.com/stretchr/testify/assert"
)

type Matcher2 interface {
    Match(r *http.Request) bool
    Diff(r *http.Request) string
}

type queryParamMatcher struct {
    expected url.Values
}

func (q queryParamMatcher) Match(r *http.Request) bool {
    return cmp.Equal(q.expected, r.URL.Query())
}

func (q queryParamMatcher) Diff(r *http.Request) string {
    return cmp.Diff(q.expected, r.URL.Query())
}

func MatchQueryParams2(qp url.Values) Matcher2 {
    return queryParamMatcher{expected: qp}
}

type Matcher func(t *testing.T, r *http.Request)

func MatchQueryParams(qp url.Values) Matcher {
    return func(t *testing.T, r *http.Request) {
        t.Helper()
        assert.Equal(t, qp, r.URL.Query())
    }
}

func MatchHeader(headers http.Header) Matcher {
    return func(t *testing.T, r *http.Request) {
        t.Helper()
        for k, v := range headers {
            assert.Equal(t, v, r.Header[k])
        }
    }
}

func MatchJSONBody(jsonBody string) Matcher {
    return func(t *testing.T, r *http.Request) {
        t.Helper()
        body, err := io.ReadAll(r.Body)
        if err != nil {
            t.Error(err.Error())
            return
        }
        assert.JSONEq(t, jsonBody, string(body))
    }
}
