# Changelog

All notable changes to `floopy-go` are documented in this file. The format
follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the
project adheres to [Semantic Versioning](https://semver.org/).

Releases are produced by `release-please` from Conventional Commits.

## [0.2.0](https://github.com/FloopyAI/floopy-go/compare/floopy-go-v0.1.0...floopy-go-v0.2.0) (2026-05-17)


### Added

* publish go sdk ([6703352](https://github.com/FloopyAI/floopy-go/commit/6703352dcf26fef0302b6e45d1a3d6a8f3266dd5))

## [Unreleased]

### Added

- Initial Go SDK: a single concurrency-safe `Client` wrapping the official
  `openai-go/v3` client via a lazy `OpenAI()` delegate, typed `Options`
  mapped to `Floopy-*` headers, an internal `net/http` transport with
  retries/backoff/timeouts honouring `Retry-After`, and the `FloopyError`
  hierarchy. `Chat`, `Embeddings`, and `Models` reach the gateway 1:1 with
  the upstream `openai-go` SDK.
- Floopy-only resources: `Feedback`, `Decisions` (+ range-over-func
  `Pages`/`Iterate`), `Experiments` (with auto `X-Floopy-Confirm`),
  `Constraints`, `Export` (JSONL streaming with optional trailer capture),
  `Evaluations`, `Routing.Explain`, and `Sessions.Get`. Each resource is
  fully typed end-to-end and returns the appropriate `FloopyError` subtype.
