package op

import (
	"sync"
	"testing"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/utils/cache"
)

func TestChannelKeyUpdateConcurrency(t *testing.T) {
	// 初始化测试缓存
	channelCache = cache.New[int, model.Channel](16)
	channelKeyCache = cache.New[int, model.ChannelKey](16)
	channelKeyCacheNeedUpdate = make(map[int]struct{})

	// 创建测试 channel
	testChannel := model.Channel{
		ID:   1,
		Name: "test-channel",
		Keys: []model.ChannelKey{
			{ID: 1, ChannelID: 1, ChannelKey: "key1", TotalCost: 0.0},
			{ID: 2, ChannelID: 1, ChannelKey: "key2", TotalCost: 0.0},
		},
	}
	channelCache.Set(1, testChannel)

	// 并发更新同一个 channel 的不同 keys
	var wg sync.WaitGroup
	iterations := 100

	// 并发更新 key1
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			key := model.ChannelKey{
				ID:        1,
				ChannelID: 1,
				TotalCost: float64(i),
			}
			if err := ChannelKeyUpdate(key); err != nil {
				t.Errorf("failed to update key1: %v", err)
			}
		}
	}()

	// 并发更新 key2
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			key := model.ChannelKey{
				ID:        2,
				ChannelID: 1,
				TotalCost: float64(i * 2),
			}
			if err := ChannelKeyUpdate(key); err != nil {
				t.Errorf("failed to update key2: %v", err)
			}
		}
	}()

	wg.Wait()

	// 验证最终状态
	finalChannel, ok := channelCache.Get(1)
	if !ok {
		t.Fatal("channel not found after concurrent updates")
	}

	if len(finalChannel.Keys) != 2 {
		t.Errorf("expected 2 keys, got %d", len(finalChannel.Keys))
	}

	// 验证两个 key 都存在
	key1Found := false
	key2Found := false
	for _, k := range finalChannel.Keys {
		if k.ID == 1 {
			key1Found = true
		}
		if k.ID == 2 {
			key2Found = true
		}
	}

	if !key1Found || !key2Found {
		t.Error("keys were lost during concurrent updates")
	}
}

func TestChannelBaseUrlUpdateConcurrency(t *testing.T) {
	// 初始化测试缓存
	channelCache = cache.New[int, model.Channel](16)

	// 创建测试 channel
	testChannel := model.Channel{
		ID:   1,
		Name: "test-channel",
		BaseUrls: []model.BaseUrl{
			{URL: "http://example.com", Delay: 100},
		},
	}
	channelCache.Set(1, testChannel)

	// 并发更新 BaseUrls
	var wg sync.WaitGroup
	iterations := 100

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				baseUrls := []model.BaseUrl{
					{URL: "http://example.com", Delay: id*100 + j},
				}
				if err := ChannelBaseUrlUpdate(1, baseUrls); err != nil {
					t.Errorf("failed to update base urls: %v", err)
				}
			}
		}(i)
	}

	wg.Wait()

	// 验证 channel 仍然存在且有效
	finalChannel, ok := channelCache.Get(1)
	if !ok {
		t.Fatal("channel not found after concurrent updates")
	}

	if len(finalChannel.BaseUrls) != 1 {
		t.Errorf("expected 1 base url, got %d", len(finalChannel.BaseUrls))
	}
}

func BenchmarkChannelKeyUpdate(b *testing.B) {
	// 初始化测试缓存
	channelCache = cache.New[int, model.Channel](16)
	channelKeyCache = cache.New[int, model.ChannelKey](16)
	channelKeyCacheNeedUpdate = make(map[int]struct{})

	// 创建测试 channel
	testChannel := model.Channel{
		ID:   1,
		Name: "test-channel",
		Keys: []model.ChannelKey{
			{ID: 1, ChannelID: 1, ChannelKey: "key1", TotalCost: 0.0},
		},
	}
	channelCache.Set(1, testChannel)

	key := model.ChannelKey{
		ID:        1,
		ChannelID: 1,
		TotalCost: 1.0,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ChannelKeyUpdate(key)
	}
}
