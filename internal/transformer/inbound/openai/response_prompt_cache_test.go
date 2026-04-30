package openai

import (
	"context"
	"encoding/json"
	"testing"
)

func TestResponseInboundPreservesResponsesOnlyFields(t *testing.T) {
	body := []byte(`{
		"model": "gpt-4.1",
		"input": "hello",
		"prompt_cache_key": "tenant-a-thread-1",
		"prompt_cache_retention": "in-memory",
		"safety_identifier": "user-hash-1",
		"previous_response_id": "resp_123",
		"truncation": "auto",
		"extra_body": {
			"caching": {"type": "enabled"},
			"expire_at": 1893456000,
			"max_tool_calls": 4,
			"context_management": {"edits": [{"type": "clear_thinking", "keep": {"type": "thinking_turns", "value": 1}}]}
		},
		"include": ["reasoning.encrypted_content"],
		"top_logprobs": 3,
		"reasoning": {
			"effort": "medium",
			"max_tokens": 256
		}
	}`)

	req, err := (&ResponseInbound{}).TransformRequest(context.Background(), body)
	if err != nil {
		t.Fatalf("TransformRequest failed: %v", err)
	}

	if req.PromptCacheKey == nil || *req.PromptCacheKey != "tenant-a-thread-1" {
		t.Fatalf("prompt_cache_key was not preserved: %v", req.PromptCacheKey)
	}
	if req.PromptCacheRetention == nil || *req.PromptCacheRetention != "in-memory" {
		t.Fatalf("prompt_cache_retention was not preserved: %v", req.PromptCacheRetention)
	}
	if req.SafetyIdentifier == nil || *req.SafetyIdentifier != "user-hash-1" {
		t.Fatalf("safety_identifier was not preserved: %v", req.SafetyIdentifier)
	}
	if req.PreviousResponseID == nil || *req.PreviousResponseID != "resp_123" {
		t.Fatalf("previous_response_id was not preserved: %v", req.PreviousResponseID)
	}
	if req.Truncation == nil || *req.Truncation != "auto" {
		t.Fatalf("truncation was not preserved: %v", req.Truncation)
	}
	var extraBody map[string]any
	if err := json.Unmarshal(req.ExtraBody, &extraBody); err != nil {
		t.Fatalf("extra_body was not preserved as JSON object: %v", err)
	}
	for _, key := range []string{"caching", "expire_at", "max_tool_calls", "context_management"} {
		if _, ok := extraBody[key]; !ok {
			t.Fatalf("extra_body field %q was not preserved: %#v", key, extraBody)
		}
	}
	if len(req.Include) != 1 || req.Include[0] != "reasoning.encrypted_content" {
		t.Fatalf("include was not preserved: %#v", req.Include)
	}
	if req.TopLogprobs == nil || *req.TopLogprobs != 3 {
		t.Fatalf("top_logprobs was not preserved: %v", req.TopLogprobs)
	}
	if req.ReasoningEffort != "medium" {
		t.Fatalf("reasoning.effort = %q, want medium", req.ReasoningEffort)
	}
	if req.ReasoningBudget == nil || *req.ReasoningBudget != 256 {
		t.Fatalf("reasoning.max_tokens was not preserved: %v", req.ReasoningBudget)
	}
}
