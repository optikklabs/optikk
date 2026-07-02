package dsl

import "strings"

// FilterOp is the operator for a parsed filter.
type FilterOp string

const (
	OpEq          FilterOp = "eq"
	OpNeq         FilterOp = "neq"
	OpContains    FilterOp = "contains"
	OpNotContains FilterOp = "not_contains"
	OpIn          FilterOp = "in"
	OpNotIn       FilterOp = "not_in"
	OpGt          FilterOp = "gt"
	OpGte         FilterOp = "gte"
	OpLt          FilterOp = "lt"
	OpLte         FilterOp = "lte"
)

// Filter is a single parsed DSL filter (e.g. service:api → {Field:"service", Op:"eq", Value:"api"}).
type Filter struct {
	Field string
	Op    FilterOp
	Value string
}

// ParseError describes a problem at a specific offset in the DSL input.
type ParseError struct {
	Offset  int
	Length  int
	Message string
}

// ParseResult contains the parsed filters and any errors.
type ParseResult struct {
	Filters []Filter
	Errors  []ParseError
}

// Parse converts a Datadog-style DSL string into filters. This is a 1:1 Go
// port of web/src/features/explorer/search/parseDsl.ts.
func Parse(input string, fields []KnownField) ParseResult {
	var result ParseResult
	for _, tok := range Tokenize(input) {
		switch tok.Kind {
		case TokenBare:
			handleBare(tok, &result)
		case TokenQuoted:
			handleQuoted(tok, &result)
		case TokenKV:
			handleKV(tok, &result, fields)
		}
	}
	return result
}

func handleBare(tok Token, r *ParseResult) {
	if tok.Raw == "" {
		return
	}
	r.Filters = append(r.Filters, Filter{Field: "search", Op: OpContains, Value: tok.Raw})
}

func handleQuoted(tok Token, r *ParseResult) {
	if tok.Value == "" {
		return
	}
	r.Filters = append(r.Filters, Filter{Field: "search", Op: OpContains, Value: tok.Value})
}

func handleKV(tok Token, r *ParseResult, fields []KnownField) {
	keyRaw := tok.Key
	valueRaw := tok.Value
	negate := strings.HasPrefix(keyRaw, "-")
	key := keyRaw
	if negate {
		key = keyRaw[1:]
	}
	if key == "" || valueRaw == "" {
		r.Errors = append(r.Errors, ParseError{Offset: tok.Offset, Length: tok.Length, Message: "Empty field or value"})
		return
	}
	if !isValidField(key, fields) {
		r.Errors = append(r.Errors, ParseError{Offset: tok.Offset, Length: tok.Length, Message: "Unknown field \"" + key + "\""})
		return
	}
	parsed := parseValue(valueRaw)
	if parsed == nil {
		r.Errors = append(r.Errors, ParseError{Offset: tok.Offset, Length: tok.Length, Message: "Could not parse value"})
		return
	}
	r.Filters = append(r.Filters, Filter{Field: key, Op: effectiveOp(parsed.op, negate), Value: parsed.value})
}

type parsedValue struct {
	op    FilterOp
	value string
}

func parseValue(raw string) *parsedValue {
	if cmp := parseComparison(raw); cmp != nil {
		return cmp
	}
	if strings.HasPrefix(raw, "(") && strings.HasSuffix(raw, ")") {
		inner := raw[1 : len(raw)-1]
		// Split on " OR " (case-insensitive).
		parts := splitOR(inner)
		if len(parts) == 0 {
			return nil
		}
		return &parsedValue{op: OpIn, value: strings.Join(parts, ",")}
	}
	return &parsedValue{op: OpEq, value: raw}
}

func parseComparison(raw string) *parsedValue {
	if strings.HasPrefix(raw, ">=") {
		return &parsedValue{op: OpGte, value: raw[2:]}
	}
	if strings.HasPrefix(raw, "<=") {
		return &parsedValue{op: OpLte, value: raw[2:]}
	}
	if strings.HasPrefix(raw, ">") {
		return &parsedValue{op: OpGt, value: raw[1:]}
	}
	if strings.HasPrefix(raw, "<") {
		return &parsedValue{op: OpLt, value: raw[1:]}
	}
	return nil
}

func effectiveOp(op FilterOp, negate bool) FilterOp {
	if !negate {
		return op
	}
	switch op {
	case OpEq:
		return OpNeq
	case OpIn:
		return OpNotIn
	case OpContains:
		return OpNotContains
	default:
		return op
	}
}

func isValidField(key string, fields []KnownField) bool {
	// Custom attribute fields start with @.
	if strings.HasPrefix(key, "@") {
		return len(key) > 1
	}
	return FindKnownField(key, fields) != nil
}

// splitOR splits on " OR " (case-insensitive), trims, and drops empty parts.
func splitOR(s string) []string {
	// Use a case-insensitive split.
	var parts []string
	remaining := s
	for {
		idx := indexOR(remaining)
		if idx < 0 {
			p := strings.TrimSpace(remaining)
			if p != "" {
				parts = append(parts, p)
			}
			break
		}
		p := strings.TrimSpace(remaining[:idx])
		if p != "" {
			parts = append(parts, p)
		}
		remaining = remaining[idx+4:] // len(" OR ") == 4
	}
	return parts
}

func indexOR(s string) int {
	lower := strings.ToLower(s)
	return strings.Index(lower, " or ")
}
