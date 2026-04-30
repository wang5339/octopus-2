package task

import (
	"testing"
	"time"

	"github.com/bestruirui/octopus/internal/helper"
	"github.com/bestruirui/octopus/internal/model"
)

func TestStartSyncModelsTaskAsyncSkipsWhenAlreadyRunning(t *testing.T) {
	syncModelsRunning.Store(false)
	originalRunner := syncModelsTaskRunner
	defer func() {
		syncModelsTaskRunner = originalRunner
		syncModelsRunning.Store(false)
	}()

	started := make(chan struct{})
	release := make(chan struct{})
	syncModelsTaskRunner = func() {
		close(started)
		<-release
	}

	if !StartSyncModelsTaskAsync() {
		t.Fatal("first StartSyncModelsTaskAsync returned false, want true")
	}
	waitForChannel(t, started, "sync task to start")

	if !IsSyncModelsTaskRunning() {
		t.Fatal("sync task running state = false, want true")
	}
	if StartSyncModelsTaskAsync() {
		t.Fatal("second StartSyncModelsTaskAsync returned true while task is already running")
	}

	close(release)
	waitForCondition(t, func() bool {
		return !IsSyncModelsTaskRunning()
	}, "sync task running state to clear")
}

func TestLastSyncModelsTimeDefaultsToZero(t *testing.T) {
	lastSyncModelsTimeMu.Lock()
	original := lastSyncModelsTime
	lastSyncModelsTime = time.Time{}
	lastSyncModelsTimeMu.Unlock()
	defer func() {
		lastSyncModelsTimeMu.Lock()
		lastSyncModelsTime = original
		lastSyncModelsTimeMu.Unlock()
	}()

	if got := GetLastSyncModelsTime(); !got.IsZero() {
		t.Fatalf("GetLastSyncModelsTime() = %v, want zero time before first successful sync", got)
	}
}

func TestCollectConfiguredModelNamesProtectsManualAndCustomModels(t *testing.T) {
	channels := []model.Channel{
		{
			AutoSync:    false,
			Model:       "manual-a, Shared-Model",
			CustomModel: "alias-a",
		},
		{
			AutoSync:    true,
			Model:       "old-removed",
			CustomModel: "custom-sync",
		},
	}

	knownModelNames, seen := collectConfiguredModelNames(channels)
	appendNormalizedLowerModelNames(&knownModelNames, seen, []string{"upstream-new", "MANUAL-A"})

	wantKnown := []string{
		"manual-a",
		"shared-model",
		"alias-a",
		"old-removed",
		"custom-sync",
		"upstream-new",
	}
	assertStringSlicesEqual(t, knownModelNames, wantKnown)

	llmPriceNames := []string{"manual-a", "alias-a", "upstream-new", "unused-zero-price"}
	deleteCandidates := helper.SubtractModelNames(llmPriceNames, knownModelNames)
	assertStringSlicesEqual(t, deleteCandidates, []string{"unused-zero-price"})
}

func waitForChannel(t *testing.T, ch <-chan struct{}, reason string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", reason)
	}
}

func waitForCondition(t *testing.T, condition func() bool, reason string) {
	t.Helper()
	deadline := time.After(time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		if condition() {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for %s", reason)
		case <-ticker.C:
		}
	}
}

func assertStringSlicesEqual(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("slice length = %d, want %d; got=%v want=%v", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("slice[%d] = %q, want %q; got=%v want=%v", i, got[i], want[i], got, want)
		}
	}
}
