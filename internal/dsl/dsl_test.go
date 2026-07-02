package dsl

import (
	"testing"
)

func TestTokenize(t *testing.T) {
	tests := []struct {
		input string
		want  []Token
	}{
		{
			input: `service:api`,
			want:  []Token{{Kind: TokenKV, Offset: 0, Length: 11, Raw: "service:api", Key: "service", Value: "api"}},
		},
		{
			input: `"connection timeout"`,
			want:  []Token{{Kind: TokenQuoted, Offset: 0, Length: 20, Raw: `"connection timeout"`, Value: "connection timeout"}},
		},
		{
			input: `service:api status:error`,
			want: []Token{
				{Kind: TokenKV, Offset: 0, Length: 11, Raw: "service:api", Key: "service", Value: "api"},
				{Kind: TokenKV, Offset: 12, Length: 12, Raw: "status:error", Key: "status", Value: "error"},
			},
		},
		{
			input: `service:(api OR web)`,
			want:  []Token{{Kind: TokenKV, Offset: 0, Length: 19, Raw: "service:(api OR web)", Key: "service", Value: "(api OR web)"}},
		},
		{
			input: `freetext`,
			want:  []Token{{Kind: TokenBare, Offset: 0, Length: 8, Raw: "freetext"}},
		},
	}
	for _, tt := range tests {
		got := Tokenize(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("Tokenize(%q): got %d tokens, want %d", tt.input, len(got), len(tt.want))
			continue
		}
		for i := range got {
			if got[i].Kind != tt.want[i].Kind || got[i].Key != tt.want[i].Key || got[i].Value != tt.want[i].Value {
				t.Errorf("Tokenize(%q)[%d] = %+v, want %+v", tt.input, i, got[i], tt.want[i])
			}
		}
	}
}

func TestParseTraceFilters(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		check   func(ParseResult) string
	}{
		{
			name:  "simple eq",
			input: "service:api",
			check: func(r ParseResult) string {
				if len(r.Errors) > 0 {
					return r.Errors[0].Message
				}
				if len(r.Filters) != 1 || r.Filters[0].Field != "service" || r.Filters[0].Op != OpEq || r.Filters[0].Value != "api" {
					return "unexpected filter"
				}
				return ""
			},
		},
		{
			name:  "negated",
			input: "-service:checkout",
			check: func(r ParseResult) string {
				if len(r.Filters) != 1 || r.Filters[0].Op != OpNeq {
					return "expected neq"
				}
				return ""
			},
		},
		{
			name:  "OR group",
			input: "service:(api OR web)",
			check: func(r ParseResult) string {
				if len(r.Filters) != 1 || r.Filters[0].Op != OpIn || r.Filters[0].Value != "api,web" {
					return "expected in filter with api,web"
				}
				return ""
			},
		},
		{
			name:  "comparison gte",
			input: "duration_ms:>=500",
			check: func(r ParseResult) string {
				if len(r.Filters) != 1 || r.Filters[0].Op != OpGte || r.Filters[0].Value != "500" {
					return "expected gte 500"
				}
				return ""
			},
		},
		{
			name:  "custom attribute",
			input: "@http.route:/api/v1",
			check: func(r ParseResult) string {
				if len(r.Filters) != 1 || r.Filters[0].Field != "@http.route" || r.Filters[0].Value != "/api/v1" {
					return "expected attribute filter"
				}
				return ""
			},
		},
		{
			name:  "quoted free text",
			input: `"connection timeout"`,
			check: func(r ParseResult) string {
				if len(r.Filters) != 1 || r.Filters[0].Field != "search" || r.Filters[0].Value != "connection timeout" {
					return "expected search filter"
				}
				return ""
			},
		},
		{
			name:  "unknown field",
			input: "bogus:value",
			check: func(r ParseResult) string {
				if len(r.Errors) != 1 {
					return "expected 1 error"
				}
				return ""
			},
		},
		{
			name:  "boolean field",
			input: "has_error:true",
			check: func(r ParseResult) string {
				if len(r.Filters) != 1 || r.Filters[0].Field != "has_error" || r.Filters[0].Value != "true" {
					return "expected has_error filter"
				}
				return ""
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Parse(tt.input, TraceKnownFields)
			if msg := tt.check(result); msg != "" {
				t.Errorf("Parse(%q): %s — got filters=%+v errors=%+v", tt.input, msg, result.Filters, result.Errors)
			}
		})
	}
}

func TestMapToTraceFilters(t *testing.T) {
	result := Parse("service:api has_error:true duration_ms:>=500 @http.route:/api", TraceKnownFields)
	tf := MapToTraceFilters(result.Filters)
	if len(tf.Services) != 1 || tf.Services[0] != "api" {
		t.Errorf("Services = %v", tf.Services)
	}
	if tf.HasError == nil || !*tf.HasError {
		t.Error("HasError should be true")
	}
	if tf.MinDurationNs != 500_000_000 {
		t.Errorf("MinDurationNs = %d, want 500000000", tf.MinDurationNs)
	}
	if len(tf.Attributes) != 1 || tf.Attributes[0].Key != "http.route" {
		t.Errorf("Attributes = %v", tf.Attributes)
	}
}

func TestMapToLogFilters(t *testing.T) {
	result := Parse("severity_text:ERROR service_name:api", LogKnownFields)
	lf := MapToLogFilters(result.Filters)
	if len(lf.Severities) != 1 || lf.Severities[0] != "ERROR" {
		t.Errorf("Severities = %v", lf.Severities)
	}
	if len(lf.Services) != 1 || lf.Services[0] != "api" {
		t.Errorf("Services = %v", lf.Services)
	}
}
