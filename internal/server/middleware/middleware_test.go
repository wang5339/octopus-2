package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestMaxRequestBodySize(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		maxBytes       int64
		bodySize       int
		expectedStatus int
	}{
		{
			name:           "small request should pass",
			maxBytes:       1024,
			bodySize:       512,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "exact size should pass",
			maxBytes:       1024,
			bodySize:       1024,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "oversized request should fail",
			maxBytes:       1024,
			bodySize:       2048,
			expectedStatus: http.StatusRequestEntityTooLarge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(MaxRequestBodySize(tt.maxBytes))
			r.POST("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			body := bytes.Repeat([]byte("a"), tt.bodySize)
			req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(body))
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestIPRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	// 允许 2 次请求每秒
	r.Use(IPRateLimit(2, 1*time.Second))
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// 第一次请求应该成功
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Errorf("first request should succeed, got status %d", w1.Code)
	}

	// 第二次请求应该成功
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Errorf("second request should succeed, got status %d", w2.Code)
	}

	// 第三次请求应该被限流
	req3 := httptest.NewRequest(http.MethodGet, "/test", nil)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusTooManyRequests {
		t.Errorf("third request should be rate limited, got status %d", w3.Code)
	}

	// 等待 1 秒后应该可以再次请求
	time.Sleep(1100 * time.Millisecond)
	req4 := httptest.NewRequest(http.MethodGet, "/test", nil)
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, req4)
	if w4.Code != http.StatusOK {
		t.Errorf("request after cooldown should succeed, got status %d", w4.Code)
	}
}

func TestIsLocalDevOrigin(t *testing.T) {
	tests := []struct {
		name   string
		origin string
		want   bool
	}{
		{name: "localhost with port", origin: "http://localhost:5173", want: true},
		{name: "ipv4 loopback with port", origin: "http://127.0.0.1:5173", want: true},
		{name: "ipv6 loopback with port", origin: "http://[::1]:5173", want: true},
		{name: "ipv6 loopback without port", origin: "http://[::1]", want: true},
		{name: "external domain", origin: "https://example.com", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isLocalDevOrigin(tt.origin); got != tt.want {
				t.Fatalf("isLocalDevOrigin(%q)=%v, want %v", tt.origin, got, tt.want)
			}
		})
	}
}
