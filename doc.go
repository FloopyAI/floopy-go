// Package floopy is the official Floopy AI Gateway SDK for Go.
//
// It wraps the official [github.com/openai/openai-go/v3] client and points it
// at the Floopy gateway, so Chat/Embeddings/Models stay a 1:1 drop-in
// replacement, and adds typed Floopy-only resources (feedback, decisions,
// experiments, constraints, decision export, evaluations, routing dry-run,
// sessions) on top. It mirrors the Node (floopy-sdk) and Python (floopy-sdk)
// SDKs so behaviour stays in lockstep across languages.
//
//	client, err := floopy.NewClient(os.Getenv("FLOOPY_API_KEY"))
//	if err != nil {
//		log.Fatal(err)
//	}
//	resp, err := client.OpenAI().Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
//		Model:    openai.ChatModelGPT4o,
//		Messages: []openai.ChatCompletionMessageParamUnion{
//			openai.UserMessage("Hello from Floopy!"),
//		},
//	})
//
// The SDK is concurrency-safe: a single *Client may be shared across
// goroutines. Every network call takes a context.Context for cancellation
// and deadlines; there is no separate async client (Go uses goroutines).
package floopy
