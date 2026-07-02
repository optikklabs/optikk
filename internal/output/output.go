// Package output formats CLI results as table, JSON, or YAML. It implements
// Datadog's TTY auto-detection: table for interactive terminals, NDJSON when
// piped. The --output flag or OPTIKK_OUTPUT env var override auto-detection.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"go.yaml.in/yaml/v3"
)

// Format is the output serialization format.
type Format string

const (
	FormatTable Format = "table"
	FormatJSON  Format = "json"
	FormatYAML  Format = "yaml"
	FormatAuto  Format = "" // resolved at runtime
)

// Resolve determines the output format using Datadog Pup's auto-detection:
//  1. Explicit --output flag or OPTIKK_OUTPUT env (highest priority)
//  2. --agent flag → json
//  3. TTY detection: terminal → table, piped → json
func Resolve(explicit string, isAgent bool) Format {
	if explicit != "" {
		return Format(strings.ToLower(explicit))
	}
	if isAgent {
		return FormatJSON
	}
	if isTerminal(os.Stdout) {
		return FormatTable
	}
	return FormatJSON
}

// isTerminal checks if a file descriptor is connected to a terminal.
func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// Writer writes formatted output to a destination.
type Writer struct {
	Format Format
	Out    io.Writer
}

// New creates a Writer with the resolved format writing to the given output.
func New(format Format, out io.Writer) *Writer {
	return &Writer{Format: format, Out: out}
}

// WriteItems renders a list of items in the configured format. For table
// output, rowFn extracts column values from each item.
func (w *Writer) WriteItems(headers []string, items []any, rowFn func(any) []string) error {
	switch w.Format {
	case FormatJSON:
		return w.WriteJSON(items)
	case FormatYAML:
		return w.WriteYAML(items)
	default:
		rows := make([][]string, len(items))
		for i, item := range rows {
			_ = item
			rows[i] = rowFn(items[i])
		}
		return w.WriteTable(headers, rows)
	}
}

// WriteOne renders a single item.
func (w *Writer) WriteOne(headers []string, item any, rowFn func(any) []string) error {
	switch w.Format {
	case FormatJSON:
		return w.WriteJSON(item)
	case FormatYAML:
		return w.WriteYAML(item)
	default:
		return w.WriteTable(headers, [][]string{rowFn(item)})
	}
}

// WriteTable renders a columnar table with headers.
func (w *Writer) WriteTable(headers []string, rows [][]string) error {
	tw := tabwriter.NewWriter(w.Out, 0, 4, 2, ' ', 0)
	// Header in bold.
	headerLine := make([]string, len(headers))
	for i, h := range headers {
		headerLine[i] = "\033[1m" + h + "\033[0m"
	}
	fmt.Fprintln(tw, strings.Join(headerLine, "\t"))
	for _, row := range rows {
		fmt.Fprintln(tw, strings.Join(row, "\t"))
	}
	return tw.Flush()
}

// WriteJSON writes v as indented JSON.
func (w *Writer) WriteJSON(v any) error {
	enc := json.NewEncoder(w.Out)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// WriteYAML writes v as YAML.
func (w *Writer) WriteYAML(v any) error {
	return yaml.NewEncoder(w.Out).Encode(v)
}

// Msg prints a status message (used for non-data responses like "deleted monitor 5").
// It is suppressed when outputting structured formats (JSON/YAML) to avoid breaking parsers.
func (w *Writer) Msg(format string, args ...any) {
	if w.Format == FormatJSON || w.Format == FormatYAML {
		return
	}
	fmt.Fprintf(w.Out, format+"\n", args...)
}
