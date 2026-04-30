package op

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/bestruirui/octopus/internal/db"
	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/utils/cache"
)

func TestNormalizeLLMName(t *testing.T) {
	got := normalizeLLMName("  GPT-4O  ")
	if got != "gpt-4o" {
		t.Fatalf("expected normalized model name, got %q", got)
	}
}

func TestNormalizeLLMNamesTrimsLowersAndDeduplicates(t *testing.T) {
	got := normalizeLLMNames([]string{" GPT-4O ", "gpt-4o", "", " Claude-3 "})
	want := []string{"gpt-4o", "claude-3"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestLLMRefreshCacheClearsStaleEntriesAndNormalizesNames(t *testing.T) {
	if err := db.InitDB("sqlite", filepath.Join(t.TempDir(), "octopus-test.db"), false); err != nil {
		t.Fatalf("InitDB returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
		llmModelCache = cache.New[string, model.LLMPrice](16)
	})

	llmModelCache = cache.New[string, model.LLMPrice](16)
	llmModelCache.Set("stale-model", model.LLMPrice{})
	llmModelCache.Set("fresh-model", model.LLMPrice{Input: 99})

	if err := db.GetDB().Create(&model.LLMInfo{
		Name: " Fresh-Model ",
		LLMPrice: model.LLMPrice{
			Input:  1,
			Output: 2,
		},
	}).Error; err != nil {
		t.Fatalf("create LLMInfo returned error: %v", err)
	}

	if err := llmRefreshCache(context.Background()); err != nil {
		t.Fatalf("llmRefreshCache returned error: %v", err)
	}

	if _, err := LLMGet("stale-model"); err == nil {
		t.Fatal("LLMGet(stale-model) returned nil error, want stale cache entry removed")
	}

	price, err := LLMGet("fresh-model")
	if err != nil {
		t.Fatalf("LLMGet(fresh-model) returned error: %v", err)
	}
	if price.Input != 1 || price.Output != 2 {
		t.Fatalf("fresh-model price = %+v, want input=1 output=2", price)
	}
}
