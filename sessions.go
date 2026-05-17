package floopy

import (
	"context"

	"github.com/openai/openai-go/v3"
)

// SessionTurn is the provenance for one reconstructed exchange.
type SessionTurn struct {
	RequestID string `json:"request_id"`
	CreatedAt string `json:"created_at"` // RFC3339
	Model     string `json:"model"`
	Provider  string `json:"provider"`
}

// Session is a conversation restored from Floopy's stored logs. Messages is
// chronological (oldest -> newest) and is a drop-in for the Messages field of
// a follow-up Chat.Completions.New call.
type Session struct {
	SessionID string                                   `json:"session_id"`
	Messages  []openai.ChatCompletionMessageParamUnion `json:"messages"`
	// TurnCount is the number of stored turns that contributed to Messages.
	TurnCount int           `json:"turn_count"`
	Turns     []SessionTurn `json:"turns"`
}

// SessionsService restores stored conversations.
type SessionsService struct{ t *transport }

// Get restores a stored conversation by its session id (the value sent on
// the floopy-session-id header at request time). Scoped to the API key's
// organization.
func (s *SessionsService) Get(ctx context.Context, sessionID string, opts ...RequestOption) (*Session, error) {
	var out Session
	_, err := s.t.do(ctx, "GET", sessionByID(sessionID), nil, nil, &out, newRequestConfig(opts))
	if err != nil {
		return nil, err
	}
	return &out, nil
}
