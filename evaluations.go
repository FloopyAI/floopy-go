package floopy

import (
	"context"
	"strconv"
)

// EvaluationStatus is the lifecycle state of an evaluation run.
type EvaluationStatus = string

// EvaluationRun is a dataset evaluation.
type EvaluationRun struct {
	ID         string           `json:"id"`
	DatasetID  string           `json:"dataset_id"`
	Model      string           `json:"model"`
	PromptID   *string          `json:"prompt_id"`
	Status     EvaluationStatus `json:"status"`
	Config     map[string]any   `json:"config"`
	CreatedAt  string           `json:"created_at"`
	StartedAt  *string          `json:"started_at"`
	FinishedAt *string          `json:"finished_at"`
}

// EvaluationResultRow is one scored row of an evaluation.
type EvaluationResultRow struct {
	ID        string         `json:"id"`
	RunID     string         `json:"run_id"`
	InputID   string         `json:"input_id"`
	Output    string         `json:"output"`
	Score     *float64       `json:"score"`
	Metadata  map[string]any `json:"metadata"`
	CreatedAt string         `json:"created_at"`
}

// EvaluationResultsPage is one page of EvaluationsService.Results.
type EvaluationResultsPage struct {
	Items      []EvaluationResultRow `json:"items"`
	NextCursor *string               `json:"next_cursor"`
	HasMore    bool                  `json:"has_more"`
}

// EvaluationCreateParams are the arguments for EvaluationsService.Create.
type EvaluationCreateParams struct {
	DatasetID string         `json:"dataset_id"`
	Model     string         `json:"model"`
	PromptID  string         `json:"prompt_id,omitempty"`
	Config    map[string]any `json:"config,omitempty"`
}

// EvaluationsService runs and inspects dataset evaluations.
type EvaluationsService struct{ t *transport }

// Create starts an evaluation run.
func (s *EvaluationsService) Create(ctx context.Context, params EvaluationCreateParams, opts ...RequestOption) (*EvaluationRun, error) {
	var out EvaluationRun
	_, err := s.t.do(ctx, "POST", endpointEvaluations, params, nil, &out, newRequestConfig(opts))
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Get fetches an evaluation run by id.
func (s *EvaluationsService) Get(ctx context.Context, evaluationID string, opts ...RequestOption) (*EvaluationRun, error) {
	var out EvaluationRun
	_, err := s.t.do(ctx, "GET", evaluationByID(evaluationID), nil, nil, &out, newRequestConfig(opts))
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Cancel cancels a running evaluation.
func (s *EvaluationsService) Cancel(ctx context.Context, evaluationID string, opts ...RequestOption) (*EvaluationRun, error) {
	var out EvaluationRun
	_, err := s.t.do(ctx, "POST", evaluationCancel(evaluationID), nil, nil, &out, newRequestConfig(opts))
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Results returns a page of scored rows for an evaluation.
func (s *EvaluationsService) Results(ctx context.Context, evaluationID string, limit int, cursor string, opts ...RequestOption) (*EvaluationResultsPage, error) {
	q := map[string]string{}
	if limit > 0 {
		q["limit"] = strconv.Itoa(limit)
	}
	if cursor != "" {
		q["cursor"] = cursor
	}
	var out EvaluationResultsPage
	_, err := s.t.do(ctx, "GET", evaluationResults(evaluationID), nil, q, &out, newRequestConfig(opts))
	if err != nil {
		return nil, err
	}
	return &out, nil
}
