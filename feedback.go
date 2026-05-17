package floopy

import "context"

// FeedbackSubmitResponse is returned by FeedbackService.Submit.
type FeedbackSubmitResponse struct {
	// Duplicate is true when feedback for this session was already recorded.
	Duplicate bool `json:"duplicate"`
	// SessionID echoes the session the feedback was attached to.
	SessionID string `json:"session_id"`
}

// FeedbackSubmitParams are the arguments for FeedbackService.Submit.
type FeedbackSubmitParams struct {
	// Score is the user rating (e.g. 0-10 / NPS-style).
	Score int `json:"score"`
	// Useful flags whether the response was useful.
	Useful bool `json:"useful"`
	// SessionID is optional; defaults to the most recent session for the key.
	SessionID string `json:"session_id,omitempty"`
}

// FeedbackService submits feedback for a completed request or session.
type FeedbackService struct{ t *transport }

// Submit posts feedback. SessionID is usually the chat completion id.
func (s *FeedbackService) Submit(ctx context.Context, params FeedbackSubmitParams, opts ...RequestOption) (*FeedbackSubmitResponse, error) {
	var out FeedbackSubmitResponse
	_, err := s.t.do(ctx, "POST", endpointFeedback, params, nil, &out, newRequestConfig(opts))
	if err != nil {
		return nil, err
	}
	return &out, nil
}
