package middleware

import (
	"net/http"

	"github.com/bestruirui/octopus/internal/server/resp"
	"github.com/gin-gonic/gin"
)

// MaxRequestBodySize 限制请求体的最大大小（默认 10MB）
// 防止恶意用户发送超大请求导致内存耗尽
func MaxRequestBodySize(maxBytes int64) gin.HandlerFunc {
	if maxBytes <= 0 {
		maxBytes = 10 * 1024 * 1024 // 默认 10MB
	}

	return func(c *gin.Context) {
		if c.Request.ContentLength > maxBytes {
			resp.Error(c, http.StatusRequestEntityTooLarge, "request body too large")
			c.Abort()
			return
		}

		// 使用 http.MaxBytesReader 限制实际读取的字节数
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)

		c.Next()
	}
}
