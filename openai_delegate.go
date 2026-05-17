package floopy

import (
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// newOpenAIDelegate builds an openai-go client pre-pointed at the Floopy
// gateway. Mirrors src/openai-delegate.ts (Node) and _openai_delegate.py
// (Python): openai-go manages auth, content-type and user-agent itself, so
// those headers are stripped from the forwarded Floopy header set; the
// remaining Floopy-* toggles ride along on every OpenAI-compatible call.
func newOpenAIDelegate(t *transport) *openai.Client {
	headers := mergeHeaders(buildFloopyHeaders(nil), t.defaultRequestHeaders())
	delete(headers, headerAuthorization)
	delete(headers, headerContentType)
	delete(headers, headerUserAgent)

	opts := []option.RequestOption{
		option.WithAPIKey(t.apiKey),
		option.WithBaseURL(t.baseURL),
		option.WithHTTPClient(t.httpClient),
		option.WithHeader(headerFloopySDK, userAgentPrefix+"/"+Version),
	}
	for k, v := range headers {
		opts = append(opts, option.WithHeader(k, v))
	}
	c := openai.NewClient(opts...)
	return &c
}
