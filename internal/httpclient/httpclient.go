package httpclient

import (
	"fmt"
	"net/http"
	"time"
)

const (
	// 默认只限制“建立连接/等待响应头”的阶段，不设置 http.Client.Timeout，
	// 避免长时间流式响应被整体超时误杀。
	DefaultResponseHeaderTimeout = 60 * time.Second
	DefaultIdleConnTimeout       = 90 * time.Second
	DefaultExpectContinueTimeout = 1 * time.Second

	// OAuth / 元数据类短请求使用整体超时，避免 http.DefaultClient 无限等待。
	DefaultShortRequestTimeout = 60 * time.Second
)

// ApplyTransportTimeouts 为 transport 补齐保守超时。
// 它不设置 http.Client.Timeout，因此可安全用于 Relay 流式请求。
func ApplyTransportTimeouts(transport *http.Transport) {
	if transport.ResponseHeaderTimeout == 0 {
		transport.ResponseHeaderTimeout = DefaultResponseHeaderTimeout
	}
	if transport.IdleConnTimeout == 0 {
		transport.IdleConnTimeout = DefaultIdleConnTimeout
	}
	if transport.ExpectContinueTimeout == 0 {
		transport.ExpectContinueTimeout = DefaultExpectContinueTimeout
	}
}

// CloneDefaultTransport 复制默认 transport，并补齐连接/响应头阶段的超时。
func CloneDefaultTransport() (*http.Transport, error) {
	transport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, fmt.Errorf("default transport is not *http.Transport")
	}
	cloned := transport.Clone()
	ApplyTransportTimeouts(cloned)
	return cloned, nil
}

// NewShortTimeoutHTTPClient 返回适合 OAuth、版本信息、辅助请求等短请求的 client。
// Relay 流式请求不要使用该 client，避免整体 Timeout 截断长流。
func NewShortTimeoutHTTPClient() *http.Client {
	cloned, err := CloneDefaultTransport()
	if err != nil {
		return &http.Client{Timeout: DefaultShortRequestTimeout}
	}
	return &http.Client{Transport: cloned, Timeout: DefaultShortRequestTimeout}
}
