package floopy

import "context"

// OrgConstraints are the org-wide spend and rate constraints. All fields are
// pointers: nil means "no limit" (and, on Put, clears any existing limit
// server-side). Put is full-replace, so every field is always sent.
type OrgConstraints struct {
	// CostLimitMonthlyUSD is a hard cap (USD) on total monthly spend.
	CostLimitMonthlyUSD *float64 `json:"cost_limit_monthly_usd"`
	// TokenWindowSeconds is the sliding window for the token rate limit.
	TokenWindowSeconds *int `json:"token_window_seconds"`
	// MaxTokensPerWindow caps tokens per TokenWindowSeconds.
	MaxTokensPerWindow *int `json:"max_tokens_per_window"`
	// MaxRequestsPerMinute caps requests per minute per API key.
	MaxRequestsPerMinute *int `json:"max_requests_per_minute"`
}

// ConstraintsService reads and replaces org constraints.
type ConstraintsService struct{ t *transport }

// Get returns the current org constraints.
func (s *ConstraintsService) Get(ctx context.Context, opts ...RequestOption) (*OrgConstraints, error) {
	var out OrgConstraints
	_, err := s.t.do(ctx, "GET", endpointConstraints, nil, nil, &out, newRequestConfig(opts))
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Put full-replaces the org constraints. Any field left nil is reset to null
// server-side (matches the gateway's PUT semantics).
func (s *ConstraintsService) Put(ctx context.Context, constraints OrgConstraints, opts ...RequestOption) (*OrgConstraints, error) {
	var out OrgConstraints
	_, err := s.t.do(ctx, "PUT", endpointConstraints, constraints, nil, &out, newRequestConfig(opts))
	if err != nil {
		return nil, err
	}
	return &out, nil
}
