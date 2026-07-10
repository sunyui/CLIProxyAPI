# Codex Reasoning Compatibility

## Display sentinel filtering

Some Codex-compatible models include the exact empty HTML comment `<!-- -->` in reasoning output. It is provider display metadata rather than reasoning content. Both supported Claude response paths remove only that exact marker:

- Codex Responses `response.reasoning_summary_text.delta` translated by the Codex-to-Claude translator.
- OpenAI-compatible `reasoning_content` translated by the OpenAI-to-Claude translator.

Both streaming filters handle markers split across upstream deltas. Similar comments, incomplete marker prefixes at the end of a reasoning block, final text, and tool payloads are preserved. Empty deltas are not emitted after filtering, and pending non-marker content is flushed before the block stops. The Codex path also flushes pending content before a thinking signature.

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
go test ./internal/translator/openai/claude -run 'ReasoningDisplaySentinel' -count=1
```

The full repository verification remains:

```bash
go test -p 1 ./...
go build -o test-output ./cmd/server && rm -f test-output
```
