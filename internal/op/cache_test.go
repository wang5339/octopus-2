package op

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/bestruirui/octopus/internal/db"
	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/utils/cache"
)

func initOpCacheTestDB(t *testing.T) {
	t.Helper()
	if err := db.InitDB("sqlite", filepath.Join(t.TempDir(), "octopus-test.db"), false); err != nil {
		t.Fatalf("InitDB returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
		resetOpCachesForTest()
	})
	resetOpCachesForTest()
}

func resetOpCachesForTest() {
	settingCache = cache.New[model.SettingKey, string](16)
	channelCache = cache.New[int, model.Channel](16)
	channelKeyCache = cache.New[int, model.ChannelKey](16)
	channelKeyCacheNeedUpdate = make(map[int]struct{})
	groupCache = cache.New[int, model.Group](16)
	groupMap = cache.New[string, model.Group](16)
	llmModelCache = cache.New[string, model.LLMPrice](16)
}

func TestSettingRefreshCacheClearsStaleEntries(t *testing.T) {
	initOpCacheTestDB(t)

	staleKey := model.SettingKey("removed_setting")
	settingCache.Set(staleKey, "stale")
	settingCache.Set(model.SettingKeyAPIBaseURL, "http://stale.local")

	if err := db.GetDB().Create(&model.Setting{
		Key:   model.SettingKeyAPIBaseURL,
		Value: "https://fresh.example",
	}).Error; err != nil {
		t.Fatalf("create setting returned error: %v", err)
	}

	if err := settingRefreshCache(context.Background()); err != nil {
		t.Fatalf("settingRefreshCache returned error: %v", err)
	}

	if _, err := SettingGetString(staleKey); err == nil {
		t.Fatal("SettingGetString(removed_setting) returned nil error, want stale cache entry removed")
	}

	got, err := SettingGetString(model.SettingKeyAPIBaseURL)
	if err != nil {
		t.Fatalf("SettingGetString(api_base_url) returned error: %v", err)
	}
	if got != "https://fresh.example" {
		t.Fatalf("api_base_url = %q, want %q", got, "https://fresh.example")
	}
}

func TestChannelRefreshCacheClearsStaleChannelsAndKeys(t *testing.T) {
	initOpCacheTestDB(t)

	channelCache.Set(99, model.Channel{
		ID:   99,
		Name: "stale-channel",
		Keys: []model.ChannelKey{{ID: 199, ChannelID: 99, ChannelKey: "stale-key"}},
	})
	channelKeyCache.Set(199, model.ChannelKey{ID: 199, ChannelID: 99, ChannelKey: "stale-key"})
	channelKeyCacheNeedUpdate[199] = struct{}{}

	if err := db.GetDB().Create(&model.Channel{
		ID:   1,
		Name: "fresh-channel",
		Keys: []model.ChannelKey{{ID: 11, ChannelID: 1, ChannelKey: "fresh-key"}},
	}).Error; err != nil {
		t.Fatalf("create channel returned error: %v", err)
	}

	if err := channelRefreshCache(context.Background()); err != nil {
		t.Fatalf("channelRefreshCache returned error: %v", err)
	}

	if _, err := ChannelGet(99, context.Background()); err == nil {
		t.Fatal("ChannelGet(99) returned nil error, want stale cache entry removed")
	}
	if _, ok := channelKeyCache.Get(199); ok {
		t.Fatal("stale channel key 199 still exists in channelKeyCache")
	}

	fresh, err := ChannelGet(1, context.Background())
	if err != nil {
		t.Fatalf("ChannelGet(1) returned error: %v", err)
	}
	if fresh.Name != "fresh-channel" || len(fresh.Keys) != 1 || fresh.Keys[0].ID != 11 {
		t.Fatalf("fresh channel cache = %#v, want channel with key 11", fresh)
	}
	if _, ok := channelKeyCache.Get(11); !ok {
		t.Fatal("fresh channel key 11 missing from channelKeyCache")
	}
	if len(channelKeyCacheNeedUpdate) != 0 {
		t.Fatalf("channelKeyCacheNeedUpdate len = %d, want 0 after full refresh", len(channelKeyCacheNeedUpdate))
	}
}

func TestGroupRefreshCacheClearsStaleEntries(t *testing.T) {
	initOpCacheTestDB(t)

	groupCache.Set(99, model.Group{ID: 99, Name: "stale-group"})
	groupMap.Set("stale-group", model.Group{ID: 99, Name: "stale-group"})

	if err := db.GetDB().Create(&model.Group{
		ID:   1,
		Name: "fresh-group",
		Mode: model.GroupModeRoundRobin,
	}).Error; err != nil {
		t.Fatalf("create group returned error: %v", err)
	}

	if err := groupRefreshCache(context.Background()); err != nil {
		t.Fatalf("groupRefreshCache returned error: %v", err)
	}

	if _, ok := groupCache.Get(99); ok {
		t.Fatal("stale group id 99 still exists in groupCache")
	}
	if _, err := GroupGetEnabledMap("stale-group", context.Background()); err == nil {
		t.Fatal("GroupGetEnabledMap(stale-group) returned nil error, want stale groupMap entry removed")
	}

	fresh, err := GroupGetEnabledMap("fresh-group", context.Background())
	if err != nil {
		t.Fatalf("GroupGetEnabledMap(fresh-group) returned error: %v", err)
	}
	if fresh.ID != 1 || fresh.Name != "fresh-group" {
		t.Fatalf("fresh group cache = %#v, want fresh group id 1", fresh)
	}
}
