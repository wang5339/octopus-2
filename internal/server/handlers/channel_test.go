package handlers

import (
	"context"
	"strings"
	"testing"

	"github.com/bestruirui/octopus/internal/model"
	transformerOutbound "github.com/bestruirui/octopus/internal/transformer/outbound"
)

func TestModelProtocolDryRunProbeDoesNotPassOrRecommend(t *testing.T) {
	probe := modelProtocolProbeFromTestResult(
		transformerOutbound.OutboundTypeAnthropic,
		testModelResult{
			Model:  "claude-3-5-sonnet",
			Passed: true,
			Error:  "Dry run: request transform succeeded; no upstream request was sent",
		},
		true,
	)

	if probe.Passed {
		t.Fatalf("dry-run protocol probe must not be treated as upstream pass")
	}
	if !probe.DryRun {
		t.Fatalf("expected dry-run probe to be marked in response")
	}
	if got := recommendModelProtocol(transformerOutbound.OutboundTypeOpenAIChat, "claude-3-5-sonnet", []modelProtocolProbeResult{probe}); got != nil {
		t.Fatalf("dry-run-only probe must not produce recommendation, got %v", *got)
	}
}

func TestModelProtocolRealProbeCanRecommend(t *testing.T) {
	probe := modelProtocolProbeFromTestResult(
		transformerOutbound.OutboundTypeAnthropic,
		testModelResult{Model: "claude-3-5-sonnet", Passed: true},
		false,
	)

	got := recommendModelProtocol(transformerOutbound.OutboundTypeOpenAIChat, "claude-3-5-sonnet", []modelProtocolProbeResult{probe})
	if got == nil || *got != transformerOutbound.OutboundTypeAnthropic {
		t.Fatalf("expected real Anthropic pass to recommend Anthropic, got %v", got)
	}
}

func TestNewChannelModelTestRequestUsesProtocolTokenLimitField(t *testing.T) {
	tests := []struct {
		name              string
		channelType       transformerOutbound.OutboundType
		modelName         string
		wantMaxTokens     bool
		wantMaxCompletion bool
	}{
		{
			name:          "openai chat keeps chat max_tokens",
			channelType:   transformerOutbound.OutboundTypeOpenAIChat,
			modelName:     "gpt-4.1",
			wantMaxTokens: true,
		},
		{
			name:              "openai responses uses max_output_tokens source field",
			channelType:       transformerOutbound.OutboundTypeOpenAIResponse,
			modelName:         "gpt-4.1",
			wantMaxCompletion: true,
		},
		{
			name:              "volcengine responses uses max_output_tokens source field",
			channelType:       transformerOutbound.OutboundTypeVolcengine,
			modelName:         "doubao-seed-1-6-lite-251015",
			wantMaxCompletion: true,
		},
		{
			name:          "anthropic keeps max_tokens",
			channelType:   transformerOutbound.OutboundTypeAnthropic,
			modelName:     "claude-3-5-sonnet",
			wantMaxTokens: true,
		},
		{
			name:          "gemini keeps max_tokens for generationConfig.maxOutputTokens",
			channelType:   transformerOutbound.OutboundTypeGemini,
			modelName:     "gemini-2.5-pro",
			wantMaxTokens: true,
		},
		{
			name:              "zen gpt routes to responses",
			channelType:       transformerOutbound.OutboundTypeZen,
			modelName:         "gpt-4.1",
			wantMaxCompletion: true,
		},
		{
			name:          "zen claude routes to anthropic",
			channelType:   transformerOutbound.OutboundTypeZen,
			modelName:     "claude-3-5-sonnet",
			wantMaxTokens: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newChannelModelTestRequest(tt.channelType, tt.modelName)

			if (req.MaxTokens != nil) != tt.wantMaxTokens {
				t.Fatalf("MaxTokens set = %v, want %v", req.MaxTokens != nil, tt.wantMaxTokens)
			}
			if (req.MaxCompletionTokens != nil) != tt.wantMaxCompletion {
				t.Fatalf("MaxCompletionTokens set = %v, want %v", req.MaxCompletionTokens != nil, tt.wantMaxCompletion)
			}
			if req.MaxTokens != nil && *req.MaxTokens != 1 {
				t.Fatalf("MaxTokens = %d, want 1", *req.MaxTokens)
			}
			if req.MaxCompletionTokens != nil && *req.MaxCompletionTokens != 1 {
				t.Fatalf("MaxCompletionTokens = %d, want 1", *req.MaxCompletionTokens)
			}
		})
	}
}

