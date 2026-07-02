// Package clitime parses Datadog-compatible time specifications for --from
// and --to CLI flags. Accepted formats:
//
//   - Relative durations: "1h", "15m", "7d", "30s"
//   - Special keyword:    "now"
//   - ISO 8601:           "2024-01-01T00:00:00Z"
//   - Epoch milliseconds: "1719936000000"
package clitime

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Parse converts a time specification into epoch milliseconds. An empty input
// is treated as "now".
func Parse(input string, now time.Time) (int64, error) {
	input = strings.TrimSpace(input)
	if input == "" || strings.EqualFold(input, "now") {
		return now.UnixMilli(), nil
	}

	// Try relative duration first (e.g. "1h", "15m", "7d").
	if dur, ok := parseRelative(input); ok {
		return now.Add(-dur).UnixMilli(), nil
	}

	// Try epoch milliseconds (all digits, at least 10 chars to avoid ambiguity).
	if len(input) >= 10 {
		if ms, err := strconv.ParseInt(input, 10, 64); err == nil {
			return ms, nil
		}
	}

	// Try ISO 8601 formats.
	for _, layout := range []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05",
		"2006-01-02",
	} {
		if t, err := time.Parse(layout, input); err == nil {
			return t.UnixMilli(), nil
		}
	}

	return 0, fmt.Errorf("unrecognised time %q (use relative like 1h, ISO 8601, or epoch ms)", input)
}

// ParseRange parses both --from and --to into epoch-ms. Defaults: from=1h, to=now.
func ParseRange(from, to string, now time.Time) (startMs, endMs int64, err error) {
	if from == "" {
		from = "1h"
	}
	if to == "" {
		to = "now"
	}
	startMs, err = Parse(from, now)
	if err != nil {
		return 0, 0, fmt.Errorf("--from: %w", err)
	}
	endMs, err = Parse(to, now)
	if err != nil {
		return 0, 0, fmt.Errorf("--to: %w", err)
	}
	if endMs <= startMs {
		return 0, 0, fmt.Errorf("--to (%d) must be after --from (%d)", endMs, startMs)
	}
	return startMs, endMs, nil
}

// parseRelative handles suffixed duration strings like "1h", "15m", "7d", "30s".
func parseRelative(s string) (time.Duration, bool) {
	if len(s) < 2 {
		return 0, false
	}
	numPart := s[:len(s)-1]
	suffix := s[len(s)-1]

	n, err := strconv.ParseFloat(numPart, 64)
	if err != nil || n < 0 {
		return 0, false
	}

	switch suffix {
	case 's':
		return time.Duration(n * float64(time.Second)), true
	case 'm':
		return time.Duration(n * float64(time.Minute)), true
	case 'h':
		return time.Duration(n * float64(time.Hour)), true
	case 'd':
		return time.Duration(n * 24 * float64(time.Hour)), true
	case 'w':
		return time.Duration(n * 7 * 24 * float64(time.Hour)), true
	default:
		return 0, false
	}
}
