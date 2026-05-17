package floopy

import (
	"context"
	"encoding/json"
	"iter"
	"strconv"
)

// Decision is one row of the per-request decision audit trail. Nullable
// gateway fields are pointers (nil == JSON null).
type Decision struct {
	RequestID        string          `json:"request_id"`
	SessionID        *string         `json:"session_id"`
	RequestCreatedAt string          `json:"request_created_at"`
	Provider         *string         `json:"provider"`
	Model            *string         `json:"model"`
	Status           string          `json:"status"`
	LatencyMs        *int            `json:"latency_ms"`
	CostMicroUSD     *int64          `json:"cost_micro_usd"`
	CacheEnabled     *bool           `json:"cache_enabled"`
	Threat           *string         `json:"threat"`
	DecisionTrace    json.RawMessage `json:"decision_trace"`
	Confidence       *float64        `json:"confidence"`
	ConfidenceReason *string         `json:"confidence_reason"`
	Explanation      *string         `json:"explanation"`
}

// DecisionListPage is one page of DecisionsService.List.
type DecisionListPage struct {
	Items      []Decision `json:"items"`
	NextCursor *string    `json:"next_cursor"`
	HasMore    bool       `json:"has_more"`
}

// DecisionListParams filters DecisionsService.List / .Pages / .Iterate. Zero
// values are omitted.
type DecisionListParams struct {
	SessionID string
	From      string // RFC3339
	To        string // RFC3339
	Limit     int
	Cursor    string
}

func (p DecisionListParams) query() map[string]string {
	q := map[string]string{}
	if p.SessionID != "" {
		q["session_id"] = p.SessionID
	}
	if p.From != "" {
		q["from"] = p.From
	}
	if p.To != "" {
		q["to"] = p.To
	}
	if p.Limit > 0 {
		q["limit"] = strconv.Itoa(p.Limit)
	}
	if p.Cursor != "" {
		q["cursor"] = p.Cursor
	}
	return q
}

// DecisionsService reads the decision audit trail.
type DecisionsService struct{ t *transport }

// Get fetches one decision by request id.
func (s *DecisionsService) Get(ctx context.Context, requestID string, opts ...RequestOption) (*Decision, error) {
	var out Decision
	_, err := s.t.do(ctx, "GET", decisionByID(requestID), nil, nil, &out, newRequestConfig(opts))
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// List fetches a single page of decisions.
func (s *DecisionsService) List(ctx context.Context, params DecisionListParams, opts ...RequestOption) (*DecisionListPage, error) {
	return s.fetchPage(ctx, params, newRequestConfig(opts))
}

func (s *DecisionsService) fetchPage(ctx context.Context, params DecisionListParams, rc *requestConfig) (*DecisionListPage, error) {
	var out DecisionListPage
	_, err := s.t.do(ctx, "GET", endpointDecisions, nil, params.query(), &out, rc)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Pages yields one *DecisionListPage per network round-trip until the gateway
// reports no more pages. Iteration stops on the first error (yielded as the
// second value).
func (s *DecisionsService) Pages(ctx context.Context, params DecisionListParams, opts ...RequestOption) iter.Seq2[*DecisionListPage, error] {
	rc := newRequestConfig(opts)
	return func(yield func(*DecisionListPage, error) bool) {
		cursor := params.Cursor
		for {
			p := params
			p.Cursor = cursor
			page, err := s.fetchPage(ctx, p, rc)
			if err != nil {
				yield(nil, err)
				return
			}
			if !yield(page, nil) {
				return
			}
			if !page.HasMore || page.NextCursor == nil || *page.NextCursor == "" {
				return
			}
			cursor = *page.NextCursor
		}
	}
}

// Iterate yields every decision across all pages.
func (s *DecisionsService) Iterate(ctx context.Context, params DecisionListParams, opts ...RequestOption) iter.Seq2[Decision, error] {
	return func(yield func(Decision, error) bool) {
		for page, err := range s.Pages(ctx, params, opts...) {
			if err != nil {
				yield(Decision{}, err)
				return
			}
			for _, d := range page.Items {
				if !yield(d, nil) {
					return
				}
			}
		}
	}
}
