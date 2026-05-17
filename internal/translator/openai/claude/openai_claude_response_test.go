package claude

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/tidwall/gjson"
)

func TestConvertOpenAIResponseToClaude_StreamToolCallIgnoresEmptyNameDeltas(t *testing.T) {
	events := convertOpenAIStreamTestEvents(t, []string{
		`data: {"id":"chatcmpl-test","object":"chat.completion.chunk","created":1,"model":"gpt-test","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"Bash","arguments":""}}]},"finish_reason":null}]}`,
		`data: {"id":"chatcmpl-test","object":"chat.completion.chunk","created":1,"model":"gpt-test","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"name":"","arguments":"{\""}}]},"finish_reason":null}]}`,
		`data: {"id":"chatcmpl-test","object":"chat.completion.chunk","created":1,"model":"gpt-test","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"name":"","arguments":"command"}}]},"finish_reason":null}]}`,
		`data: {"id":"chatcmpl-test","object":"chat.completion.chunk","created":1,"model":"gpt-test","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"name":"","arguments":"\":\"pwd\"}"}}]},"finish_reason":null}]}`,
		`data: {"id":"chatcmpl-test","object":"chat.completion.chunk","created":1,"model":"gpt-test","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`,
		`data: [DONE]`,
	})

	assertSingleBashToolUse(t, events)
	assertToolArgsAndStopReason(t, events)
}

func TestConvertOpenAIResponseToClaude_StreamToolCallSuppressesRepeatedNameStart(t *testing.T) {
	events := convertOpenAIStreamTestEvents(t, []string{
		`data: {"id":"chatcmpl-test","object":"chat.completion.chunk","created":1,"model":"gpt-test","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"Bash","arguments":""}}]},"finish_reason":null}]}`,
		`data: {"id":"chatcmpl-test","object":"chat.completion.chunk","created":1,"model":"gpt-test","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"name":"Bash","arguments":"{\"command\":\"pwd\"}"}}]},"finish_reason":null}]}`,
		`data: {"id":"chatcmpl-test","object":"chat.completion.chunk","created":1,"model":"gpt-test","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`,
		`data: [DONE]`,
	})

	assertSingleBashToolUse(t, events)
	assertToolArgsAndStopReason(t, events)
}

func TestConvertOpenAIResponseToClaude_StreamReadToolCallSanitizesPages(t *testing.T) {
	testCases := []struct {
		name        string
		arguments   string
		wantPages   string
		pagesExists bool
	}{
		{
			name:      "drops empty pages",
			arguments: `{"file_path":"/tmp/example.txt","pages":""}`,
		},
		{
			name:      "drops null pages",
			arguments: `{"file_path":"/tmp/example.txt","pages":null}`,
		},
		{
			name:        "preserves non-empty pages",
			arguments:   `{"file_path":"/tmp/example.pdf","pages":"1-2"}`,
			wantPages:   "1-2",
			pagesExists: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			escapedArgs := strconv.Quote(tc.arguments)
			events := convertOpenAIStreamTestEventsForRequest(t, []byte(`{"stream":true,"tools":[{"name":"Read"}]}`), []string{
				`data: {"id":"chatcmpl-test","object":"chat.completion.chunk","created":1,"model":"gpt-test","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"Read","arguments":` + escapedArgs + `}}]},"finish_reason":null}]}`,
				`data: {"id":"chatcmpl-test","object":"chat.completion.chunk","created":1,"model":"gpt-test","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`,
				`data: [DONE]`,
			})

			args := collectToolInputJSON(t, events)
			parsed := gjson.Parse(args)
			if got := parsed.Get("file_path").String(); got == "" {
				t.Fatalf("file_path missing from tool arguments: %s", args)
			}
			pages := parsed.Get("pages")
			if pages.Exists() != tc.pagesExists {
				t.Fatalf("pages exists = %v, want %v; args=%s", pages.Exists(), tc.pagesExists, args)
			}
			if tc.pagesExists && pages.String() != tc.wantPages {
				t.Fatalf("pages = %q, want %q; args=%s", pages.String(), tc.wantPages, args)
			}
		})
	}
}

