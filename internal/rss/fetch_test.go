package rss

import (
	"testing"
)

func TestParsePubDate_CommonFormats(t *testing.T) {
	samples := []string{
		"Mon, 02 Jan 2006 15:04:05 -0700",
		"Mon, 02 Jan 2006 15:04:05 MST",
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02",
	}

	for _, s := range samples {
		if got, err := parsePubDate(s); err != nil {
			t.Errorf("parsePubDate(%q) returned error: %v", s, err)
		} else if got.IsZero() {
			t.Errorf("parsePubDate(%q) returned zero time", s)
		}
	}
}

func TestParsePubDate_Empty(t *testing.T) {
	if _, err := parsePubDate(""); err == nil {
		t.Fatalf("expected error for empty pubDate")
	}
}

func TestNewHTTPClient_HasTimeout(t *testing.T) {
	c := NewHTTPClient()
	if c == nil {
		t.Fatal("NewHTTPClient returned nil")
	}
	if c.Timeout == 0 {
		t.Fatalf("expected non-zero Timeout, got 0")
	}
}
