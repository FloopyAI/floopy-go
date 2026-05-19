package floopy

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"math/rand/v2"
	"mime/multipart"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// retryableStatus is the set of HTTP statuses the SDK retries (matches the
// Node/Python SDKs).
var retryableStatus = map[int]bool{
	408: true, 409: true, 425: true, 429: true,
	500: true, 502: true, 503: true, 504: true,
}

// transport is the internal HTTP layer for Floopy-only endpoints: bearer
// auth, Floopy-* header injection, bounded retries with exponential backoff +
// jitter (Retry-After honoured), per-call timeouts, request-id capture, and
// mapping of non-2xx responses to typed errors. The OpenAI-compatible surface
// does not go through here — it is delegated to openai-go.
//
// Security: the API key only ever appears in the Authorization header; the
// SDK never logs request or response bodies.
type transport struct {
	apiKey         string
	baseURL        string
	timeout        time.Duration
	maxRetries     int
	defaultHeaders map[string]string
	defaultOptions *Options
	httpClient     *http.Client
}

func newTransport(apiKey string, cfg *clientConfig) (*transport, error) {
	if apiKey == "" {
		return nil, &FloopyError{Message: "api key is required to construct a Floopy client"}
	}
	hc := cfg.httpClient
	if hc == nil {
		hc = &http.Client{}
	}
	return &transport{
		apiKey:         apiKey,
		baseURL:        strings.TrimRight(cfg.baseURL, "/"),
		timeout:        cfg.timeout,
		maxRetries:     cfg.maxRetries,
		defaultHeaders: cfg.defaultHeaders,
		defaultOptions: cfg.defaultOptions,
		httpClient:     hc,
	}, nil
}

func (t *transport) authAndUAHeaders() map[string]string {
	return map[string]string{
		headerAuthorization: "Bearer " + t.apiKey,
		headerUserAgent: fmt.Sprintf("%s/%s go/%s", userAgentPrefix, Version,
			strings.TrimPrefix(runtime.Version(), "go")),
	}
}

// defaultRequestHeaders is the header set forwarded to the OpenAI delegate.
func (t *transport) defaultRequestHeaders() map[string]string {
	return mergeHeaders(
		t.defaultHeaders,
		buildFloopyHeaders(t.defaultOptions),
		t.authAndUAHeaders(),
	)
}

func (t *transport) buildRequestHeaders(rc *requestConfig) map[string]string {
	var perCallOpts *Options
	var perCallHeaders map[string]string
	if rc != nil {
		perCallOpts = rc.options
		perCallHeaders = rc.headers
	}
	return mergeHeaders(
		t.defaultHeaders,
		buildFloopyHeaders(t.defaultOptions),
		buildFloopyHeaders(perCallOpts),
		t.authAndUAHeaders(),
		perCallHeaders,
	)
}

func (t *transport) buildURL(path string, query map[string]string) string {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	u := t.baseURL + path
	if len(query) > 0 {
		vals := url.Values{}
		for k, v := range query {
			if v != "" {
				vals.Set(k, v)
			}
		}
		if enc := vals.Encode(); enc != "" {
			u += "?" + enc
		}
	}
	return u
}

func (t *transport) timeoutFor(rc *requestConfig) time.Duration {
	if rc != nil && rc.timeout != nil {
		return *rc.timeout
	}
	return t.timeout
}

// do issues a request with retries and decodes a 2xx JSON body into out (a
// pointer; pass nil to discard the body). It returns the X-Request-Id header
// when present.
func (t *transport) do(
	ctx context.Context,
	method, path string,
	body any,
	query map[string]string,
	out any,
	rc *requestConfig,
) (string, error) {
	resp, err := t.requestRaw(ctx, method, path, body, query, rc)
	if err != nil {
		return "", err
	}
	return t.decodeJSON(resp, out)
}

