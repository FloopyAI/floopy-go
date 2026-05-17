# `floopy-go` examples

Runnable snippets for every public surface of the SDK. This is a **separate
Go module** (with a `replace` onto the parent) so `go get
github.com/FloopyAI/floopy-go` never pulls example-only dependencies.

## Setup

```sh
# from floopy-go/examples
cp .env.example .env        # add your FLOOPY_API_KEY
set -a; source .env; set +a
go run ./chat
```

By default examples talk to `https://api.floopy.ai/v1`. To point at a local
gateway, set `FLOOPY_BASE_URL=http://localhost:8000/v1`.

## Files

| Directory | What it shows |
| --- | --- |
| `chat` | Basic chat completion (drop-in for `openai-go`) |
| `chat_stream` | Streaming response |
| `embeddings` | Batch embeddings |
| `feedback` | Submit NPS-style feedback |
| `decisions_list` | List + paginate decisions (range-over-func iterators) |
| `export_decisions` | Stream the JSONL export + trailer |
| `experiments_create` | Create + roll back an experiment |
| `constraints` | Read + full-replace org constraints |
| `routing_explain` | Routing dry-run (Pro plan) |
