package floopy

import (
	"net/url"
	"time"
)

// Defaults applied when the corresponding client option is not set. They
// match the Node and Python SDKs so behaviour stays in lockstep.
const (
	// DefaultBaseURL is the public Floopy gateway. Override with
	// WithBaseURL for self-hosted gateways.
	DefaultBaseURL = "https://api.floopy.ai/v1"
	// DefaultTimeout is the per-request timeout when none is set.
	DefaultTimeout = 60 * time.Second
	// DefaultMaxRetries is the number of retries for transient failures
	// (network errors and retryable status codes).
	DefaultMaxRetries = 2

	userAgentPrefix = "floopy-sdk"
)

// Header names recognised by the gateway. Names are case-insensitive on the
// wire but kept verbatim to match the Node/Python SDKs and the gateway docs.
const (
	headerCacheEnabled       = "Floopy-Cache-Enabled"
	headerCacheBucketMaxSize = "Floopy-Cache-Bucket-Max-Size"
	headerPromptID           = "Floopy-Prompt-Id"
	headerPromptVersion      = "Floopy-Prompt-Version"
	headerLLMSecurityEnabled = "floopy-llm-security-enabled"
	headerConfirm            = "X-Floopy-Confirm"
	headerRequestID          = "X-Request-Id"
	headerAuthorization      = "Authorization"
	headerContentType        = "Content-Type"
	headerUserAgent          = "User-Agent"
	headerFloopySDK          = "X-Floopy-SDK"
)

// ConfirmExperiments is the X-Floopy-Confirm value the gateway requires on
// experiment create/rollback (gateway control SEC-009). The experiments
// resource injects it automatically.
const ConfirmExperiments = "experiments"

// Gateway endpoints, relative to the base URL.
const (
	endpointFeedback        = "/feedback"
	endpointDecisions       = "/decisions"
	endpointExperiments     = "/experiments"
	endpointConstraints     = "/constraints"
	endpointExportDecisions = "/export/decisions"
	endpointRoutingExplain  = "/routing/explain"
	endpointEvaluations     = "/evaluations"
)

// pathSeg percent-encodes a single path segment (matches JS
// encodeURIComponent / Python urllib.parse.quote(safe="")).
func pathSeg(v string) string {
	return url.PathEscape(v)
}

func decisionByID(id string) string      { return endpointDecisions + "/" + pathSeg(id) }
func sessionByID(id string) string       { return "/session/" + pathSeg(id) }
func experimentResults(id string) string { return endpointExperiments + "/" + pathSeg(id) + "/results" }
func experimentRollback(id string) string {
	return endpointExperiments + "/" + pathSeg(id) + "/rollback"
}
func evaluationByID(id string) string    { return endpointEvaluations + "/" + pathSeg(id) }
func evaluationResults(id string) string { return endpointEvaluations + "/" + pathSeg(id) + "/results" }
func evaluationCancel(id string) string  { return endpointEvaluations + "/" + pathSeg(id) + "/cancel" }
