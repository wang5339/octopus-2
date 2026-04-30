package responsebody

import (
	"strings"
	"testing"
)

func TestReadAllRejectsBodyOverLimit(t *testing.T) {
	oldMaxBytes := MaxBytes
	MaxBytes = 8
	t.Cleanup(func() {
		MaxBytes = oldMaxBytes
	})

	_, err := ReadAll(strings.NewReader("123456789"))
	if err == nil {
		t.Fatal("expected body size error")
	}
	if err.Error() != "upstream response body exceeds 8 byte limit" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReadAllAllowsBodyAtLimit(t *testing.T) {
	oldMaxBytes := MaxBytes
	MaxBytes = 8
	t.Cleanup(func() {
		MaxBytes = oldMaxBytes
	})

	got, err := ReadAll(strings.NewReader("12345678"))
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}
	if string(got) != "12345678" {
		t.Fatalf("got %q", string(got))
	}
}
