package floopy

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClientRequiresAPIKey(t *testing.T) {
	if _, err := NewClient(""); err == nil {
		t.Fatal("expected error for empty api key")
	} else if _, ok := AsError(err); !ok {
		t.Fatalf("expected *FloopyError, got %T", err)
	}
}

func TestLazyOpenAIDelegateReused(t *testing.T) {
	c, err := NewClient("fl_test")
	if err != nil {
		t.Fatal(err)
	}
	first := c.OpenAI()
	if first != c.OpenAI() {
		t.Fatal("expected the same lazily-built openai client to be reused")
	}
}

func TestIncludesFloopyHeadersWhenOptionsSet(t *testing.T) {
	var got http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
		w.Header().Set(headerRequestID, "req_x")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c, err := NewClient("fl_test",
		WithBaseURL(srv.URL+"/v1"),
		WithOptions(Options{
			Cache:              &CacheOptions{Enabled: Ptr(true), BucketMaxSize: Ptr(4)},
			PromptID:           "p1",
			LLMSecurityEnabled: Ptr(true),
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.transport.do(context.Background(), "GET", "/decisions/abc", nil, nil, &struct{}{}, nil); err != nil {
		t.Fatal(err)
	}
	if got.Get(headerAuthorization) != "Bearer fl_test" {
		t.Errorf("authorization = %q", got.Get(headerAuthorization))
	}
	if got.Get(headerCacheEnabled) != "true" {
		t.Errorf("cache-enabled = %q", got.Get(headerCacheEnabled))
	}
	if got.Get(headerCacheBucketMaxSize) != "4" {
		t.Errorf("cache-bucket-max-size = %q", got.Get(headerCacheBucketMaxSize))
	}
	if got.Get(headerPromptID) != "p1" {
		t.Errorf("prompt-id = %q", got.Get(headerPromptID))
	}
	if got.Get(headerLLMSecurityEnabled) != "true" {
		t.Errorf("llm-security-enabled = %q", got.Get(headerLLMSecurityEnabled))
	}
	if ua := got.Get(headerUserAgent); ua == "" || ua[:len(userAgentPrefix)] != userAgentPrefix {
		t.Errorf("user-agent = %q", ua)
	}
}

func TestMapsNon2xxIntoTypedError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "5")
		w.Header().Set(headerRequestID, "req_x")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"code":"rate_limited","message":"slow down"}}`))
	}))
	defer srv.Close()

	c, _ := NewClient("fl_test", WithBaseURL(srv.URL+"/v1"), WithMaxRetries(0))
	_, err := c.Decisions.List(context.Background(), DecisionListParams{})
	var rle *RateLimitError
	if !errors.As(err, &rle) {
		t.Fatalf("expected *RateLimitError, got %T (%v)", err, err)
	}
	if rle.Status != 429 || rle.RequestID != "req_x" || rle.RetryAfterSeconds != 5 {
		t.Errorf("unexpected error fields: %+v", rle.FloopyError)
	}
	if base, ok := AsError(err); !ok || base.Status != 429 {
		t.Errorf("AsError failed to recover base: %v %v", base, ok)
	}
}

func TestRetries5xxUpToMaxRetries(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{}`))
			return
		}
		_, _ = w.Write([]byte(`{"items":[],"next_cursor":null,"has_more":false}`))
	}))
	defer srv.Close()

	c, _ := NewClient("fl_test", WithBaseURL(srv.URL+"/v1"), WithMaxRetries(2))
	if _, err := c.Decisions.List(context.Background(), DecisionListParams{}); err != nil {
		t.Fatal(err)
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestConnectionErrorWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close() // force a connection failure

	c, _ := NewClient("fl_test", WithBaseURL(srv.URL+"/v1"), WithMaxRetries(0))
	_, err := c.Decisions.List(context.Background(), DecisionListParams{})
	var ce *ConnectionError
	if !errors.As(err, &ce) {
		t.Fatalf("expected *ConnectionError, got %T (%v)", err, err)
	}
}
