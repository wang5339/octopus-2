package outbound

import (
	"reflect"
	"testing"

	"github.com/bestruirui/octopus/internal/transformer/outbound/authropic"
	"github.com/bestruirui/octopus/internal/transformer/outbound/gemini"
	"github.com/bestruirui/octopus/internal/transformer/outbound/openai"
)

func TestGetForModelZenRoutesByModelPrefix(t *testing.T) {
	tests := []struct {
		name      string
		modelName string
		wantType  reflect.Type
	}{
		{
			name:      "claude models use Anthropic Messages",
			modelName: "claude-3-5-sonnet",
			wantType:  reflect.TypeOf(&authropic.MessageOutbound{}),
		},
		{
			name:      "gpt models use OpenAI Responses",
			modelName: "gpt-5-mini",
			wantType:  reflect.TypeOf(&openai.ResponseOutbound{}),
		},
		{
			name:      "gemini models use Gemini contents",
			modelName: "gemini-2.5-flash",
			wantType:  reflect.TypeOf(&gemini.MessagesOutbound{}),
		},
		{
			name:      "unknown models fall back to OpenAI Chat",
			modelName: "unknown-model",
			wantType:  reflect.TypeOf(&openai.ChatOutbound{}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetForModel(OutboundTypeZen, tt.modelName)
			if got == nil {
				t.Fatalf("GetForModel(%d, %q) returned nil", OutboundTypeZen, tt.modelName)
			}
			if gotType := reflect.TypeOf(got); gotType != tt.wantType {
				t.Fatalf("GetForModel(%d, %q) type = %v, want %v", OutboundTypeZen, tt.modelName, gotType, tt.wantType)
			}
		})
	}
}

func TestGetForModelNonZenUsesRegisteredFactory(t *testing.T) {
	got := GetForModel(OutboundTypeAnthropic, "gpt-5-mini")
	if got == nil {
		t.Fatal("GetForModel for non-Zen channel returned nil")
	}
	if gotType, wantType := reflect.TypeOf(got), reflect.TypeOf(&authropic.MessageOutbound{}); gotType != wantType {
		t.Fatalf("GetForModel for non-Zen channel type = %v, want %v", gotType, wantType)
	}
}
