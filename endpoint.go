package mockhttp

import (
    "net/http"
    "sync/atomic"
    "testing"
)

// Scenario is a mock case for a specific endpoint
type Scenario struct {
    executionCount int64
    times          int
    builders       []Responder
    matchers       []Matcher
}

func newScenario(matchers []Matcher) *Scenario {
    return &Scenario{
        matchers: matchers,
        times:    1,
    }
}

// Match verifies if request matches expectations
func (s *Scenario) Match(t *testing.T, r *http.Request) {
    t.Helper()

    atomic.AddInt64(&s.executionCount, 1)

    for _, m := range s.matchers {
        m(t, r)
    }
}

// Times sets the how many requests it is expected to be received by this endpoint.
func (s *Scenario) Times(n int) *Scenario {
    s.times = n
    return s
}

// TimesCalled return how many times this Scenario was executed
func (s *Scenario) TimesCalled() int {
    return int(atomic.LoadInt64(&s.executionCount))
}

// Respond set up a collection of Responders.
func (s *Scenario) Respond(builders ...Responder) *Scenario {
    s.builders = builders
    return s
}

func (s *Scenario) respondTo(w http.ResponseWriter) {
    mw := newMemoryResponseWriter()

    for _, b := range s.builders {
        b(mw)
    }

    mw.flush(w)
}

// Endpoint defines an HTTP method and path that have
// multiple mocked scenarios to produce responses.
type Endpoint struct {
    method string
    path   string

    requestCount int64
    scenarios    []*Scenario
}

func newEndpoint(method, path string) *Endpoint {
    return &Endpoint{method: method, path: path}
}

// Handler create an HTTP handler that executes each scenario in the order
// they were defined. If a scenario defines a Times expectation, the scenario
// is executed the number of times it's defined.
func (e *Endpoint) Handler(t *testing.T) http.HandlerFunc {
    t.Helper()

    var responsePlan []int
    for index, s := range e.scenarios {
        for i := 0; i < s.times; i++ {
            responsePlan = append(responsePlan, index)
        }
    }

    return func(w http.ResponseWriter, r *http.Request) {
        plan := atomic.LoadInt64(&e.requestCount)
        if plan >= int64(len(responsePlan)) {
            // if endpoint called more times than planned
            // just use the last scenario for response
            plan = int64(len(responsePlan) - 1)
        }

        currentScenarioIndex := responsePlan[plan]
        scenario := e.scenarios[currentScenarioIndex]

        scenario.Match(t, r)
        scenario.respondTo(w)

        atomic.AddInt64(&e.requestCount, 1)
    }
}

// Name returns the endpoint name (method + path) that this Returner represents.
func (e *Endpoint) Name() string {
    return endpointName(e.method, e.path)
}

// AddScenario appends a scenario to the endpoint
func (e *Endpoint) AddScenario(s *Scenario) {
    e.scenarios = append(e.scenarios, s)
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

func endpointName(m, p string) string {
    return m + " " + p
}