// decodeJSON consumes resp (closing its body) and unmarshals a 2xx body
// into out (nil discards it). Shared by do and doMultipart.
func (t *transport) decodeJSON(resp *http.Response, out any) (string, error) {
	defer resp.Body.Close()
	requestID := resp.Header.Get(headerRequestID)
	if resp.StatusCode == http.StatusNoContent || out == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return requestID, nil
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return requestID, newConnectionError("error reading Floopy gateway response", err)
	}
	if len(raw) == 0 {
		return requestID, nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return requestID, &FloopyError{Message: "failed to decode gateway response: " + err.Error(), RequestID: requestID, cause: err}
	}
	return requestID, nil
}

// doBytes issues a request and returns the raw 2xx body (no JSON
// decoding) — used to download file content.
func (t *transport) doBytes(
	ctx context.Context,
	method, path string,
	query map[string]string,
	rc *requestConfig,
) ([]byte, string, error) {
	resp, err := t.requestRaw(ctx, method, path, nil, query, rc)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	requestID := resp.Header.Get(headerRequestID)
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, requestID, newConnectionError("error reading Floopy gateway response", err)
	}
	return raw, requestID, nil
}

// doMultipart sends a multipart/form-data request (file upload): one file
// part plus text fields. The JSON 2xx body is decoded into out.
func (t *transport) doMultipart(
	ctx context.Context,
	method, path string,
	fields map[string]string,
	fileField, filename string,
	content []byte,
	out any,
	rc *requestConfig,
) (string, error) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for k, v := range fields {
		if err := mw.WriteField(k, v); err != nil {
			return "", &FloopyError{Message: "failed to build multipart body: " + err.Error(), cause: err}
		}
	}
	fw, err := mw.CreateFormFile(fileField, filename)
	if err != nil {
		return "", &FloopyError{Message: "failed to build multipart body: " + err.Error(), cause: err}
	}
	if _, err := fw.Write(content); err != nil {
		return "", &FloopyError{Message: "failed to build multipart body: " + err.Error(), cause: err}
	}
	if err := mw.Close(); err != nil {
		return "", &FloopyError{Message: "failed to build multipart body: " + err.Error(), cause: err}
	}
	u := t.buildURL(path, nil)
	headers := t.buildRequestHeaders(rc)
	headers[headerContentType] = mw.FormDataContentType()
	resp, err := t.sendWithRetry(ctx, method, u, headers, buf.Bytes(), rc)
	if err != nil {
		return "", err
	}
	return t.decodeJSON(resp, out)
}

func (t *transport) requestRaw(
	ctx context.Context,
	method, path string,
	body any,
	query map[string]string,
	rc *requestConfig,
) (*http.Response, error) {
	u := t.buildURL(path, query)
	headers := t.buildRequestHeaders(rc)

	var bodyBytes []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, &FloopyError{Message: "failed to encode request body: " + err.Error(), cause: err}
		}
		bodyBytes = b
		if _, ok := headers[headerContentType]; !ok {
			headers[headerContentType] = "application/json"
		}
	}
	return t.sendWithRetry(ctx, method, u, headers, bodyBytes, rc)
}

