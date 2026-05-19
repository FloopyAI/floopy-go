package floopy

import (
	"sync"

	"github.com/openai/openai-go/v3"
)

// Client is the Floopy gateway client. It wraps the official openai-go client
// (reachable via OpenAI) and exposes typed Floopy-only resources. A *Client
// is safe for concurrent use by multiple goroutines.
type Client struct {
	transport *transport

	openaiOnce   sync.Once
	openaiClient *openai.Client

	// Feedback submits NPS-style feedback for a request/session.
	Feedback *FeedbackService
	// Decisions reads the per-request decision audit trail.
	Decisions *DecisionsService
	// Experiments manages A/B routing experiments.
	Experiments *ExperimentsService
	// Constraints reads and full-replaces org spend/rate constraints.
	Constraints *ConstraintsService
	// Export streams the decision log as typed JSONL.
	Export *ExportService
	// Evaluations runs and inspects dataset evaluations.
	Evaluations *EvaluationsService
	// Routing exposes the routing dry-run (Pro plan).
	Routing *RoutingService
	// Sessions restores stored conversations.
	Sessions *SessionsService
	// Files manages Batch API input/output files.
	Files *FilesService
	// Batches manages asynchronous batch jobs.
	Batches *BatchesService
}

// NewClient constructs a Floopy client. apiKey is required (starts with
// "fl_"); pass options to override the base URL, timeout, retries, default
// headers, Floopy options, or the HTTP client.
//
//	client, err := floopy.NewClient(os.Getenv("FLOOPY_API_KEY"),
//		floopy.WithOptions(floopy.Options{
//			Cache: &floopy.CacheOptions{Enabled: floopy.Ptr(true)},
//		}),
//	)
func NewClient(apiKey string, opts ...ClientOption) (*Client, error) {
	cfg := &clientConfig{
		baseURL:    DefaultBaseURL,
		timeout:    DefaultTimeout,
		maxRetries: DefaultMaxRetries,
	}
	for _, o := range opts {
		o(cfg)
	}
	tr, err := newTransport(apiKey, cfg)
	if err != nil {
		return nil, err
	}
	c := &Client{transport: tr}
	c.Feedback = &FeedbackService{tr}
	c.Decisions = &DecisionsService{tr}
	c.Experiments = &ExperimentsService{tr}
	c.Constraints = &ConstraintsService{tr}
	c.Export = &ExportService{tr}
	c.Evaluations = &EvaluationsService{tr}
	c.Routing = &RoutingService{tr}
	c.Sessions = &SessionsService{tr}
	c.Files = &FilesService{tr}
	c.Batches = &BatchesService{tr}
	return c, nil
}

// OpenAI returns a lazily-instantiated openai-go client pre-configured to
// talk to the Floopy gateway. client.OpenAI().Chat.Completions.New(...) and
// client.OpenAI().Embeddings.New(...) are 1:1 drop-in replacements for the
// upstream openai-go package. The client is built once and reused.
func (c *Client) OpenAI() *openai.Client {
	c.openaiOnce.Do(func() {
		c.openaiClient = newOpenAIDelegate(c.transport)
	})
	return c.openaiClient
}

// BaseURL returns the resolved gateway base URL.
func (c *Client) BaseURL() string { return c.transport.baseURL }
