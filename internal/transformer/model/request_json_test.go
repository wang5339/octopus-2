package model

import (
	"encoding/json"
	"testing"
)

func TestInternalLLMRequestPromptCacheKeyIsString(t *testing.T) {
	raw := []byte(`{
		"model": "gpt-4.1",
		"messages": [{"role": "user", "content": "hello"}],
		"prompt_cache_key": "tenant-a-thread-1"
	}`)

	var req InternalLLMRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		t.Fatalf("prompt_cache_key string should unmarshal: %v", err)
	}

	if req.PromptCacheKey == nil || *req.PromptCacheKey != "tenant-a-thread-1" {
		t.Fatalf("unexpected prompt_cache_key: %v", req.PromptCacheKey)
	}

	encoded, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(encoded, &payload); err != nil {
		t.Fatalf("unmarshal encoded request: %v", err)
	}
	if got, ok := payload["prompt_cache_key"].(string); !ok || got != "tenant-a-thread-1" {
		t.Fatalf("encoded prompt_cache_key = %#v, want string tenant-a-thread-1", payload["prompt_cache_key"])
	}
}

func TestInternalLLMRequestClearHelpFieldsStripsResponsesOnlyFields(t *testing.T) {
	content := "hello"
	extraBody := json.RawMessage(`{"caching":{"type":"enabled"}}`)
	promptCacheRetention := "in-memory"
	previousResponseID := "resp_123"
	truncation := "auto"
	req := &InternalLLMRequest{
		Model:                "gpt-4.1",
		Messages:             []Message{{Role: "user", Content: MessageContent{Content: &content}}},
		ExtraBody:            extraBody,
		PromptCacheRetention: &promptCacheRetention,
		Include:              []string{"reasoning.encrypted_content"},
		PreviousResponseID:   &previousResponseID,
		Truncation:           &truncation,
	}

	req.ClearHelpFields()

	if req.ExtraBody != nil {
		t.Fatalf("ExtraBody should be cleared: %s", req.ExtraBody)
	}
	if req.PromptCacheRetention != nil {
		t.Fatalf("PromptCacheRetention should be cleared: %v", req.PromptCacheRetention)
	}
	if req.Include != nil {
		t.Fatalf("Include should be cleared: %#v", req.Include)
	}
	if req.PreviousResponseID != nil {
		t.Fatalf("PreviousResponseID should be cleared: %v", req.PreviousResponseID)
	}
	if req.Truncation != nil {
		t.Fatalf("Truncation should be cleared: %v", req.Truncation)
	}
}
