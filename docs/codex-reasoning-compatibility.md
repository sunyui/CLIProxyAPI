# Codex Reasoning Compatibility

## Display sentinel filtering

Some Codex-compatible models include the exact empty HTML comment `<!-- -->` in `response.reasoning_summary_text.delta`. It is provider display metadata rather than reasoning content. The Codex-to-Claude streaming translator removes only that exact marker before emitting Claude `thinking_delta` events.

The filter is streaming-safe when the marker spans multiple upstream deltas. Similar comments, incomplete marker prefixes at the end of a reasoning block, final text, and tool payloads are preserved. Empty deltas are not emitted after filtering, and pending non-marker content is flushed before the thinking signature and block stop.

## Preserved team compatibility

The upstream merge keeps these team-specific behaviors:

- Claude-to-OpenAI `cache_control` metadata propagation.
- Removal of `Read.pages` for non-PDF files while preserving PDF page ranges.
- Removal of an empty `EnterWorktree.name` when `path` is present.
- OpenAI-compatible SSE whitespace, metadata, keepalive, and plain JSON error handling.
- Antigravity reasoning replay without sparse `null` parts.
- The Codex direct image edit path `/images/edit`; the public API remains `/v1/images/edits`.

## Verification

Run the focused translator regression suite:

```bash
go test ./internal/translator/codex/claude -run 'Sentinel|ReasoningSummaryMarker|Thinking|FunctionCall' -count=1
```

The full repository verification remains:

```bash
go test -p 1 ./...
go build -o test-output ./cmd/server && rm -f test-output
```
