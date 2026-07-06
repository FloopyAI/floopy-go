package floopy

import (
	"context"

	"github.com/openai/openai-go/v3"
)

// FirewallDecision is the LLM-firewall verdict for a routing dry-run.
type FirewallDecision = string

const (
	FirewallAllow      FirewallDecision = "allow"
	FirewallBlockInput FirewallDecision = "block_input"
)

// RoutingExplainResult is the outcome of a routing dry-run.
type RoutingExplainResult struct {
	// WouldSelect is the provider/model the gateway would route to, or nil
	// when the firewall blocks the request.
	WouldSelect      map[string]string `json:"would_select"`
	FirewallDecision FirewallDecision  `json:"firewall_decision"`
	Reasoning        *string           `json:"reasoning"`
	RoutingRuleID    *string           `json:"routing_rule_id"`
}

// RoutingExplainParams are the arguments for RoutingService.Explain. Messages
// uses the same openai-go type as Chat completions, so a request can be
// dry-run before it is sent.
type RoutingExplainParams struct {
	Model       string
	Messages    []openai.ChatCompletionMessageParamUnion
	Temperature *float64
	// MaxTokens is the legacy output-token cap. Prefer MaxCompletionTokens;
	// the gateway coerces this into max_completion_tokens before forwarding
	// to any OpenAI-compatible provider.
	MaxTokens *int
	// MaxCompletionTokens is the canonical output-token cap (the current
	// OpenAI standard). Takes precedence over MaxTokens when both are set.
	MaxCompletionTokens *int
	TopP                *float64
}

type routingExplainBody struct {
	Model               string                                   `json:"model"`
	Messages            []openai.ChatCompletionMessageParamUnion `json:"messages"`
	Temperature         *float64                                 `json:"temperature,omitempty"`
	MaxTokens           *int                                     `json:"max_tokens,omitempty"`
	MaxCompletionTokens *int                                     `json:"max_completion_tokens,omitempty"`
	TopP                *float64                                 `json:"top_p,omitempty"`
}

// RoutingService exposes the routing dry-run.
type RoutingService struct{ t *transport }

// Explain runs the router and firewall without calling a provider (Pro
// plan). WouldSelect is nil when the firewall would block the request.
func (s *RoutingService) Explain(ctx context.Context, params RoutingExplainParams, opts ...RequestOption) (*RoutingExplainResult, error) {
	body := routingExplainBody(params)
	var out RoutingExplainResult
	_, err := s.t.do(ctx, "POST", endpointRoutingExplain, body, nil, &out, newRequestConfig(opts))
	if err != nil {
		return nil, err
	}
	return &out, nil
}
