package oktaexpr

import "fmt"

type tokenType int

const (
	tokEOF tokenType = iota
	tokName
	tokString
	tokInt
	tokFloat
	tokPlus
	tokNE
	tokGTE
	tokLTE
	tokGT
	tokLT
	tokEQ
	tokAnd
	tokOr
	tokNot
	tokNull
	tokBool
	tokColon
	tokQuestion
	tokMemberOf
	tokMemberOfAny
	tokMemberOfName
	tokMemberOfGroupStartsWith
	tokMemberOfGroupContains
	tokMemberOfGroupNameRegex
	tokUser
	tokClass
	tokLParen
	tokRParen
	tokComma
	tokDot
	tokLBrace
	tokRBrace
)

type token struct {
	typ   tokenType
	value string
	pos   int
}

// keywords holds identifier text that resolves to a fixed token rather than
// a NAME, matching the exact case variants recognized by the source grammar.
//
// The Python source matched these via regex alternation tried in priority
// order without word boundaries, which meant a keyword prefix inside a
// longer identifier (e.g. "nested" starting with "ne") was misread as the
// keyword followed by a truncated NAME. Scanning the full identifier greedily
// first and only then checking for an exact keyword match (below) fixes that
// bug while preserving every other behavior.
var keywords = map[string]tokenType{
	"and": tokAnd, "And": tokAnd, "AND": tokAnd,
	"or": tokOr, "Or": tokOr, "OR": tokOr,
	"not": tokNot, "Not": tokNot, "NOT": tokNot,
	"ne": tokNE, "Ne": tokNE, "NE": tokNE,
	"eq": tokEQ, "Eq": tokEQ, "EQ": tokEQ,
	"true": tokBool, "True": tokBool, "TRUE": tokBool,
	"false": tokBool, "False": tokBool, "FALSE": tokBool,
	"null": tokNull, "Null": tokNull, "NULL": tokNull,
	"user":                          tokUser,
	"isMemberOfGroupNameContains":   tokMemberOfGroupContains,
	"isMemberOfGroupNameRegex":      tokMemberOfGroupNameRegex,
	"isMemberOfGroupNameStartsWith": tokMemberOfGroupStartsWith,
	"isMemberOfAnyGroup":            tokMemberOfAny,
	"isMemberOfGroupName":           tokMemberOfName,
	"isMemberOfGroup":               tokMemberOf,
	"String":                        tokClass,
	"Arrays":                        tokClass,
	"Convert":                       tokClass,
	"Iso3166Convert":                tokClass,
	"Groups":                        tokClass,
}

func isNameStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isNameContinue(c byte) bool {
	return isNameStart(c) || (c >= '0' && c <= '9') || c == '-'
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

// tokenize scans an Okta expression into a flat token slice.
func tokenize(input string) ([]token, error) {
	var toks []token
	i := 0
	n := len(input)

	for i < n {
		c := input[i]

		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			i++
			continue
		}

		start := i

		switch {
		case c == '(':
			toks = append(toks, token{tokLParen, "(", start})
			i++
		case c == ')':
			toks = append(toks, token{tokRParen, ")", start})
			i++
		case c == ',':
			toks = append(toks, token{tokComma, ",", start})
			i++
		case c == '.':
			toks = append(toks, token{tokDot, ".", start})
			i++
		case c == '{':
			toks = append(toks, token{tokLBrace, "{", start})
			i++
		case c == '}':
			toks = append(toks, token{tokRBrace, "}", start})
			i++
		case c == ':':
			toks = append(toks, token{tokColon, ":", start})
			i++
		case c == '?':
			toks = append(toks, token{tokQuestion, "?", start})
			i++
		case c == '"' || c == '\'':
			quote := c
			j := i + 1
			for j < n && input[j] != quote {
				if input[j] == '\\' && j+1 < n {
					j += 2
					continue
				}
				j++
			}
			if j >= n {
				return nil, fmt.Errorf("unterminated string literal starting at character %d", start)
			}
			value := input[i : j+1]
			toks = append(toks, token{tokString, value, start})
			i = j + 1
		case c == '!':
			if i+1 < n && input[i+1] == '=' {
				toks = append(toks, token{tokNE, "!=", start})
				i += 2
			} else {
				toks = append(toks, token{tokNot, "!", start})
				i++
			}
		case c == '>':
			if i+1 < n && input[i+1] == '=' {
				toks = append(toks, token{tokGTE, ">=", start})
				i += 2
			} else {
				toks = append(toks, token{tokGT, ">", start})
				i++
			}
		case c == '<':
			if i+1 < n && input[i+1] == '=' {
				toks = append(toks, token{tokLTE, "<=", start})
				i += 2
			} else {
				toks = append(toks, token{tokLT, "<", start})
				i++
			}
		case c == '=':
			if i+1 < n && input[i+1] == '=' {
				toks = append(toks, token{tokEQ, "==", start})
				i += 2
			} else {
				return nil, fmt.Errorf("bad character '=' at character %d", start)
			}
		case c == '&':
			if i+1 < n && input[i+1] == '&' {
				toks = append(toks, token{tokAnd, "&&", start})
				i += 2
			} else {
				return nil, fmt.Errorf("bad character '&' at character %d", start)
			}
		case c == '|':
			if i+1 < n && input[i+1] == '|' {
				toks = append(toks, token{tokOr, "||", start})
				i += 2
			} else {
				return nil, fmt.Errorf("bad character '|' at character %d", start)
			}
		case c == '+':
			toks = append(toks, token{tokPlus, "+", start})
			i++
		case isDigit(c):
			j := i
			for j < n && isDigit(input[j]) {
				j++
			}
			typ := tokInt
			// A "." is only consumed as a decimal point when followed by at
			// least one more digit, so a bare trailing "." (or a "." that
			// starts a path-chain access, which never follows a digit in
			// this grammar) is left for the caller to handle separately.
			if j < n && input[j] == '.' && j+1 < n && isDigit(input[j+1]) {
				typ = tokFloat
				j++
				for j < n && isDigit(input[j]) {
					j++
				}
			}
			toks = append(toks, token{typ, input[i:j], start})
			i = j
		case isNameStart(c):
			j := i
			for j < n && isNameContinue(input[j]) {
				j++
			}
			text := input[i:j]
			if typ, ok := keywords[text]; ok {
				toks = append(toks, token{typ, text, start})
			} else {
				toks = append(toks, token{tokName, text, start})
			}
			i = j
		default:
			return nil, fmt.Errorf("bad character %q at character %d", c, start)
		}
	}

	toks = append(toks, token{tokEOF, "", n})
	return toks, nil
}
