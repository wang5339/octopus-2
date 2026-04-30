package openai

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/bestruirui/octopus/internal/transformer/model"
)

func TestConvertToResponsesRequestPreservesResponsesOnlyFields(t *testing.T) {
	content := "hello"
	promptCacheKey := "tenant-a-thread-1"
	promptCacheRetention := "in-memory"
	safetyIdentifier := "user-hash-1"
	previousResponseID := "resp_123"
	truncation := "auto"
	topLogprobs := int64(3)
	reasoningBudget := int64(256)
	reasoningEffort := "medium"
	internalReq := &model.InternalLLMRequest{
		Model: "gpt-4.1",
		Messages: []model.Message{
			{
				Role: "user",
				Content: model.MessageContent{
					Content: &content,
				},
			},
		},
		PromptCacheKey:       &promptCacheKey,
		PromptCacheRetention: &promptCacheRetention,
		SafetyIdentifier:     &safetyIdentifier,
		PreviousResponseID:   &previousResponseID,
		Truncation:           &truncation,
		Include:              []string{"reasoning.encrypted_content"},
		TopLogprobs:          &topLogprobs,
		ReasoningEffort:      reasoningEffort,
		ReasoningBudget:      &reasoningBudget,
	}

	responsesReq := ConvertToResponsesRequest(internalReq)
	if responsesReq.PromptCacheKey == nil || *responsesReq.PromptCacheKey != promptCacheKey {
		t.Fatalf("prompt_cache_key was not preserved: %v", responsesReq.PromptCacheKey)
	}
	if responsesReq.PromptCacheRetention == nil || *responsesReq.PromptCacheRetention != promptCacheRetention {
		t.Fatalf("prompt_cache_retention was not preserved: %v", responsesReq.PromptCacheRetention)
	}
	if responsesReq.SafetyIdentifier == nil || *responsesReq.SafetyIdentifier != safetyIdentifier {
		t.Fatalf("safety_identifier was not preserved: %v", responsesReq.SafetyIdentifier)
	}
	if responsesReq.PreviousResponseID == nil || *responsesReq.PreviousResponseID != previousResponseID {
		t.Fatalf("previous_response_id was not preserved: %v", responsesReq.PreviousResponseID)
	}
	if responsesReq.Truncation == nil || *responsesReq.Truncation != truncation {
		t.Fatalf("truncation was not preserved: %v", responsesReq.Truncation)
	}
	if !reflect.DeepEqual(responsesReq.Include, []string{"reasoning.encrypted_content"}) {
		t.Fatalf("include = %#v, want reasoning.encrypted_content", responsesReq.Include)
	}
	if responsesReq.TopLogprobs == nil || *responsesReq.TopLogprobs != topLogprobs {
		t.Fatalf("top_logprobs was not preserved: %v", responsesReq.TopLogprobs)
	}
	if responsesReq.Reasoning == nil {
		t.Fatal("reasoning was not preserved")
	}
	if responsesReq.Reasoning.Effort != reasoningEffort {
		t.Fatalf("reasoning.effort = %q, want %q", responsesReq.Reasoning.Effort, reasoningEffort)
	}
	if responsesReq.Reasoning.MaxTokens == nil || *responsesReq.Reasoning.MaxTokens != reasoningBudget {
		t.Fatalf("reasoning.max_tokens was not preserved: %v", responsesReq.Reasoning.MaxTokens)
	}

	encoded, err := json.Marshal(responsesReq)
	if err != nil {
		t.Fatalf("marshal responses request: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(encoded, &payload); err != nil {
		t.Fatalf("unmarshal encoded request: %v", err)
	}
	if got, ok := payload["prompt_cache_key"].(string); !ok || got != promptCacheKey {
		t.Fatalf("encoded prompt_cache_key = %#v, want %q", payload["prompt_cache_key"], promptCacheKey)
	}
	if got, ok := payload["prompt_cache_retention"].(string); !ok || got != promptCacheRetention {
		t.Fatalf("encoded prompt_cache_retention = %#v, want %q", payload["prompt_cache_retention"], promptCacheRetention)
	}
	if got, ok := payload["safety_identifier"].(string); !ok || got != safetyIdentifier {
		t.Fatalf("encoded safety_identifier = %#v, want %q", payload["safety_identifier"], safetyIdentifier)
	}
	if got, ok := payload["previous_response_id"].(string); !ok || got != previousResponseID {
		t.Fatalf("encoded previous_response_id = %#v, want %q", payload["previous_response_id"], previousResponseID)
	}
	if got, ok := payload["truncation"].(string); !ok || got != truncation {
		t.Fatalf("encoded truncation = %#v, want %q", payload["truncation"], truncation)
	}
	if got, ok := payload["top_logprobs"].(float64); !ok || got != float64(topLogprobs) {
		t.Fatalf("encoded top_logprobs = %#v, want %d", payload["top_logprobs"], topLogprobs)
	}
	include, ok := payload["include"].([]any)
	if !ok || len(include) != 1 || include[0] != "reasoning.encrypted_content" {
		t.Fatalf("encoded include = %#v, want reasoning.encrypted_content", payload["include"])
	}
	reasoning, ok := payload["reasoning"].(map[string]any)
	if !ok {
		t.Fatalf("encoded reasoning = %#v, want object", payload["reasoning"])
	}
	if got, ok := reasoning["effort"].(string); !ok || got != reasoningEffort {
		t.Fatalf("encoded reasoning.effort = %#v, want %q", reasoning["effort"], reasoningEffort)
	}
	if got, ok := reasoning["max_tokens"].(float64); !ok || got != float64(reasoningBudget) {
		t.Fatalf("encoded reasoning.max_tokens = %#v, want %d", reasoning["max_tokens"], reasoningBudget)
	}
}