func TestModelTestLimitError(t *testing.T) {
	tests := []struct {
		name      string
		count     int
		dryRun    bool
		wantError bool
	}{
		{name: "empty models", count: 0, dryRun: true, wantError: true},
		{name: "dry run over global max", count: maxModelTestCount + 1, dryRun: true, wantError: true},
		{name: "real request over max", count: maxRealModelTestCount + 1, dryRun: false, wantError: true},
		{name: "dry run at global max", count: maxModelTestCount, dryRun: true, wantError: false},
		{name: "real request at max", count: maxRealModelTestCount, dryRun: false, wantError: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := modelTestLimitError(tt.count, tt.dryRun)
			if (got != "") != tt.wantError {
				t.Fatalf("modelTestLimitError(%d, %v) = %q, wantError=%v", tt.count, tt.dryRun, got, tt.wantError)
			}
		})
	}
}

func TestModelProtocolDetectLimitError(t *testing.T) {
	tests := []struct {
		name          string
		modelCount    int
		protocolCount int
		dryRun        bool
		wantError     bool
	}{
		{name: "empty models", modelCount: 0, protocolCount: 5, dryRun: true, wantError: true},
		{name: "dry run over global model max", modelCount: maxModelTestCount + 1, protocolCount: 5, dryRun: true, wantError: true},
		{name: "real request over probe max", modelCount: 3, protocolCount: 5, dryRun: false, wantError: true},
		{name: "real request at probe max", modelCount: 2, protocolCount: 5, dryRun: false, wantError: false},
		{name: "dry run at global model max", modelCount: maxModelTestCount, protocolCount: 5, dryRun: true, wantError: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := modelProtocolDetectLimitError(tt.modelCount, tt.protocolCount, tt.dryRun)
			if (got != "") != tt.wantError {
				t.Fatalf(
					"modelProtocolDetectLimitError(%d, %d, %v) = %q, wantError=%v",
					tt.modelCount,
					tt.protocolCount,
					tt.dryRun,
					got,
					tt.wantError,
				)
			}
		})
	}
}

func TestRunChannelModelTestDryRunAppliesAllowedExtraBody(t *testing.T) {
	paramOverride := `{"extra_body":{"caching":{"type":"enabled"},"max_tool_calls":4}}`
	channel := model.Channel{
		Type: transformerOutbound.OutboundTypeVolcengine,
		BaseUrls: []model.BaseUrl{
			{URL: "https://ark.example.com/api/v3"},
		},
		Keys: []model.ChannelKey{
			{Enabled: true, ChannelKey: "test-key"},
		},
		ParamOverride: &paramOverride,
	}

	result := runChannelModelTestWithType(
		context.Background(),
		&channel,
		transformerOutbound.OutboundTypeVolcengine,
		"doubao-seed-1-6-lite-251015",
		true,
	)

	if !result.Passed {
		t.Fatalf("expected dry-run with allowed extra_body to pass, got %+v", result)
	}
	if !strings.Contains(result.Error, "Dry run: request transform succeeded") {
		t.Fatalf("expected dry-run success message, got %+v", result)
	}
}

func TestRunChannelModelTestDryRunRejectsUnsupportedExtraBody(t *testing.T) {
	paramOverride := `{"extra_body":{"prompt_cache_key":"should-not-pass-through"}}`
	channel := model.Channel{
		Type: transformerOutbound.OutboundTypeVolcengine,
		BaseUrls: []model.BaseUrl{
			{URL: "https://ark.example.com/api/v3"},
		},
		Keys: []model.ChannelKey{
			{Enabled: true, ChannelKey: "test-key"},
		},
		ParamOverride: &paramOverride,
	}

	result := runChannelModelTestWithType(
		context.Background(),
		&channel,
		transformerOutbound.OutboundTypeVolcengine,
		"doubao-seed-1-6-lite-251015",
		true,
	)

	if result.Passed {
		t.Fatalf("expected unsupported extra_body to fail, got %+v", result)
	}
	want := `Failed to build request: failed to marshal responses api request: extra_body field "prompt_cache_key" is not allowed for volcengine responses`
	if result.Error != want {
		t.Fatalf("unexpected error: %q, want %q", result.Error, want)
	}
}
