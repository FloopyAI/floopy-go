package floopy

import (
	"encoding/json"
	"errors"
	"fmt"
)

// FloopyError is the base error returned by every Floopy-only resource.
// Transport timeouts and connection failures are reported as *TimeoutError
// and *ConnectionError respectively. Errors from the OpenAI-compatible
// surface (OpenAI().Chat / .Embeddings / .Models) come from the upstream
// openai-go package, not this type.
//
// Use errors.As with the concrete pointer types (e.g. *RateLimitError) to
// branch on the failure mode, or AsError to recover the base *FloopyError
// from any Floopy error.
type FloopyError struct {
	// Message is the human-readable error message (gateway-provided when
	// available, otherwise "HTTP <status>").
	Message string
	// Status is the HTTP status code, or 0 for transport-level errors.
	Status int
	// Code is the gateway error code, when present.
	Code string
	// RequestID echoes the X-Request-Id response header, when present.
	RequestID string
	// Body is the parsed error body (decoded JSON) or the raw string.
	Body any

	cause error
}

func (e *FloopyError) Error() string {
	if e.RequestID != "" {
		return fmt.Sprintf("floopy: %s (status=%d request_id=%s)", e.Message, e.Status, e.RequestID)
	}
	if e.Status != 0 {
		return fmt.Sprintf("floopy: %s (status=%d)", e.Message, e.Status)
	}
	return "floopy: " + e.Message
}

func (e *FloopyError) Unwrap() error      { return e.cause }
func (e *FloopyError) base() *FloopyError { return e }

// floopyError is implemented by every Floopy error so AsError can recover the
// shared *FloopyError regardless of the concrete type.
type floopyError interface {
	error
	base() *FloopyError
}

// AsError extracts the base *FloopyError from any error in err's chain
// produced by this SDK. It returns false for non-Floopy errors (e.g.
// openai-go errors from the OpenAI-compatible surface).
func AsError(err error) (*FloopyError, bool) {
	var fe floopyError
	if errors.As(err, &fe) {
		return fe.base(), true
	}
	return nil, false
}

// AuthError is returned on HTTP 401, and on 403 without a feature field.
type AuthError struct{ *FloopyError }

// PlanError is returned on HTTP 403 when the response carries a feature
// field: the current plan does not include the requested capability.
type PlanError struct {
	*FloopyError
	// Feature is the plan capability the request needed.
	Feature string
}

// RateLimitError is returned on HTTP 429.
type RateLimitError struct {
	*FloopyError
	// RetryAfterSeconds is the Retry-After header value, or 0 if absent.
	RetryAfterSeconds int
}

// ValidationError is returned on HTTP 400.
type ValidationError struct{ *FloopyError }

// NotFoundError is returned on HTTP 404.
type NotFoundError struct{ *FloopyError }

// ConflictError is returned on HTTP 409.
type ConflictError struct{ *FloopyError }

// ServerError is returned on HTTP 5xx.
type ServerError struct{ *FloopyError }

// TimeoutError is returned when a request exceeds its deadline.
type TimeoutError struct{ *FloopyError }

// ConnectionError is returned on a network failure talking to the gateway.
type ConnectionError struct{ *FloopyError }

func newTimeoutError(msg string, cause error) *TimeoutError {
	return &TimeoutError{&FloopyError{Message: msg, cause: cause}}
}

func newConnectionError(msg string, cause error) *ConnectionError {
	return &ConnectionError{&FloopyError{Message: msg, cause: cause}}
}

// errorFromStatus maps an HTTP status + parsed body to the right typed error.
// The gateway returns {"error": {"code", "message", "feature"}}; a plain-text
// body is preserved on Body and surfaced generically.
func errorFromStatus(status int, body any, requestID string) error {
	var message, code, feature string
	if m, ok := body.(map[string]any); ok {
		if eo, ok := m["error"].(map[string]any); ok {
			message, _ = eo["message"].(string)
			code, _ = eo["code"].(string)
			feature, _ = eo["feature"].(string)
		}
	}
	if message == "" {
		message = fmt.Sprintf("HTTP %d", status)
	}
	base := &FloopyError{Message: message, Status: status, Code: code, RequestID: requestID, Body: body}

	switch {
	case status == 400:
		return &ValidationError{base}
	case status == 401:
		return &AuthError{base}
	case status == 403:
		if feature != "" {
			return &PlanError{FloopyError: base, Feature: feature}
		}
		return &AuthError{base}
	case status == 404:
		return &NotFoundError{base}
	case status == 409:
		return &ConflictError{base}
	case status == 429:
		return &RateLimitError{FloopyError: base}
	case status >= 500:
		return &ServerError{base}
	default:
		return base
	}
}

// parseErrorBody best-effort decodes a JSON error body, falling back to the
// raw string.
func parseErrorBody(raw []byte) any {
	if len(raw) == 0 {
		return nil
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	return v
}
