// Package dsl implements a Datadog-style query DSL parser for the optikk CLI.
// It is a faithful Go port of the TypeScript tokenizer and parser used by the
// web UI (web/src/features/explorer/search/).
package dsl

// TokenKind classifies a DSL token.
type TokenKind string

const (
	TokenBare   TokenKind = "bare"
	TokenQuoted TokenKind = "quoted"
	TokenKV     TokenKind = "kv"
)

// Token is a single parsed element from a DSL input string.
type Token struct {
	Kind   TokenKind
	Offset int
	Length int
	Raw    string
	Key    string // populated for KV tokens
	Value  string // populated for KV and Quoted tokens
}

// Tokenize splits a DSL input string into tokens. This is a 1:1 Go port of
// web/src/features/explorer/search/tokenizeDsl.ts.
func Tokenize(input string) []Token {
	var out []Token
	i := 0
	for i < len(input) {
		ch := input[i]
		if ch == ' ' || ch == '\t' {
			i++
			continue
		}
		if ch == '"' {
			i = readQuoted(input, i, &out)
			continue
		}
		i = readBareOrKV(input, i, &out)
	}
	return out
}

func readQuoted(input string, start int, out *[]Token) int {
	j := start + 1
	for j < len(input) && input[j] != '"' {
		j++
	}
	endIncl := j
	if j < len(input) {
		endIncl = j + 1
	}
	*out = append(*out, Token{
		Kind:   TokenQuoted,
		Offset: start,
		Length: endIncl - start,
		Raw:    input[start:endIncl],
		Value:  input[start+1 : j],
	})
	return endIncl
}

func readBareOrKV(input string, start int, out *[]Token) int {
	j := start
	colonAt := -1
	inParens := false
	for j < len(input) {
		c := input[j]
		if (c == ' ' || c == '\t') && !inParens {
			break
		}
		if c == '(' {
			inParens = true
		} else if c == ')' {
			inParens = false
		} else if c == ':' && colonAt == -1 {
			colonAt = j
		}
		j++
	}
	raw := input[start:j]
	if colonAt > start {
		key := input[start:colonAt]
		value := input[colonAt+1 : j]
		*out = append(*out, Token{Kind: TokenKV, Offset: start, Length: j - start, Raw: raw, Key: key, Value: value})
	} else {
		*out = append(*out, Token{Kind: TokenBare, Offset: start, Length: j - start, Raw: raw})
	}
	return j
}
