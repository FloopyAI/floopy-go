# `floopy-go`

> Official Floopy AI Gateway SDK for Go. **Drop-in wrapper around the
> [`openai-go`](https://github.com/openai/openai-go) SDK** with Floopy's
> cache, audit, experiments, routing, and security on top.

[![Go Reference](https://pkg.go.dev/badge/github.com/FloopyAI/floopy-go.svg)](https://pkg.go.dev/github.com/FloopyAI/floopy-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/FloopyAI/floopy-go)](https://goreportcard.com/report/github.com/FloopyAI/floopy-go)
[![docs](https://img.shields.io/badge/docs-floopy.ai-blue)](https://floopy.ai/docs/guides/sdks/floopy-sdk-go)

## Why

`floopy-go` wraps the official `openai-go` package and points it at the
Floopy gateway, so:

- **Zero migration cost** for `Chat`/`Embeddings`/`Models` — same types,
  same methods, via `client.OpenAI()`.
- **Security updates** to the OpenAI SDK reach you on `go get -u` without
  forks or parity drift.
- **Floopy-only features** (audit, experiments, constraints, decision
  export, feedback, routing dry-run, sessions) get **first-class typed
  methods** instead of hand-rolled `net/http` calls.

## Install

```sh
go get github.com/FloopyAI/floopy-go@latest
```

Requires Go `>= 1.23` (range-over-func iterators).

## Quick start

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/FloopyAI/floopy-go"
	"github.com/openai/openai-go/v3"
)

func main() {
	client, err := floopy.NewClient(os.Getenv("FLOOPY_API_KEY"))
	if err != nil {
		log.Fatal(err)
	}

	resp, err := client.OpenAI().Chat.Completions.New(context.Background(), openai.ChatCompletionNewParams{
		Model: openai.ChatModelGPT4o,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage("Hello from Floopy!"),
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resp.Choices[0].Message.Content)
}
```

A single `*floopy.Client` is safe for concurrent use. Every network call
takes a `context.Context`; there is no separate async client (Go uses
goroutines).

### Migrating from `openai-go`

```diff
- import "github.com/openai/openai-go/v3"
- import "github.com/openai/openai-go/v3/option"
- client := openai.NewClient(option.WithAPIKey(os.Getenv("OPENAI_API_KEY")))
+ import "github.com/FloopyAI/floopy-go"
+ fl, _ := floopy.NewClient(os.Getenv("FLOOPY_API_KEY"))
+ client := fl.OpenAI()

  resp, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{ /* ... */ })
```

`client.OpenAI()` returns a lazily-built `*openai.Client` pre-pointed at the
gateway, so types and runtime behaviour are identical to upstream.

## Floopy options (cache, prompt versioning, security firewall)

```go
client, _ := floopy.NewClient(os.Getenv("FLOOPY_API_KEY"),
	floopy.WithOptions(floopy.Options{
		Cache:              &floopy.CacheOptions{Enabled: floopy.Ptr(true), BucketMaxSize: floopy.Ptr(3)},
		PromptID:           "cd4249d5-44d5-46c8-8961-9eb3861e1f7e",
		PromptVersion:      "1",
		LLMSecurityEnabled: floopy.Ptr(true),
	}),
)
```

These map to `Floopy-*` headers forwarded to **every** request (both
OpenAI-compat calls and Floopy-only ones). Per-call overrides are available
via a trailing `...floopy.RequestOption` on every resource method
(`floopy.WithRequestHeader`, `floopy.WithRequestTimeout`,
`floopy.WithRequestOptions`).

| Option | Header | Purpose |
| --- | --- | --- |
| `Cache.Enabled` | `Floopy-Cache-Enabled` | Toggle exact + semantic cache |
| `Cache.BucketMaxSize` | `Floopy-Cache-Bucket-Max-Size` | Max entries per semantic bucket |
| `PromptID` | `Floopy-Prompt-Id` | Stored prompt to resolve |
| `PromptVersion` | `Floopy-Prompt-Version` | Pinned version for `PromptID` |
| `LLMSecurityEnabled` | `floopy-llm-security-enabled` | LLM firewall pre-check |

## Floopy-only resources

Each resource maps to a public `/v1/*` gateway endpoint and is typed
end-to-end. Errors are `*floopy.FloopyError` subtypes (see below).

### `Feedback`

```go
resp, _ := client.OpenAI().Chat.Completions.New(ctx, params)
client.Feedback.Submit(ctx, floopy.FeedbackSubmitParams{Score: 9, Useful: true, SessionID: resp.ID})
```

### `Decisions`

```go
d, _ := client.Decisions.Get(ctx, requestID)
page, _ := client.Decisions.List(ctx, floopy.DecisionListParams{From: since, Limit: 50})

for d, err := range client.Decisions.Iterate(ctx, floopy.DecisionListParams{From: since}) { /* one at a time */ }
for p, err := range client.Decisions.Pages(ctx, floopy.DecisionListParams{From: since})   { /* page at a time */ }
```

### `Experiments`

```go
exp, _ := client.Experiments.Create(ctx, floopy.ExperimentCreateParams{
	Name:                  "cost-vs-quality",
	VariantARoutingRuleID: ruleA,
	VariantBRoutingRuleID: ruleB,
})
results, _ := client.Experiments.Results(ctx, exp.ID)
client.Experiments.Rollback(ctx, exp.ID)
```

`Create` and `Rollback` automatically include the `X-Floopy-Confirm:
experiments` header the gateway requires (SEC-009).

### `Constraints`

```go
current, _ := client.Constraints.Get(ctx)
client.Constraints.Put(ctx, floopy.OrgConstraints{CostLimitMonthlyUSD: floopy.Ptr(100.0)})
```

`Put` is full-replace: any `nil` field is reset server-side.

### `Export`

```go
for row, err := range client.Export.Decisions(ctx, floopy.ExportDecisionsParams{From: start, To: end}) {
	// streamed JSONL, parsed and typed
}

// to also read the trailer (truncation reasons, totals):
stream := client.Export.DecisionsWithTrailer(ctx, floopy.ExportDecisionsParams{From: start, To: end})
for row, err := range stream.Rows { /* ... */ }
fmt.Println(stream.Trailer()) // populated after iteration completes
```

### `Evaluations`

```go
run, _ := client.Evaluations.Create(ctx, floopy.EvaluationCreateParams{DatasetID: dsID, Model: "gpt-4o"})
status, _ := client.Evaluations.Get(ctx, run.ID)
results, _ := client.Evaluations.Results(ctx, run.ID, 100, "")
client.Evaluations.Cancel(ctx, run.ID)
```

### `Routing.Explain`

```go
explain, _ := client.Routing.Explain(ctx, floopy.RoutingExplainParams{Model: "gpt-4o", Messages: messages})
fmt.Println(explain.WouldSelect, explain.FirewallDecision)
```

Pro plan only. `WouldSelect` is `nil` if the firewall blocks the request.

### `Sessions`

```go
session, _ := client.Sessions.Get(ctx, sessionID)
// session.Messages is a drop-in for a follow-up chat completion
client.OpenAI().Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
	Model: openai.ChatModelGPT4o, Messages: session.Messages,
})
```

## Streaming

`Chat` streaming is delegated to `openai-go`
(`client.OpenAI().Chat.Completions.NewStreaming(...)`). `Export.Decisions`
is a range-over-func iterator over the gateway's JSONL stream and skips the
trailer record.

## Error handling

Every Floopy-only call returns a `*floopy.FloopyError` subtype:

```go
import "errors"

_, err := client.Decisions.List(ctx, params)
var rl *floopy.RateLimitError
var plan *floopy.PlanError
switch {
case errors.As(err, &rl):
	time.Sleep(time.Duration(max(rl.RetryAfterSeconds, 1)) * time.Second)
case errors.As(err, &plan):
	log.Printf("upgrade plan: feature %s not in current plan", plan.Feature)
}
```

`floopy.AsError(err)` recovers the base `*floopy.FloopyError` from any
Floopy error. Errors from `Chat`/`Embeddings` are emitted by `openai-go`
(`*openai.Error`), not this package.

## Security

- The API key is only ever sent in the `Authorization` header; the SDK
  never logs request or response bodies.
- TLS certificate verification is on by default (`net/http`).
- Releases are immutable Git tags verified by the Go module checksum
  database (`sum.golang.org`) — no registry account or long-lived token is
  involved in publishing.

## Self-hosting / custom base URL

```go
client, _ := floopy.NewClient(os.Getenv("FLOOPY_API_KEY"),
	floopy.WithBaseURL("https://gateway.internal.acme.com/v1"))
```

## Links

- Full SDK guide: <https://floopy.ai/docs/guides/sdks/floopy-sdk-go>
  ([Português](https://floopy.ai/pt/docs/guides/sdks/floopy-sdk-go))
- Go reference: <https://pkg.go.dev/github.com/FloopyAI/floopy-go>
- API reference: <https://floopy.ai/docs/guides/api-reference>
- Changelog: [`CHANGELOG.md`](./CHANGELOG.md)

## License

Apache-2.0 © Floopy
