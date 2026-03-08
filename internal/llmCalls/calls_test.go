package llmcalls

import (
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
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
