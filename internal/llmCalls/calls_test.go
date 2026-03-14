package llmcalls

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nikaydo/personal-assistant/internal/config"
	"github.com/nikaydo/personal-assistant/internal/models"
)

type countingRoundTripper struct {
	mu    sync.Mutex
	calls int
}

func (rt *countingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.mu.Lock()
	rt.calls++
	rt.mu.Unlock()

	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
		Header:     make(http.Header),
		Request:    req,
	}, nil
}

func (rt *countingRoundTripper) Calls() int {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return rt.calls
}

type timeoutRoundTripper struct{}

func (timeoutRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	<-req.Context().Done()
	return nil, req.Context().Err()
}

func TestDoReq_UsesSharedHTTPClient(t *testing.T) {
	oldClient := httpClient
	rt := &countingRoundTripper{}
	httpClient = &http.Client{
		Timeout:   time.Second,
		Transport: rt,
	}
	t.Cleanup(func() {
		httpClient = oldClient
	})

	if _, err := doReq([]byte(`{}`), "http://example.com", "token", http.MethodPost); err != nil {
		t.Fatalf("first doReq failed: %v", err)
	}
	if _, err := doReq([]byte(`{}`), "http://example.com", "token", http.MethodPost); err != nil {
		t.Fatalf("second doReq failed: %v", err)
	}

	if rt.Calls() != 2 {
		t.Fatalf("unexpected call count: got=%d want=2", rt.Calls())
	}
}

func TestDoReq_RespectsClientTimeout(t *testing.T) {
	oldClient := httpClient
	httpClient = &http.Client{
		Timeout:   20 * time.Millisecond,
		Transport: timeoutRoundTripper{},
	}
	t.Cleanup(func() {
		httpClient = oldClient
	})

	if _, err := doReq([]byte(`{}`), "http://example.com", "token", http.MethodPost); err == nil {
		t.Fatalf("expected timeout error")
	}
}

type sequenceRoundTripper struct {
	mu      sync.Mutex
	status  []int
	calls   int
	payload string
}

func (rt *sequenceRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.calls++
	code := http.StatusOK
	if len(rt.status) > 0 {
		idx := rt.calls - 1
		if idx >= len(rt.status) {
			idx = len(rt.status) - 1
		}
		code = rt.status[idx]
	}
	body := rt.payload
	if body == "" {
		body = `{"ok":true}`
	}
	if code >= 400 {
		body = fmt.Sprintf(`{"error":"status %d"}`, code)
	}
	return &http.Response{
		StatusCode: code,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
		Request:    req,
	}, nil
}

func (rt *sequenceRoundTripper) Calls() int {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return rt.calls
}

func TestDoReqWithRetry_RetriesRetryableStatus(t *testing.T) {
	oldClient := httpClient
	rt := &sequenceRoundTripper{status: []int{http.StatusTooManyRequests, http.StatusOK}}
	httpClient = &http.Client{Timeout: time.Second, Transport: rt}
	t.Cleanup(func() { httpClient = oldClient })

	cfg := config.Config{
		LLMRetryMaxAttempts: 2,
		LLMRetryBaseDelayMs: 1,
		LLMRetryMaxDelayMs:  1,
	}
	if _, err := doReqWithRetry([]byte(`{}`), "http://example.com", "token", http.MethodPost, cfg); err != nil {
		t.Fatalf("doReqWithRetry failed: %v", err)
	}
	if rt.Calls() != 2 {
		t.Fatalf("unexpected retries count: got=%d want=2", rt.Calls())
	}
}

func TestDoReqWithRetry_DoesNotRetryNonRetryableStatus(t *testing.T) {
	oldClient := httpClient
	rt := &sequenceRoundTripper{status: []int{http.StatusBadRequest, http.StatusOK}}
	httpClient = &http.Client{Timeout: time.Second, Transport: rt}
	t.Cleanup(func() { httpClient = oldClient })

	cfg := config.Config{
		LLMRetryMaxAttempts: 3,
		LLMRetryBaseDelayMs: 1,
		LLMRetryMaxDelayMs:  1,
	}
	if _, err := doReqWithRetry([]byte(`{}`), "http://example.com", "token", http.MethodPost, cfg); err == nil {
		t.Fatalf("expected error")
	}
	if rt.Calls() != 1 {
		t.Fatalf("non-retryable status should not be retried: got=%d want=1", rt.Calls())
	}
}

type fallbackWebSearchOptionsRoundTripper struct {
	mu     sync.Mutex
	calls  int
	bodies []string
}

func (rt *fallbackWebSearchOptionsRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	payload, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	rt.mu.Lock()
	rt.calls++
	rt.bodies = append(rt.bodies, string(payload))
	rt.mu.Unlock()

	if strings.Contains(string(payload), `"web_search_options"`) {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"Provider returned error","code":400,"metadata":{"raw":"{\"error\":{\"message\":\"Web search options not supported with this model.\",\"type\":\"invalid_request_error\",\"param\":\"web_search_options\"}}","provider_name":"Azure","is_byok":false}}}`)),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	}

	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"choices":[{"message":{"content":"ok"}}]}`)),
		Header:     make(http.Header),
		Request:    req,
	}, nil
}

func (rt *fallbackWebSearchOptionsRoundTripper) snapshot() (int, []string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	out := make([]string, len(rt.bodies))
	copy(out, rt.bodies)
	return rt.calls, out
}

func TestAsk_RetriesWithoutWebSearchOptionsWhenProviderRejectsThem(t *testing.T) {
	oldClient := httpClient
	rt := &fallbackWebSearchOptionsRoundTripper{}
	httpClient = &http.Client{Timeout: time.Second, Transport: rt}
	t.Cleanup(func() { httpClient = oldClient })

	body := models.RequestBody{
		Model:   "openai/gpt-4o-mini",
		Plugins: []models.Plugin{{ID: "web"}},
		WebSearchOptions: &models.WebSearchOptions{
			SearchContextSize: "medium",
		},
	}

	resp, err := Ask(body, config.Config{})
	if err != nil {
		t.Fatalf("Ask failed: %v", err)
	}
	if len(resp.Choices) != 1 || resp.Choices[0].Message.Content != "ok" {
		t.Fatalf("unexpected response: %+v", resp)
	}

	calls, bodies := rt.snapshot()
	if calls != 2 {
		t.Fatalf("expected 2 requests, got %d", calls)
	}
	if !strings.Contains(bodies[0], `"web_search_options"`) {
		t.Fatalf("expected first request to include web_search_options, got %s", bodies[0])
	}
	if strings.Contains(bodies[1], `"web_search_options"`) {
		t.Fatalf("expected fallback request to omit web_search_options, got %s", bodies[1])
	}

	var second map[string]any
	if err := json.Unmarshal([]byte(bodies[1]), &second); err != nil {
		t.Fatalf("failed to parse fallback request: %v", err)
	}
	plugins, ok := second["plugins"].([]any)
	if !ok || len(plugins) != 1 {
		t.Fatalf("expected plugins to remain in fallback request, got %+v", second["plugins"])
	}
}
