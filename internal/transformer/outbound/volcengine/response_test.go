package volcengine

import (
	"context"
	"encoding/json"
	"io"
	"slices"
	"testing"

	"github.com/bestruirui/octopus/internal/transformer/model"
	"github.com/bestruirui/octopus/internal/transformer/outbound/openai"
)

func TestConvertToResponsesInputHandlesEmptyItems(t *testing.T) {
	got := convertToResponsesInput(openai.ResponsesInput{})

	if got.Text != nil {
		t.Fatalf("expected text input to stay nil, got %q", *got.Text)
	}
	if got.Items == nil {
		t.Fatal("expected empty item slice, got nil")
	}
	if len(got.Items) != 0 {
		t.Fatalf("expected no items, got %#v", got.Items)
	}
}

func TestResponseOutboundHandlesSystemOnlyMessages(t *testing.T) {
	content := "only system instructions"
	req := &model.InternalLLMRequest{
		Model: "doubao-seed-1-6-lite-251015",
		Messages: []model.Message{
			{
				Role: "system",
				Content: model.MessageContent{
					Content: &content,
				},
			},
		},
	}

	httpReq, err := (&ResponseOutbound{}).TransformRequest(context.Background(), req, "https://ark.example.com/api/v3", "test-key")
	if err != nil {
		t.Fatalf("TransformRequest returned error: %v", err)
	}

	body, err := io.ReadAll(httpReq.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}

	input, ok := payload["input"].([]any)
	if !ok {
		t.Fatalf("input = %#v, want empty array", payload["input"])
	}
	if len(input) != 0 {
		t.Fatalf("input = %#v, want empty array", input)
	}
}

func TestResponseOutboundStripsOpenAIOnlyFields(t *testing.T) {
	content := "hello"
	parallelToolCalls := true
	serviceTier := "auto"
	user := "user-1"
	safetyIdentifier := "user-hash-1"
	promptCacheKey := "tenant-a-thread-1"
	promptCacheRetention := "in-memory"
	previousResponseID := "resp_123"
	truncation := "auto"
	temperature := 0.2
	topP := 0.7
	topLogprobs := int64(3)
	maxOutputTokens := int64(64)
	reasoningBudget := int64(256)
	req := &model.InternalLLMRequest{
		Model: "doubao-seed-1-6-lite-251015",
		Messages: []model.Message{
			{
				Role: "user",
				Content: model.MessageContent{
					Content: &content,
				},
			},
		},
		ParallelToolCalls:    &parallelToolCalls,
		ServiceTier:          &serviceTier,
		User:                 &user,
		SafetyIdentifier:     &safetyIdentifier,
		PromptCacheKey:       &promptCacheKey,
		PromptCacheRetention: &promptCacheRetention,
		PreviousResponseID:   &previousResponseID,
		Truncation:           &truncation,
		Metadata:             map[string]string{"trace": "demo"},
		MaxCompletionTokens:  &maxOutputTokens,
		Temperature:          &temperature,
		TopP:                 &topP,
		TopLogprobs:          &topLogprobs,
		Include:              []string{"reasoning.encrypted_content"},
		ReasoningEffort:      "medium",
		ReasoningBudget:      &reasoningBudget,
		ExtraBody: json.RawMessage(`{
			"caching": {"type": "enabled"},
			"expire_at": 1893456000,
			"max_tool_calls": 4,
			"context_management": {
				"edits": [
					{"type": "clear_thinking", "keep": {"type": "thinking_turns", "value": 1}}
				]
			}
		}`),
	}

	httpReq, err := (&ResponseOutbound{}).TransformRequest(context.Background(), req, "https://ark.example.com/api/v3", "test-key")
	if err != nil {
		t.Fatalf("TransformRequest returned error: %v", err)
	}

	body, err := io.ReadAll(httpReq.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}

	for _, key := range []string{
		"parallel_tool_calls",
		"service_tier",
		"user",
		"safety_identifier",
		"prompt_cache_key",
		"prompt_cache_retention",
		"truncation",
		"metadata",
		"top_logprobs",
		"include",
	} {
		if _, ok := payload[key]; ok {
			t.Fatalf("payload should strip %q, got body %s", key, string(body))
		}
	}

	for _, key := range []string{
		"model",
		"input",
		"previous_response_id",
		"max_output_tokens",
		"temperature",
		"top_p",
		"reasoning",
		"thinking",
		"caching",
		"expire_at",
		"max_tool_calls",
		"context_management",
	} {
		if _, ok := payload[key]; !ok {
			t.Fatalf("payload should preserve supported field %q, got body %s", key, string(body))
		}
	}

	if got := payload["previous_response_id"]; got != previousResponseID {
		t.Fatalf("previous_response_id = %#v, want %q", got, previousResponseID)
	}
	reasoning, ok := payload["reasoning"].(map[string]any)
	if !ok {
		t.Fatalf("reasoning = %#v, want object", payload["reasoning"])
	}
	if got := reasoning["effort"]; got != "medium" {
		t.Fatalf("reasoning.effort = %#v, want medium", got)
	}
	if _, ok := reasoning["max_tokens"]; ok {
		t.Fatalf("reasoning.max_tokens should be stripped, got %#v", reasoning)
	}
	caching, ok := payload["caching"].(map[string]any)
	if !ok || caching["type"] != "enabled" {
		t.Fatalf("caching = %#v, want enabled object", payload["caching"])
	}
	if got := payload["expire_at"]; got != float64(1893456000) {
		t.Fatalf("expire_at = %#v, want 1893456000", got)
	}
	if got := payload["max_tool_calls"]; got != float64(4) {
		t.Fatalf("max_tool_calls = %#v, want 4", got)
	}
	contextManagement, ok := payload["context_management"].(map[string]any)
	if !ok || contextManagement["edits"] == nil {
		t.Fatalf("context_management = %#v, want edits object", payload["context_management"])
	}

	keys := make([]string, 0, len(payload))
	for key := range payload {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	t.Logf("volcengine request keys: %v", keys)
}

func TestResponseOutboundRejectsUnsupportedExtraBodyField(t *testing.T) {
	content := "hello"
	req := &model.InternalLLMRequest{
		Model: "doubao-seed-1-6-lite-251015",
		Messages: []model.Message{
			{
				Role: "user",
				Content: model.MessageContent{
					Content: &content,
				},
			},
		},
		ExtraBody: json.RawMessage(`{"prompt_cache_key": "should-not-pass-through"}`),
	}

	_, err := (&ResponseOutbound{}).TransformRequest(context.Background(), req, "https://ark.example.com/api/v3", "test-key")
	if err == nil {
		t.Fatal("expected unsupported extra_body field to fail")
	}
	if err.Error() != `failed to marshal responses api request: extra_body field "prompt_cache_key" is not allowed for volcengine responses` {
		t.Fatalf("unexpected error: %v", err)
	}
}
