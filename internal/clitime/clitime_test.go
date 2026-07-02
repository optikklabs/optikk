package clitime

import (
	"testing"
	"time"
)

// reference point for deterministic tests.
var ref = time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)

func TestParseRelative(t *testing.T) {
	tests := []struct {
		input string
		want  int64
	}{
		{"1h", ref.Add(-1 * time.Hour).UnixMilli()},
		{"15m", ref.Add(-15 * time.Minute).UnixMilli()},
		{"7d", ref.Add(-7 * 24 * time.Hour).UnixMilli()},
		{"30s", ref.Add(-30 * time.Second).UnixMilli()},
		{"2w", ref.Add(-2 * 7 * 24 * time.Hour).UnixMilli()},
	}
	for _, tt := range tests {
		got, err := Parse(tt.input, ref)
		if err != nil {
			t.Errorf("Parse(%q) error: %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("Parse(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestParseNow(t *testing.T) {
	for _, input := range []string{"now", "NOW", "", " now "} {
		got, err := Parse(input, ref)
		if err != nil {
			t.Errorf("Parse(%q) error: %v", input, err)
			continue
		}
		if got != ref.UnixMilli() {
			t.Errorf("Parse(%q) = %d, want %d", input, got, ref.UnixMilli())
		}
	}
}

func TestParseISO8601(t *testing.T) {
	tests := []struct {
		input string
		want  int64
	}{
		{"2024-01-01T00:00:00Z", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()},
		{"2024-01-01", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()},
	}
	for _, tt := range tests {
		got, err := Parse(tt.input, ref)
		if err != nil {
			t.Errorf("Parse(%q) error: %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("Parse(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestParseEpochMs(t *testing.T) {
	got, err := Parse("1719936000000", ref)
	if err != nil {
		t.Fatal(err)
	}
	if got != 1719936000000 {
		t.Errorf("got %d, want 1719936000000", got)
	}
}

func TestParseRangeDefaults(t *testing.T) {
	start, end, err := ParseRange("", "", ref)
	if err != nil {
		t.Fatal(err)
	}
	if end != ref.UnixMilli() {
		t.Errorf("end = %d, want %d", end, ref.UnixMilli())
	}
	wantStart := ref.Add(-1 * time.Hour).UnixMilli()
	if start != wantStart {
		t.Errorf("start = %d, want %d", start, wantStart)
	}
}

func TestParseRangeInvalid(t *testing.T) {
	// end before start
	_, _, err := ParseRange("15m", "1h", ref)
	if err == nil {
		t.Error("expected error for end < start")
	}
}

func TestParseInvalid(t *testing.T) {
	for _, input := range []string{"xyz", "ago", "-1h"} {
		_, err := Parse(input, ref)
		if err == nil {
			t.Errorf("Parse(%q) expected error", input)
		}
	}
}
