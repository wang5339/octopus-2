package op

import "testing"

func TestAPIKeyHashAndMask(t *testing.T) {
	raw := "sk-octopus-1234567890abcdef"
	hash := APIKeyHash(raw)
	if hash == "" || hash == raw {
		t.Fatalf("hash should be non-empty and different from raw key, got %q", hash)
	}
	if APIKeyHash(raw) != hash {
		t.Fatalf("hash should be deterministic")
	}

	masked := APIKeyMask(raw)
	if masked == raw {
		t.Fatalf("masked key should not equal raw key")
	}
	if masked != "sk-octopus-********cdef" {
		t.Fatalf("unexpected masked key: %q", masked)
	}
}
