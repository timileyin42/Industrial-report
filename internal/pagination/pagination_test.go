package pagination

import (
	"testing"
	"time"
)

func TestEncodeDecode_RoundTrip(t *testing.T) {
	want := Cursor{Time: time.Date(2026, 8, 10, 12, 30, 0, 0, time.UTC), Tiebreak: "ZG-0001"}

	token, err := Encode(want)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if token == "" {
		t.Fatal("expected a non-empty token")
	}

	got, err := Decode(token)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !got.Time.Equal(want.Time) || got.Tiebreak != want.Tiebreak {
		t.Fatalf("round trip mismatch: got %+v, want %+v", got, want)
	}
}

func TestDecode_RejectsEmptyToken(t *testing.T) {
	if _, err := Decode(""); err == nil {
		t.Fatal("expected an error decoding an empty cursor token, got none")
	}
}

func TestDecode_RejectsMalformedBase64(t *testing.T) {
	if _, err := Decode("not valid base64!!!"); err == nil {
		t.Fatal("expected an error decoding a malformed base64 token, got none")
	}
}

func TestDecode_RejectsValidBase64ButNotJSON(t *testing.T) {
	// "hello" base64url-encoded — decodes fine as base64, but isn't the
	// expected JSON cursor shape underneath.
	if _, err := Decode("aGVsbG8"); err == nil {
		t.Fatal("expected an error decoding base64 that isn't a valid cursor payload, got none")
	}
}

// TestEncodeDecode_DifferentCursorsProduceDifferentTokens guards against a
// degenerate Encode that ignores its input (e.g. always returning the same
// token) — every list endpoint relies on distinct rows producing distinct,
// individually resumable cursors.
func TestEncodeDecode_DifferentCursorsProduceDifferentTokens(t *testing.T) {
	a, err := Encode(Cursor{Time: time.Unix(1000, 0), Tiebreak: "a"})
	if err != nil {
		t.Fatalf("Encode a: %v", err)
	}
	b, err := Encode(Cursor{Time: time.Unix(2000, 0), Tiebreak: "b"})
	if err != nil {
		t.Fatalf("Encode b: %v", err)
	}
	if a == b {
		t.Fatal("expected different cursors to encode to different tokens")
	}
}
