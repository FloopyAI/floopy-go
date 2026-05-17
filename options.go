package floopy

import (
	"net/http"
	"time"
)

// Ptr returns a pointer to v. It is a convenience for the optional *bool /
// *int fields on Options and CacheOptions (a nil pointer means "leave the
// gateway default", a non-nil pointer sends the toggle explicitly).
//
//	floopy.WithOptions(floopy.Options{
//		Cache: &floopy.CacheOptions{Enabled: floopy.Ptr(true)},
//	})
func Ptr[T any](v T) *T { return &v }

// CacheOptions maps to the Floopy-Cache-* headers. A nil field is omitted.
type CacheOptions struct {
	// Enabled toggles the exact + semantic cache for the request.
	Enabled *bool
	// BucketMaxSize caps the number of entries per semantic cache bucket.
	BucketMaxSize *int
}

// Options are gateway behaviour toggles, mapped to Floopy-* headers and
// forwarded to every request (both OpenAI-compatible and Floopy-only). Empty
// / nil fields are omitted.
type Options struct {
	// Cache controls. Maps to Floopy-Cache-* headers.
	Cache *CacheOptions
	// PromptID is a stored prompt id; the gateway resolves it to the active
	// prompt content. Maps to Floopy-Prompt-Id.
	PromptID string
	// PromptVersion pins a prompt version. Use with PromptID. Maps to
	// Floopy-Prompt-Version.
	PromptVersion string
	// LLMSecurityEnabled toggles the LLM firewall pre-check. Maps to
	// floopy-llm-security-enabled.
	LLMSecurityEnabled *bool
}

// clientConfig is the resolved client configuration.
type clientConfig struct {
	baseURL        string
	timeout        time.Duration
	maxRetries     int
	defaultHeaders map[string]string
	defaultOptions *Options
	httpClient     *http.Client
}

// ClientOption configures a Client at construction time.
type ClientOption func(*clientConfig)

// WithBaseURL overrides the gateway base URL (default DefaultBaseURL). Use it
// for self-hosted gateways, e.g. "https://gateway.internal.acme.com/v1".
func WithBaseURL(url string) ClientOption {
	return func(c *clientConfig) { c.baseURL = url }
}

// WithTimeout sets the default per-request timeout (default DefaultTimeout).
func WithTimeout(d time.Duration) ClientOption {
	return func(c *clientConfig) { c.timeout = d }
}

// WithMaxRetries sets the retry budget for transient failures (default
// DefaultMaxRetries).
func WithMaxRetries(n int) ClientOption {
	return func(c *clientConfig) { c.maxRetries = n }
}

// WithDefaultHeaders adds headers sent on every Floopy-only request. They
// have the highest precedence and can override SDK-managed headers.
func WithDefaultHeaders(h map[string]string) ClientOption {
	return func(c *clientConfig) {
		c.defaultHeaders = make(map[string]string, len(h))
		for k, v := range h {
			c.defaultHeaders[k] = v
		}
	}
}

// WithOptions sets the default Floopy options forwarded on every request.
func WithOptions(o Options) ClientOption {
	return func(c *clientConfig) { c.defaultOptions = &o }
}

// WithHTTPClient sets the *http.Client used for Floopy-only requests. The
// SDK manages per-request timeouts via context, so a client without its own
// Timeout is recommended (an http.Client.Timeout also caps streaming).
func WithHTTPClient(hc *http.Client) ClientOption {
	return func(c *clientConfig) { c.httpClient = hc }
}

// requestConfig holds per-call overrides, merged on top of client defaults.
type requestConfig struct {
	headers map[string]string
	timeout *time.Duration
	options *Options
}

// RequestOption overrides client defaults for a single call. Every resource
// method accepts a trailing ...RequestOption.
type RequestOption func(*requestConfig)

// WithRequestHeader sets a single per-call header (highest precedence).
func WithRequestHeader(key, value string) RequestOption {
	return func(r *requestConfig) {
		if r.headers == nil {
			r.headers = map[string]string{}
		}
		r.headers[key] = value
	}
}

// WithRequestHeaders merges per-call headers (highest precedence).
func WithRequestHeaders(h map[string]string) RequestOption {
	return func(r *requestConfig) {
		if r.headers == nil {
			r.headers = make(map[string]string, len(h))
		}
		for k, v := range h {
			r.headers[k] = v
		}
	}
}

// WithRequestTimeout overrides the client timeout for a single call.
func WithRequestTimeout(d time.Duration) RequestOption {
	return func(r *requestConfig) { r.timeout = &d }
}

// WithRequestOptions overrides the Floopy options for a single call.
func WithRequestOptions(o Options) RequestOption {
	return func(r *requestConfig) { r.options = &o }
}

func newRequestConfig(opts []RequestOption) *requestConfig {
	rc := &requestConfig{}
	for _, o := range opts {
		o(rc)
	}
	return rc
}
