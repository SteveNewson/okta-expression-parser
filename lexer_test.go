package oktaexpr

import "testing"

func TestTokenize_Types(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  []tokenType
	}{
		{"and keyword", "and", []tokenType{tokAnd, tokEOF}},
		{"AND keyword", "AND", []tokenType{tokAnd, tokEOF}},
		{"double ampersand", "&&", []tokenType{tokAnd, tokEOF}},
		{"or keyword", "or", []tokenType{tokOr, tokEOF}},
		{"double pipe", "||", []tokenType{tokOr, tokEOF}},
		{"not keyword", "not", []tokenType{tokNot, tokEOF}},
		{"bang", "!", []tokenType{tokNot, tokEOF}},
		{"bang equal", "!=", []tokenType{tokNE, tokEOF}},
		{"ne keyword", "ne", []tokenType{tokNE, tokEOF}},
		{"eq keyword", "eq", []tokenType{tokEQ, tokEOF}},
		{"double equal", "==", []tokenType{tokEQ, tokEOF}},
		{"greater than", ">", []tokenType{tokGT, tokEOF}},
		{"greater or equal", ">=", []tokenType{tokGTE, tokEOF}},
		{"less than", "<", []tokenType{tokLT, tokEOF}},
		{"less or equal", "<=", []tokenType{tokLTE, tokEOF}},
		{"integer", "123", []tokenType{tokInt, tokEOF}},
		{"string literal", `"abc"`, []tokenType{tokString, tokEOF}},
		{"true literal", "true", []tokenType{tokBool, tokEOF}},
		{"false literal", "FALSE", []tokenType{tokBool, tokEOF}},
		{"null literal", "null", []tokenType{tokNull, tokEOF}},
		{"user keyword", "user", []tokenType{tokUser, tokEOF}},
		{"plain name", "foo", []tokenType{tokName, tokEOF}},
		{"name with hyphen", "employee-id", []tokenType{tokName, tokEOF}},
		{"class name", "String", []tokenType{tokClass, tokEOF}},
		{"isMemberOfGroup", `isMemberOfGroup("x")`, []tokenType{tokMemberOf, tokLParen, tokString, tokRParen, tokEOF}},
		{"isMemberOfAnyGroup", `isMemberOfAnyGroup("x")`, []tokenType{tokMemberOfAny, tokLParen, tokString, tokRParen, tokEOF}},
		{"isMemberOfGroupName", `isMemberOfGroupName("x")`, []tokenType{tokMemberOfName, tokLParen, tokString, tokRParen, tokEOF}},
		{"isMemberOfGroupNameStartsWith", `isMemberOfGroupNameStartsWith("x")`, []tokenType{tokMemberOfGroupStartsWith, tokLParen, tokString, tokRParen, tokEOF}},
		{"isMemberOfGroupNameContains", `isMemberOfGroupNameContains("x")`, []tokenType{tokMemberOfGroupContains, tokLParen, tokString, tokRParen, tokEOF}},
		{"isMemberOfGroupNameRegex", `isMemberOfGroupNameRegex("x")`, []tokenType{tokMemberOfGroupNameRegex, tokLParen, tokString, tokRParen, tokEOF}},
		{"literals", "(),.{}",
			[]tokenType{tokLParen, tokRParen, tokComma, tokDot, tokLBrace, tokRBrace, tokEOF}},
		{"question and colon", "?:", []tokenType{tokQuestion, tokColon, tokEOF}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// When
			toks, err := tokenize(tc.input)

			// Then
			if err != nil {
				t.Fatalf("tokenize(%q): unexpected error %v", tc.input, err)
			}
			if len(toks) != len(tc.want) {
				t.Fatalf("tokenize(%q): got %d tokens %v, want %d tokens %v", tc.input, len(toks), toks, len(tc.want), tc.want)
			}
			for i, want := range tc.want {
				if toks[i].typ != want {
					t.Errorf("tokenize(%q)[%d]: got type %d (%q), want type %d", tc.input, i, toks[i].typ, toks[i].value, want)
				}
			}
		})
	}
}

// TestTokenize_KeywordPrefixInsideIdentifier guards against a real bug in
// the Python source: its keyword regexes had no word-boundary anchor, so an
// identifier that merely started with a keyword-like prefix (e.g. "nested"
// starts with "ne") was misread as that keyword followed by a truncated
// NAME. user.nested demonstrably fails to parse in the Python source for
// exactly this reason.
func TestTokenize_KeywordPrefixInsideIdentifier(t *testing.T) {
	t.Parallel()

	tests := []string{
		"nested",       // starts with "ne" (NE)
		"android",      // starts with "and" (AND)
		"notification", // starts with "not" (NOT)
		"truely",       // starts with "true" (BOOL)
		"usercount",    // starts with "user" (USER); the Python source got this one right via \b
		"organization", // starts with "or" (OR)
		"equity",       // starts with "eq" (EQ)
		"nullable",     // starts with "null" (NULL)
	}

	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// When
			toks, err := tokenize(name)

			// Then
			if err != nil {
				t.Fatalf("tokenize(%q): unexpected error %v", name, err)
			}
			if len(toks) != 2 || toks[0].typ != tokName || toks[0].value != name {
				t.Errorf("tokenize(%q): got %v, want a single NAME token %q", name, toks, name)
			}
		})
	}
}

func TestTokenize_StringLiteral(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string // raw token value, including quotes
	}{
		{"simple string", `"hello"`, `"hello"`},
		{"empty string", `""`, `""`},
		{"escaped quote inside string", `"a\"b"`, `"a\"b"`},
		{"escaped backslash inside string", `"a\\b"`, `"a\\b"`},
		{"single-quoted string", `'hello'`, `'hello'`},
		{"single-quoted empty string", `''`, `''`},
		{"single-quoted string containing a double quote", `'a"b'`, `'a"b'`},
		{"escaped single quote inside single-quoted string", `'a\'b'`, `'a\'b'`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// When
			toks, err := tokenize(tc.input)

			// Then
			if err != nil {
				t.Fatalf("tokenize(%q): unexpected error %v", tc.input, err)
			}
			if len(toks) != 2 || toks[0].typ != tokString {
				t.Fatalf("tokenize(%q): got %v, want a single STRING token", tc.input, toks)
			}
			if toks[0].value != tc.want {
				t.Errorf("tokenize(%q): got value %q, want %q", tc.input, toks[0].value, tc.want)
			}
		})
	}
}

func TestTokenize_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{"unterminated string", `"unterminated`},
		{"unterminated single-quoted string", `'unterminated`},
		{"bare ampersand", "1 & 1"},
		{"bare pipe", "1 | 1"},
		{"bare equals", "1 = 1"},
		{"unknown character", "1 ~ 1"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// When
			_, err := tokenize(tc.input)

			// Then
			if err == nil {
				t.Errorf("tokenize(%q): got nil error, want an error", tc.input)
			}
		})
	}
}

func TestTokenize_IgnoresWhitespace(t *testing.T) {
	t.Parallel()

	// When
	toks, err := tokenize("1   ==\t2\n")

	// Then
	if err != nil {
		t.Fatalf("tokenize: unexpected error %v", err)
	}
	want := []tokenType{tokInt, tokEQ, tokInt, tokEOF}
	if len(toks) != len(want) {
		t.Fatalf("tokenize: got %d tokens %v, want %d", len(toks), toks, len(want))
	}
	for i, w := range want {
		if toks[i].typ != w {
			t.Errorf("tokenize[%d]: got type %d, want %d", i, toks[i].typ, w)
		}
	}
}
