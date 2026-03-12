package llmcalls

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nikaydo/personal-assistant/internal/config"
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
