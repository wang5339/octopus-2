package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/bestruirui/octopus/internal/server/resp"
	"github.com/gin-gonic/gin"
)

// RateLimiter 简单的速率限制器
type RateLimiter struct {
	visitors map[string]*visitor
	mu       sync.RWMutex
	rate     int           // 每个时间窗口允许的请求数
	window   time.Duration // 时间窗口
}

type visitor struct {
	count      int
	lastReset  time.Time
	mu         sync.Mutex
}

// NewRateLimiter 创建速率限制器
// rate: 每个时间窗口允许的请求数
// window: 时间窗口（如 1 分钟）
func NewRateLimiter(rate int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate,
		window:   window,
	}

	// 启动清理 goroutine，定期清理过期的访客记录
	go rl.cleanup()

	return rl
}

// cleanup 定期清理过期的访客记录，防止内存泄漏
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, v := range rl.visitors {
			v.mu.Lock()
			if now.Sub(v.lastReset) > rl.window*2 {
				delete(rl.visitors, key)
			}
			v.mu.Unlock()
		}
		rl.mu.Unlock()
	}
}

// getVisitor 获取或创建访客记录
func (rl *RateLimiter) getVisitor(key string) *visitor {
	rl.mu.RLock()
	v, exists := rl.visitors[key]
	rl.mu.RUnlock()

	if exists {
		return v
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// 双重检查
	v, exists = rl.visitors[key]
	if exists {
		return v
	}

	v = &visitor{
		lastReset: time.Now(),
	}
	rl.visitors[key] = v
	return v
}

// Allow 检查是否允许请求
func (rl *RateLimiter) Allow(key string) bool {
	v := rl.getVisitor(key)

	v.mu.Lock()
	defer v.mu.Unlock()

	now := time.Now()
	if now.Sub(v.lastReset) > rl.window {
		v.count = 0
		v.lastReset = now
	}

	if v.count >= rl.rate {
		return false
	}

	v.count++
	return true
}

// RateLimit 速率限制中间件
// rate: 每个时间窗口允许的请求数
// window: 时间窗口
// keyFunc: 生成限流 key 的函数（如基于 IP、API Key 等）
func RateLimit(rate int, window time.Duration, keyFunc func(*gin.Context) string) gin.HandlerFunc {
	limiter := NewRateLimiter(rate, window)

	return func(c *gin.Context) {
		key := keyFunc(c)
		if key == "" {
			c.Next()
			return
		}

		if !limiter.Allow(key) {
			resp.Error(c, http.StatusTooManyRequests, "rate limit exceeded")
			c.Abort()
			return
		}

		c.Next()
	}
}

// IPRateLimit 基于 IP 的速率限制
func IPRateLimit(rate int, window time.Duration) gin.HandlerFunc {
	return RateLimit(rate, window, func(c *gin.Context) string {
		return c.ClientIP()
	})
}
