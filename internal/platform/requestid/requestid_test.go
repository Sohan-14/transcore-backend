package requestid

import (
	"context"
	"encoding/hex"
	"strings"
	"testing"
)

func TestHeaderConstant(t *testing.T) {
	if Header != "X-Request-ID" {
		t.Fatalf("Header = %q, want %q", Header, "X-Request-ID")
	}
}

func TestContextRoundTrip(t *testing.T) {
	base := context.Background()

	if got := FromContext(base); got != "" {
		t.Fatalf("FromContext(empty) = %q, want empty string", got)
	}

	const wantID = "req-abc_123.test"
	ctx := NewContext(base, wantID)

	if got := FromContext(ctx); got != wantID {
		t.Fatalf("FromContext(ctx) = %q, want %q", got, wantID)
	}

	// Original context must remain untouched.
	if got := FromContext(base); got != "" {
		t.Fatalf("FromContext(base after NewContext) = %q, want empty string", got)
	}
}

func TestGenerate(t *testing.T) {
	id1 := Generate()
	id2 := Generate()

	if len(id1) != 32 {
		t.Fatalf("len(Generate()) = %d, want 32", len(id1))
	}
	if _, err := hex.DecodeString(id1); err != nil {
		t.Fatalf("Generate() = %q is not valid hex: %v", id1, err)
	}
	if id1 == id2 {
		t.Fatalf("Generate() returned duplicate IDs: %q", id1)
	}
	if !Valid(id1) {
		t.Fatalf("Generate() produced invalid ID according to Valid(): %q", id1)
	}
}

func TestValid(t *testing.T) {
	maxValid := strings.Repeat("a", maxLen)
	tooLong := strings.Repeat("a", maxLen+1)

	tests := []struct {
		name string
		id   string
		want bool
	}{
		{name: "empty", id: "", want: false},
		{name: "single char", id: "x", want: true},
		{name: "alphanumeric and allowed punctuation", id: "Req_01-abc.DEF", want: true},
		{name: "exact max length", id: maxValid, want: true},
		{name: "exceeds max length", id: tooLong, want: false},
		{name: "leading space", id: " abc123", want: false},
		{name: "trailing space", id: "abc123 ", want: false},
		{name: "newline log injection", id: "abc123\nlevel=error", want: false},
		{name: "carriage return", id: "abc123\r", want: false},
		{name: "slash or path characters", id: "abc/123", want: false},
		{name: "non-ascii unicode", id: "req-日本語", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Valid(tc.id); got != tc.want {
				t.Fatalf("Valid(%q) = %v, want %v", tc.id, got, tc.want)
			}
		})
	}
}
