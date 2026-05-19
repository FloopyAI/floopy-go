package floopy

import (
	"context"
	"strconv"
)

// BatchRequestCounts is the per-batch progress summary.
type BatchRequestCounts struct {
	Total     *int `json:"total,omitempty"`
	Completed *int `json:"completed,omitempty"`
	Failed    *int `json:"failed,omitempty"`
}

// Batch mirrors the OpenAI batch object; nullable fields are pointers.
type Batch struct {
	ID            string              `json:"id"`
	Object        string              `json:"object,omitempty"`
	Endpoint      string              `json:"endpoint,omitempty"`
	Status        string              `json:"status,omitempty"`
	InputFileID   string              `json:"input_file_id,omitempty"`
	OutputFileID  *string             `json:"output_file_id,omitempty"`
	ErrorFileID   *string             `json:"error_file_id,omitempty"`
	CreatedAt     *int64              `json:"created_at,omitempty"`
	CompletedAt   *int64              `json:"completed_at,omitempty"`
	RequestCounts *BatchRequestCounts `json:"request_counts,omitempty"`
}

// BatchList is the response of BatchesService.List.
type BatchList struct {
	Object  string  `json:"object,omitempty"`
	Data    []Batch `json:"data"`
	HasMore *bool   `json:"has_more,omitempty"`
}

// BatchCreateParams are the arguments for BatchesService.Create.
type BatchCreateParams struct {
	InputFileID      string            `json:"input_file_id"`
	Endpoint         string            `json:"endpoint"`
	CompletionWindow string            `json:"completion_window"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}

// BatchListParams filters BatchesService.List.
type BatchListParams struct {
	Limit int
	After string
}

func (p BatchListParams) query() map[string]string {
	q := map[string]string{}
	if p.Limit != 0 {
		q["limit"] = strconv.Itoa(p.Limit)
	}
	if p.After != "" {
		q["after"] = p.After
	}
	return q
}

// BatchesService manages asynchronous batch jobs. Select the upstream
// with floopy.WithProvider (the floopy-provider header).
type BatchesService struct{ t *transport }

// Create starts a batch from a previously uploaded input file.
func (s *BatchesService) Create(ctx context.Context, params BatchCreateParams, opts ...RequestOption) (*Batch, error) {
	var out Batch
	_, err := s.t.do(ctx, "POST", endpointBatches, params, nil, &out, newRequestConfig(opts))
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// List returns batches for the organization.
func (s *BatchesService) List(ctx context.Context, params BatchListParams, opts ...RequestOption) (*BatchList, error) {
	var out BatchList
	_, err := s.t.do(ctx, "GET", endpointBatches, nil, params.query(), &out, newRequestConfig(opts))
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Get retrieves a single batch (poll its status).
func (s *BatchesService) Get(ctx context.Context, batchID string, opts ...RequestOption) (*Batch, error) {
	var out Batch
	_, err := s.t.do(ctx, "GET", batchByID(batchID), nil, nil, &out, newRequestConfig(opts))
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Cancel requests cancellation of an in-progress batch.
func (s *BatchesService) Cancel(ctx context.Context, batchID string, opts ...RequestOption) (*Batch, error) {
	var out Batch
	_, err := s.t.do(ctx, "POST", batchCancel(batchID), nil, nil, &out, newRequestConfig(opts))
	if err != nil {
		return nil, err
	}
	return &out, nil
}
