package relay

import (
	"strings"
	"testing"
)

func TestReadUpstreamErrorBodyTruncatesLargeBody(t *testing.T) {
	body := strings.NewReader(strings.Repeat("x", maxUpstreamErrorBodyBytes+1024))

	got, err := readUpstreamErrorBody(body)
	if err != nil {
		t.Fatalf("readUpstreamErrorBody returned error: %v", err)
	}

	if !strings.HasSuffix(got, upstreamErrorBodyTruncation) {
		t.Fatalf("expected truncated suffix, got length=%d suffix=%q", len(got), got[len(got)-min(len(got), len(upstreamErrorBodyTruncation)):])
	}
	if len(got) != maxUpstreamErrorBodyBytes+len(upstreamErrorBodyTruncation) {
		t.Fatalf("got length %d, want %d", len(got), maxUpstreamErrorBodyBytes+len(upstreamErrorBodyTruncation))
	}
}

func TestReadUpstreamErrorBodyKeepsSmallBody(t *testing.T) {
	const want = `{"error":"bad request"}`

	got, err := readUpstreamErrorBody(strings.NewReader(want))
	if err != nil {
		t.Fatalf("readUpstreamErrorBody returned error: %v", err)
	}
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
