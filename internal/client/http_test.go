package client

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

func TestSocksProxyDialContextHonorsCanceledContext(t *testing.T) {
	client, err := newHTTPClientCustomProxy("socks5://127.0.0.1:1")
	if err != nil {
		t.Fatalf("newHTTPClientCustomProxy returned error: %v", err)
	}
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport = %T, want *http.Transport", client.Transport)
	}
	if transport.DialContext == nil {
		t.Fatal("DialContext should be set for socks proxy")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	conn, err := transport.DialContext(ctx, "tcp", "example.com:443")
	if conn != nil {
		_ = conn.Close()
		t.Fatal("expected nil connection for canceled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("DialContext error = %v, want context.Canceled", err)
	}
}
