package thinking_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/registry"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/thinking"
	_ "github.com/router-for-me/CLIProxyAPI/v7/internal/thinking/provider/claude"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

func TestApplyThinking_UserDefinedClaudePreservesAdaptiveLevel(t *testing.T) {
	reg := registry.GetGlobalRegistry()
	clientID := "test-user-defined-claude-" + t.Name()
	modelID := "custom-claude-4-6"
	reg.RegisterClient(clientID, "claude", []*registry.ModelInfo{{ID: modelID, UserDefined: true}})
	t.Cleanup(func() {
		reg.UnregisterClient(clientID)
	})

	tests := []struct {
		name  string
		model string
		body  []byte
	}{
		{
			name:  "claude adaptive effort body",
			model: modelID,
			body:  []byte(`{"thinking":{"type":"adaptive"},"output_config":{"effort":"high"}}`),
		},
		{
			name:  "suffix level",
			model: modelID + "(high)",
			body:  []byte(`{}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := thinking.ApplyThinking(tt.body, tt.model, "openai", "claude", "claude")
			if err != nil {
				t.Fatalf("ApplyThinking() error = %v", err)
			}
			if got := gjson.GetBytes(out, "thinking.type").String(); got != "adaptive" {
				t.Fatalf("thinking.type = %q, want %q, body=%s", got, "adaptive", string(out))
			}
			if got := gjson.GetBytes(out, "output_config.effort").String(); got != "high" {
				t.Fatalf("output_config.effort = %q, want %q, body=%s", got, "high", string(out))
			}
			if gjson.GetBytes(out, "thinking.budget_tokens").Exists() {
				t.Fatalf("thinking.budget_tokens should be removed, body=%s", string(out))
			}
		})
	}
}

func TestValidateConfig_LevelOnlyBudgetZeroDoesNotWarn(t *testing.T) {
	var logs bytes.Buffer
	oldOut := log.StandardLogger().Out
	oldLevel := log.GetLevel()
	log.SetOutput(&logs)
	log.SetLevel(log.WarnLevel)
	t.Cleanup(func() {
		log.SetOutput(oldOut)
		log.SetLevel(oldLevel)
	})

	model := &registry.ModelInfo{
		ID: "gpt-5.5",
		Thinking: &registry.ThinkingSupport{
			Levels: []string{"low", "medium", "high", "xhigh"},
		},
	}
	got, err := thinking.ValidateConfig(thinking.ThinkingConfig{Mode: thinking.ModeNone, Budget: 0}, model, "openai", "openai", false)
	if err != nil {
		t.Fatalf("ValidateConfig() error = %v", err)
	}
	if got == nil {
		t.Fatal("ValidateConfig() returned nil config")
	}
	if got.Budget != 0 {
		t.Fatalf("budget = %d, want 0", got.Budget)
	}
	if strings.Contains(logs.String(), "budget zero not allowed") {
		t.Fatalf("unexpected warning log: %s", logs.String())
	}
}
