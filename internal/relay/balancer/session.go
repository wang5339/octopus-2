package balancer

import (
	"fmt"
	"sync"
	"time"
)

// SessionEntry 会话保持条目
type SessionEntry struct {
	ChannelID    int
	ChannelKeyID int
	Timestamp    time.Time
}

// 全局会话存储
var globalSession sync.Map // key: string -> value: *SessionEntry

// sessionKey 生成会话键：apiKeyID:requestModel
func sessionKey(apiKeyID int, requestModel string) string {
	return fmt.Sprintf("%d:%s", apiKeyID, requestModel)
}

// GetSticky 获取粘性通道（ttl 内有效）
// ttl 由 Group.SessionKeepTime 决定，返回 nil 表示无有效会话
func GetSticky(apiKeyID int, requestModel string, ttl time.Duration) *SessionEntry {
	key := sessionKey(apiKeyID, requestModel)
	v, ok := globalSession.Load(key)
	if !ok {
		return nil
	}
	entry := v.(*SessionEntry)

	if time.Since(entry.Timestamp) > ttl {
		// 过期，惰性清除
		globalSession.Delete(key)
		return nil
	}

	return entry
}

// SetSticky 写入/更新粘性记录
func SetSticky(apiKeyID int, requestModel string, channelID, keyID int) {
	key := sessionKey(apiKeyID, requestModel)
	globalSession.Store(key, &SessionEntry{
		ChannelID:    channelID,
		ChannelKeyID: keyID,
		Timestamp:    time.Now(),
	})
}

// CleanupExpiredSessions 清理过期的会话记录
// 应该定期调用（如每小时一次）以防止内存泄漏
func CleanupExpiredSessions(maxAge time.Duration) int {
	count := 0
	now := time.Now()
	globalSession.Range(func(key, value interface{}) bool {
		entry := value.(*SessionEntry)
		if now.Sub(entry.Timestamp) > maxAge {
			globalSession.Delete(key)
			count++
		}
		return true
	})
	return count
}