func TestConvertOpenAIResponseToClaudeNonStream_ReadToolCallSanitizesPages(t *testing.T) {
	testCases := []struct {
		name string
		raw  []byte
	}{
		{
			name: "drops empty pages",
			raw:  []byte(`{"id":"chatcmpl-test","model":"gpt-test","choices":[{"message":{"tool_calls":[{"id":"call_1","type":"function","function":{"name":"Read","arguments":"{\"file_path\":\"/tmp/example.txt\",\"pages\":\"\"}"}}]},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`),
		},
		{
			name: "drops null pages",
			raw:  []byte(`{"id":"chatcmpl-test","model":"gpt-test","choices":[{"message":{"tool_calls":[{"id":"call_1","type":"function","function":{"name":"Read","arguments":"{\"file_path\":\"/tmp/example.txt\",\"pages\":null}"}}]},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			out := ConvertOpenAIResponseToClaudeNonStream(context.Background(), "gpt-test", []byte(`{"tools":[{"name":"Read"}]}`), nil, tc.raw, nil)
			input := gjson.GetBytes(out, "content.0.input")
			if got := input.Get("file_path").String(); got == "" {
				t.Fatalf("file_path missing from tool input: %s", string(out))
			}
			if pages := input.Get("pages"); pages.Exists() {
				t.Fatalf("pages exists = true, want false; output=%s", string(out))
			}
		})
	}
}

func convertOpenAIStreamTestEvents(t *testing.T, chunks []string) []gjson.Result {
	t.Helper()

	return convertOpenAIStreamTestEventsForRequest(t, []byte(`{"stream":true,"tools":[{"name":"Bash"}]}`), chunks)
}

func convertOpenAIStreamTestEventsForRequest(t *testing.T, originalRequest []byte, chunks []string) []gjson.Result {
	t.Helper()

	var param any
	var events []gjson.Result

	for _, chunk := range chunks {
		out := ConvertOpenAIResponseToClaude(context.Background(), "gpt-test", originalRequest, nil, []byte(chunk), &param)
		for _, raw := range out {
			event := parseSSEDataJSON(t, string(raw))
			events = append(events, event)
		}
	}

	return events
}

func parseSSEDataJSON(t *testing.T, raw string) gjson.Result {
	t.Helper()

	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		parsed := gjson.Parse(payload)
		if !parsed.Exists() {
			t.Fatalf("invalid SSE JSON payload: %q", payload)
		}
		return parsed
	}

	t.Fatalf("missing SSE data line: %q", raw)
	return gjson.Result{}
}

func assertSingleBashToolUse(t *testing.T, events []gjson.Result) {
	t.Helper()

	var toolStarts []gjson.Result
	for _, event := range events {
		if event.Get("type").String() == "content_block_start" && event.Get("content_block.type").String() == "tool_use" {
			toolStarts = append(toolStarts, event)
		}
	}

	if len(toolStarts) != 1 {
		t.Fatalf("tool_use content_block_start count = %d, want 1; starts=%v", len(toolStarts), toolStarts)
	}
	if got := toolStarts[0].Get("content_block.name").String(); got != "Bash" {
		t.Fatalf("tool name = %q, want Bash", got)
	}
}

func collectToolInputJSON(t *testing.T, events []gjson.Result) string {
	t.Helper()

	var gotArgs string
	for _, event := range events {
		if event.Get("type").String() == "content_block_delta" && event.Get("delta.type").String() == "input_json_delta" {
			gotArgs += event.Get("delta.partial_json").String()
		}
	}
	if gotArgs == "" {
		t.Fatalf("missing input_json_delta in events: %v", events)
	}
	return gotArgs
}

func assertToolArgsAndStopReason(t *testing.T, events []gjson.Result) {
	t.Helper()

	gotArgs := collectToolInputJSON(t, events)
	var gotStopReason string
	for _, event := range events {
		if event.Get("type").String() == "message_delta" {
			gotStopReason = event.Get("delta.stop_reason").String()
		}
	}

	if !strings.Contains(gotArgs, `"command":"pwd"`) {
		t.Fatalf("tool arguments = %q, want command pwd", gotArgs)
	}
	if gotStopReason != "tool_use" {
		t.Fatalf("stop_reason = %q, want tool_use", gotStopReason)
	}
}