func (t *transport) sendWithRetry(
	ctx context.Context,
	method, u string,
	headers map[string]string,
	bodyBytes []byte,
	rc *requestConfig,
) (*http.Response, error) {
	timeout := t.timeoutFor(rc)

	attempt := 0
	for {
		attemptCtx, cancel := context.WithTimeout(ctx, timeout)
		var reader io.Reader
		if bodyBytes != nil {
			reader = bytes.NewReader(bodyBytes)
		}
		req, err := http.NewRequestWithContext(attemptCtx, method, u, reader)
		if err != nil {
			cancel()
			return nil, &FloopyError{Message: "failed to build request: " + err.Error(), cause: err}
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		resp, err := t.httpClient.Do(req)
		if err != nil {
			cancel()
			if ctx.Err() != nil && !errors.Is(ctx.Err(), context.DeadlineExceeded) {
				// Parent context cancelled by the caller.
				return nil, &FloopyError{Message: "request cancelled by caller", cause: ctx.Err()}
			}
			if isTimeout(err) || errors.Is(err, context.DeadlineExceeded) {
				return nil, newTimeoutError(fmt.Sprintf("request timed out after %s", timeout), err)
			}
			if attempt < t.maxRetries {
				if serr := t.sleepBackoff(ctx, attempt, ""); serr != nil {
					return nil, serr
				}
				attempt++
				continue
			}
			return nil, newConnectionError("network error talking to Floopy gateway", err)
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			// Hand the body to the caller; cancel fires when they close it.
			return wrapBody(resp, cancel), nil
		}

		if attempt < t.maxRetries && retryableStatus[resp.StatusCode] {
			retryAfter := resp.Header.Get("Retry-After")
			drainClose(resp)
			cancel()
			if serr := t.sleepBackoff(ctx, attempt, retryAfter); serr != nil {
				return nil, serr
			}
			attempt++
			continue
		}

		mapped := t.errorFromResponse(resp)
		drainClose(resp)
		cancel()
		return nil, mapped
	}
}

func (t *transport) errorFromResponse(resp *http.Response) error {
	requestID := resp.Header.Get(headerRequestID)
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	mapped := errorFromStatus(resp.StatusCode, parseErrorBody(raw), requestID)
	if rle, ok := mapped.(*RateLimitError); ok {
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if secs, err := strconv.Atoi(ra); err == nil {
				rle.RetryAfterSeconds = secs
			}
		}
	}
	return mapped
}

func (t *transport) sleepBackoff(ctx context.Context, attempt int, retryAfter string) error {
	var d time.Duration
	if retryAfter != "" {
		if secs, err := strconv.Atoi(retryAfter); err == nil && secs > 0 {
			d = time.Duration(secs) * time.Second
		}
	}
	if d == 0 {
		base := 250 * time.Millisecond * (1 << attempt)
		jitter := time.Duration(rand.Float64() * float64(base) * 0.25)
		d = base + jitter
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return &FloopyError{Message: "request cancelled while backing off", cause: ctx.Err()}
	}
}

// streamLines streams a JSONL response, yielding one raw JSON object per
// non-empty line. The underlying connection is closed when iteration ends
// (including an early break). Errors are yielded as the second value.
func (t *transport) streamLines(
	ctx context.Context,
	method, path string,
	query map[string]string,
	rc *requestConfig,
) iter.Seq2[json.RawMessage, error] {
	return func(yield func(json.RawMessage, error) bool) {
		resp, err := t.requestRaw(ctx, method, path, nil, query, rc)
		if err != nil {
			yield(nil, err)
			return
		}
		defer resp.Body.Close()
		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
		for scanner.Scan() {
			line := bytes.TrimSpace(scanner.Bytes())
			if len(line) == 0 {
				continue
			}
			cp := make([]byte, len(line))
			copy(cp, line)
			if !yield(json.RawMessage(cp), nil) {
				return
			}
		}
		if err := scanner.Err(); err != nil {
			if isTimeout(err) {
				yield(nil, newTimeoutError("request timed out while streaming", err))
				return
			}
			yield(nil, newConnectionError("network error while streaming from Floopy gateway", err))
		}
	}
}

// bodyCloser wraps the response body so closing it also cancels the
// per-attempt context.
type bodyCloser struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (b *bodyCloser) Close() error {
	err := b.ReadCloser.Close()
	b.cancel()
	return err
}

func wrapBody(resp *http.Response, cancel context.CancelFunc) *http.Response {
	resp.Body = &bodyCloser{ReadCloser: resp.Body, cancel: cancel}
	return resp
}

func drainClose(resp *http.Response) {
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	_ = resp.Body.Close()
}

func isTimeout(err error) bool {
	var te interface{ Timeout() bool }
	return errors.As(err, &te) && te.Timeout()
}
