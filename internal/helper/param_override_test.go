package helper

import (
	"encoding/json"
	"net/url"
	"testing"

	transformerModel "github.com/bestruirui/octopus/internal/transformer/model"
)

func TestApplyParamOverride(t *testing.T) {
	content := "hello"
	temperature := 1.0
	maxTokens := int64(16)
	req := &transformerModel.InternalLLMRequest{
		Model:       "gpt-4",
		Messages:    []transformerModel.Message{{Role: "user", Content: transformerModel.MessageContent{Content: &content}}},
		Temperature: &temperature,
		MaxTokens:   &maxTokens,
	}

	raw := `{"temperature":0.2,"max_tokens":8}`
	if err := ApplyParamOverride(req, &raw); err != nil {
		t.Fatalf("ApplyParamOverride returned error: %v", err)
	}
	if req.Temperature == nil || *req.Temperature != 0.2 {
		t.Fatalf("expected temperature=0.2, got %#v", req.Temperature)
	}
	if req.MaxTokens == nil || *req.MaxTokens != 8 {
		t.Fatalf("expected max_tokens=8, got %#v", req.MaxTokens)
	}
	if req.Model != "gpt-4" {
		t.Fatalf("model should not change, got %q", req.Model)
	}
}

func TestApplyParamOverrideRejectsRoutingFields(t *testing.T) {
	content := "hello"
	req := &transformerModel.InternalLLMRequest{
		Model:    "gpt-4",
		Messages: []transformerModel.Message{{Role: "user", Content: transformerModel.MessageContent{Content: &content}}},
	}

	raw := `{"model":"other-model"}`
	if err := ApplyParamOverride(req, &raw); err == nil {
		t.Fatal("expected model override to be rejected")
	}
}

func TestApplyParamOverrideInvalidJSON(t *testing.T) {
	content := "hello"
	req := &transformerModel.InternalLLMRequest{
		Model:    "gpt-4",
		Messages: []transformerModel.Message{{Role: "user", Content: transformerModel.MessageContent{Content: &content}}},
	}

	raw := `{bad json`
	if err := ApplyParamOverride(req, &raw); err == nil {
		t.Fatal("expected invalid JSON to fail")
	}
}

func TestApplyParamOverridePreservesResponsesHelpFields(t *testing.T) {
	content := "hello"
	previousResponseID := "resp_123"
	truncation := "auto"
	reasoningBudget := int64(256)
	req := &transformerModel.InternalLLMRequest{
		Model:              "gpt-4.1",
		Messages:           []transformerModel.Message{{Role: "user", Content: transformerModel.MessageContent{Content: &content}}},
		PreviousResponseID: &previousResponseID,
		Truncation:         &truncation,
		Include:            []string{"reasoning.encrypted_content"},
		ReasoningBudget:    &reasoningBudget,
	}

	raw := `{"temperature":0.2}`
	if err := ApplyParamOverride(req, &raw); err != nil {
		t.Fatalf("ApplyParamOverride returned error: %v", err)
	}
	if req.PreviousResponseID == nil || *req.PreviousResponseID != previousResponseID {
		t.Fatalf("previous_response_id was not preserved: %v", req.PreviousResponseID)
	}
	if req.Truncation == nil || *req.Truncation != truncation {
		t.Fatalf("truncation was not preserved: %v", req.Truncation)
	}
	if len(req.Include) != 1 || req.Include[0] != "reasoning.encrypted_content" {
		t.Fatalf("include was not preserved: %#v", req.Include)
	}
	if req.ReasoningBudget == nil || *req.ReasoningBudget != reasoningBudget {
		t.Fatalf("reasoning budget was not preserved: %v", req.ReasoningBudget)
	}
}

func TestApplyParamOverrideAllowsExtraBody(t *testing.T) {
	content := "hello"
	req := &transformerModel.InternalLLMRequest{
		Model:    "gpt-4.1",
		Messages: []transformerModel.Message{{Role: "user", Content: transformerModel.MessageContent{Content: &content}}},
	}

	raw := `{"extra_body":{"caching":{"type":"enabled"},"max_tool_calls":4}}`
	if err := ApplyParamOverride(req, &raw); err != nil {
		t.Fatalf("ApplyParamOverride returned error: %v", err)
	}

	var extraBody map[string]any
	if err := json.Unmarshal(req.ExtraBody, &extraBody); err != nil {
		t.Fatalf("extra_body was not preserved as JSON object: %v", err)
	}
	if _, ok := extraBody["caching"]; !ok {
		t.Fatalf("extra_body.caching was not preserved: %#v", extraBody)
	}
	if got := extraBody["max_tool_calls"]; got != float64(4) {
		t.Fatalf("extra_body.max_tool_calls = %#v, want 4", got)
	}
}

func TestApplyParamOverrideAllowsPromptCacheRetention(t *testing.T) {
	content := "hello"
	req := &transformerModel.InternalLLMRequest{
		Model:    "gpt-4.1",
		Messages: []transformerModel.Message{{Role: "user", Content: transformerModel.MessageContent{Content: &content}}},
	}

	raw := `{"prompt_cache_retention":"in-memory"}`
	if err := ApplyParamOverride(req, &raw); err != nil {
		t.Fatalf("ApplyParamOverride returned error: %v", err)
	}
	if req.PromptCacheRetention == nil || *req.PromptCacheRetention != "in-memory" {
		t.Fatalf("prompt_cache_retention was not applied: %v", req.PromptCacheRetention)
	}
}

func TestCloneInternalLLMRequestDeepCopiesHelpFields(t *testing.T) {
	content := "hello"
	arrayInputs := true
	req := &transformerModel.InternalLLMRequest{
		Model:               "gpt-4.1",
		Messages:            []transformerModel.Message{{Role: "user", Content: transformerModel.MessageContent{Content: &content}}},
		RawRequest:          []byte(`{"model":"gpt-4.1"}`),
		TransformerMetadata: map[string]string{"source": "responses"},
		TransformOptions:    transformerModel.TransformOptions{ArrayInputs: &arrayInputs},
		Include:             []string{"reasoning.encrypted_content"},
		Query:               url.Values{"beta": []string{"true"}},
	}

	cloned, err := CloneInternalLLMRequest(req)
	if err != nil {
		t.Fatalf("CloneInternalLLMRequest returned error: %v", err)
	}

	cloned.RawRequest[0] = '['
	cloned.TransformerMetadata["source"] = "mutated"
	*cloned.TransformOptions.ArrayInputs = false
	cloned.Include[0] = "mutated"
	cloned.Query["beta"][0] = "false"

	if string(req.RawRequest) != `{"model":"gpt-4.1"}` {
		t.Fatalf("RawRequest was aliased: %s", req.RawRequest)
	}
	if req.TransformerMetadata["source"] != "responses" {
		t.Fatalf("TransformerMetadata was aliased: %#v", req.TransformerMetadata)
	}
	if req.TransformOptions.ArrayInputs == nil || *req.TransformOptions.ArrayInputs != true {
		t.Fatalf("TransformOptions.ArrayInputs was aliased: %#v", req.TransformOptions.ArrayInputs)
	}
	if len(req.Include) != 1 || req.Include[0] != "reasoning.encrypted_content" {
		t.Fatalf("Include was aliased: %#v", req.Include)
	}
	if got := req.Query.Get("beta"); got != "true" {
		t.Fatalf("Query was aliased: %#v", req.Query)
	}
}
