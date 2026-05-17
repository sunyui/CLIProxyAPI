package executor

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
	cliproxyauth "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/executor"
	sdktranslator "github.com/router-for-me/CLIProxyAPI/v6/sdk/translator"
	"github.com/tidwall/gjson"
)

func TestOpenAICompatExecutorCompactPassthrough(t *testing.T) {
	var gotPath string
	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		gotBody = body
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_1","object":"response.compaction","usage":{"input_tokens":1,"output_tokens":2,"total_tokens":3}}`))
	}))
	defer server.Close()

	executor := NewOpenAICompatExecutor("openai-compatibility", &config.Config{})
	auth := &cliproxyauth.Auth{Attributes: map[string]string{
		"base_url": server.URL + "/v1",
		"api_key":  "test",
	}}
	payload := []byte(`{"model":"gpt-5.1-codex-max","input":[{"role":"user","content":"hi"}]}`)
	resp, err := executor.Execute(context.Background(), auth, cliproxyexecutor.Request{
		Model:   "gpt-5.1-codex-max",
		Payload: payload,
	}, cliproxyexecutor.Options{
		SourceFormat: sdktranslator.FromString("openai-response"),
		Alt:          "responses/compact",
		Stream:       false,
	})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if gotPath != "/v1/responses/compact" {
		t.Fatalf("path = %q, want %q", gotPath, "/v1/responses/compact")
	}
	if !gjson.GetBytes(gotBody, "input").Exists() {
		t.Fatalf("expected input in body")
	}
	if gjson.GetBytes(gotBody, "messages").Exists() {
		t.Fatalf("unexpected messages in body")
	}
	if string(resp.Payload) != `{"id":"resp_1","object":"response.compaction","usage":{"input_tokens":1,"output_tokens":2,"total_tokens":3}}` {
		t.Fatalf("payload = %s", string(resp.Payload))
	}
}

func TestOpenAICompatExecutorStreamSSEWhitespaceAndMetadata(t *testing.T) {
	stream := executeOpenAICompatStreamTest(t, "   data: {\"chunk\":1}\n:event comment\nevent: ping\nid: 1\nretry: 1000\n\ndata: [DONE]\n")
	var payloads []string
	for chunk := range stream.Chunks {
		if chunk.Err != nil {
			t.Fatalf("unexpected stream error: %v", chunk.Err)
		}
		payloads = append(payloads, string(chunk.Payload))
	}
	joined := strings.Join(payloads, "\n")
	if !strings.Contains(joined, `{"chunk":1}`) {
		t.Fatalf("translated payloads missing trimmed data chunk: %q", joined)
	}
	if strings.Contains(joined, "event: ping") || strings.Contains(joined, "retry: 1000") {
		t.Fatalf("metadata lines should not be forwarded: %q", joined)
	}
}

func TestOpenAICompatExecutorStreamJSONErrorBody(t *testing.T) {
	stream := executeOpenAICompatStreamTest(t, `{"error":"upstream failed"}`+"\n")
	chunk, ok := <-stream.Chunks
	if !ok {
		t.Fatal("missing stream error chunk")
	}
	if chunk.Err == nil {
		t.Fatalf("expected stream error, got payload %q", string(chunk.Payload))
	}
	statusProvider, ok := chunk.Err.(interface{ StatusCode() int })
	if !ok {
		t.Fatalf("stream error does not expose status: %T", chunk.Err)
	}
	if got := statusProvider.StatusCode(); got != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d", got, http.StatusBadGateway)
	}
	if !strings.Contains(chunk.Err.Error(), "upstream failed") {
		t.Fatalf("error = %q, want upstream body", chunk.Err.Error())
	}
	if extra, ok := <-stream.Chunks; ok {
		t.Fatalf("unexpected extra chunk after error: %+v", extra)
	}
}

func executeOpenAICompatStreamTest(t *testing.T, responseBody string) *cliproxyexecutor.StreamResult {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(responseBody))
	}))
	t.Cleanup(server.Close)

	executor := NewOpenAICompatExecutor("openai-compatibility", &config.Config{})
	auth := &cliproxyauth.Auth{Attributes: map[string]string{
		"base_url": server.URL + "/v1",
		"api_key":  "test",
	}}
	stream, err := executor.ExecuteStream(context.Background(), auth, cliproxyexecutor.Request{
		Model:   "gpt-test",
		Payload: []byte(`{"model":"gpt-test","messages":[{"role":"user","content":"hi"}],"stream":true}`),
	}, cliproxyexecutor.Options{
		SourceFormat: sdktranslator.FormatOpenAI,
		Stream:       true,
	})
	if err != nil {
		t.Fatalf("ExecuteStream error: %v", err)
	}
	return stream
}
