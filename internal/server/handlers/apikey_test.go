package handlers

import (
	"testing"

	"github.com/bestruirui/octopus/internal/model"
)

func ptr[T any](v T) *T {
	return &v
}

func TestUpdateAPIKeyRequestApplyToPreservesMissingFields(t *testing.T) {
	apiKey := model.APIKey{
		ID:              1,
		Name:            "old",
		APIKey:          "sk-old",
		APIKeyHash:      "hash-old",
		Enabled:         true,
		ExpireAt:        123,
		MaxCost:         45.6,
		SupportedModels: "gpt-4o",
	}

	req := updateAPIKeyRequest{
		ID:   1,
		Name: ptr("new"),
	}

	req.applyTo(&apiKey)

	if apiKey.Name != "new" {
		t.Fatalf("expected name to be updated, got %q", apiKey.Name)
	}
	if apiKey.APIKey != "sk-old" ||
		apiKey.APIKeyHash != "hash-old" ||
		!apiKey.Enabled ||
		apiKey.ExpireAt != 123 ||
		apiKey.MaxCost != 45.6 ||
		apiKey.SupportedModels != "gpt-4o" {
		t.Fatalf("missing fields should be preserved, got %+v", apiKey)
	}
}

func TestUpdateAPIKeyRequestApplyToAllowsZeroValues(t *testing.T) {
	apiKey := model.APIKey{
		ID:              1,
		Name:            "old",
		APIKey:          "sk-old",
		APIKeyHash:      "hash-old",
		Enabled:         true,
		ExpireAt:        123,
		MaxCost:         45.6,
		SupportedModels: "gpt-4o",
	}

	req := updateAPIKeyRequest{
		ID:              1,
		Enabled:         ptr(false),
		ExpireAt:        ptr(int64(0)),
		MaxCost:         ptr(0.0),
		SupportedModels: ptr(""),
	}

	req.applyTo(&apiKey)

	if apiKey.Enabled ||
		apiKey.ExpireAt != 0 ||
		apiKey.MaxCost != 0 ||
		apiKey.SupportedModels != "" {
		t.Fatalf("explicit zero values should be applied, got %+v", apiKey)
	}
	if apiKey.Name != "old" || apiKey.APIKey != "sk-old" || apiKey.APIKeyHash != "hash-old" {
		t.Fatalf("unmentioned identity fields should be preserved, got %+v", apiKey)
	}
}

func TestNewAPIKeyResponseHidesHash(t *testing.T) {
	apiKey := model.APIKey{
		ID:              1,
		Name:            "demo",
		APIKey:          "sk-octopus-********abcd",
		APIKeyHash:      "secret-hash",
		Enabled:         true,
		ExpireAt:        123,
		MaxCost:         4.5,
		SupportedModels: "gpt-4o",
	}

	resp := newAPIKeyResponse(apiKey)
	if resp.APIKey != apiKey.APIKey {
		t.Fatalf("response should include masked/api key field, got %+v", resp)
	}
}
