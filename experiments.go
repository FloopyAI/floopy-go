package floopy

import (
	"context"
	"iter"
	"strconv"
)

// ExperimentStatus is the lifecycle state of an experiment.
type ExperimentStatus = string

// Experiment is an A/B routing experiment.
type Experiment struct {
	ID                    string           `json:"id"`
	Name                  string           `json:"name"`
	Description           *string          `json:"description"`
	Status                ExperimentStatus `json:"status"`
	VariantARoutingRuleID string           `json:"variant_a_routing_rule_id"`
	VariantBRoutingRuleID string           `json:"variant_b_routing_rule_id"`
	SplitPercentage       int              `json:"split_percentage"`
	CreatedAt             string           `json:"created_at"`
	RolledBackAt          *string          `json:"rolled_back_at"`
}

// ExperimentListPage is one page of ExperimentsService.List.
type ExperimentListPage struct {
	Items      []Experiment `json:"items"`
	NextCursor *string      `json:"next_cursor"`
	HasMore    bool         `json:"has_more"`
}

// VariantResults holds the aggregated metrics for one experiment variant.
type VariantResults struct {
	RoutingRuleID       string  `json:"routing_rule_id"`
	SampleSize          int     `json:"sample_size"`
	SuccessRate         float64 `json:"success_rate"`
	AverageLatencyMs    float64 `json:"average_latency_ms"`
	AverageCostMicroUSD float64 `json:"average_cost_micro_usd"`
}

// ExperimentResults is the computed outcome of an experiment.
type ExperimentResults struct {
	ExperimentID string         `json:"experiment_id"`
	VariantA     VariantResults `json:"variant_a"`
	VariantB     VariantResults `json:"variant_b"`
	// Winner is "A", "B", "tie", or nil when undecided.
	Winner     *string `json:"winner"`
	ComputedAt string  `json:"computed_at"`
}

// ExperimentListParams filters ExperimentsService.List. Zero values omitted.
type ExperimentListParams struct {
	Status ExperimentStatus
	From   string
	To     string
	Limit  int
	Cursor string
}

func (p ExperimentListParams) query() map[string]string {
	q := map[string]string{}
	if p.Status != "" {
		q["status"] = p.Status
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

// ExperimentCreateParams are the arguments for ExperimentsService.Create.
type ExperimentCreateParams struct {
	Name                  string `json:"name"`
	VariantARoutingRuleID string `json:"variant_a_routing_rule_id"`
	VariantBRoutingRuleID string `json:"variant_b_routing_rule_id"`
	Description           string `json:"description,omitempty"`
	SplitPercentage       *int   `json:"split_percentage,omitempty"`
}

// ExperimentsService manages A/B routing experiments.
type ExperimentsService struct{ t *transport }

// withConfirm injects X-Floopy-Confirm: experiments (gateway SEC-009 requires
// it on create/rollback) without mutating the caller's options.
func withConfirm(opts []RequestOption) []RequestOption {
	return append(opts[:len(opts):len(opts)], WithRequestHeader(headerConfirm, ConfirmExperiments))
}

// List fetches a single page of experiments.
func (s *ExperimentsService) List(ctx context.Context, params ExperimentListParams, opts ...RequestOption) (*ExperimentListPage, error) {
	var out ExperimentListPage
	_, err := s.t.do(ctx, "GET", endpointExperiments, nil, params.query(), &out, newRequestConfig(opts))
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Pages yields one *ExperimentListPage per round-trip until exhausted.
func (s *ExperimentsService) Pages(ctx context.Context, params ExperimentListParams, opts ...RequestOption) iter.Seq2[*ExperimentListPage, error] {
	return func(yield func(*ExperimentListPage, error) bool) {
		cursor := params.Cursor
		for {
			p := params
			p.Cursor = cursor
			page, err := s.List(ctx, p, opts...)
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

// Create starts an experiment. The X-Floopy-Confirm: experiments header is
// injected automatically.
func (s *ExperimentsService) Create(ctx context.Context, params ExperimentCreateParams, opts ...RequestOption) (*Experiment, error) {
	var out Experiment
	_, err := s.t.do(ctx, "POST", endpointExperiments, params, nil, &out, newRequestConfig(withConfirm(opts)))
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Rollback reverts an experiment. The X-Floopy-Confirm: experiments header is
// injected automatically.
func (s *ExperimentsService) Rollback(ctx context.Context, experimentID string, opts ...RequestOption) (*Experiment, error) {
	var out Experiment
	_, err := s.t.do(ctx, "POST", experimentRollback(experimentID), nil, nil, &out, newRequestConfig(withConfirm(opts)))
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Results returns the computed outcome of an experiment.
func (s *ExperimentsService) Results(ctx context.Context, experimentID string, opts ...RequestOption) (*ExperimentResults, error) {
	var out ExperimentResults
	_, err := s.t.do(ctx, "GET", experimentResults(experimentID), nil, nil, &out, newRequestConfig(opts))
	if err != nil {
		return nil, err
	}
	return &out, nil
}
