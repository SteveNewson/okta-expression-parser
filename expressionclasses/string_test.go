package expressionclasses_test

import (
	"testing"

	"github.com/stevenewson/okta-expression-parser/expressionclasses"
)

func TestString_Call(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		method string
		args   []any
		want   any
	}{
		{"len", "len", []any{"This"}, 4},
		{"len empty string", "len", []any{""}, 0},
		{"stringContains true", "stringContains", []any{"hello", "ell"}, true},
		{"stringContains false", "stringContains", []any{"hello", "xyz"}, false},
		{"stringContains non-string is false, not an error", "stringContains", []any{"hello", 5}, false},
		{"startsWith true", "startsWith", []any{"hello", "he"}, true},
		{"startsWith false", "startsWith", []any{"hello", "lo"}, false},
		{"toLowerCase", "toLowerCase", []any{"ABC"}, "abc"},
		{"toLowerCase passes through non-string", "toLowerCase", []any{5}, 5},
		{"toUpperCase", "toUpperCase", []any{"abc"}, "ABC"},
		{"append", "append", []any{"a", "b"}, "a$b"},
		{"join multiple", "join", []any{",", "a", "b", "c"}, "a,b,c"},
		{"join single", "join", []any{",", "a"}, "a"},
		{"removeSpaces", "removeSpaces", []any{"a b c"}, "abc"},
		{"replace all occurrences", "replace", []any{"aaa", "a", "b"}, "bbb"},
		{"replaceFirst only first occurrence", "replaceFirst", []any{"aaa", "a", "b"}, "baa"},
		{"contains true", "contains", []any{"hello", "ell"}, true},
		{"contains false", "contains", []any{"hello", "xyz"}, false},
		{"stringSwitch matches first key", "stringSwitch", []any{"hello", "default", "he", "matched"}, "matched"},
		{"stringSwitch falls through to default", "stringSwitch", []any{"hello", "default", "zz", "matched"}, "default"},
		{"stringSwitch odd kv pairs returns false", "stringSwitch", []any{"hello", "default", "he"}, false},
		{"substring", "substring", []any{"hello", 1, 3}, "el"},
		{"substring end beyond length clamps to len-1", "substring", []any{"hello", 0, 100}, "hell"},
		{"substringAfter found", "substringAfter", []any{"hello", "l"}, "lo"},
		{"substringAfter multi-char delimiter excluded from result", "substringAfter", []any{"abc@okta.com", "@"}, "okta.com"},
		{"substringAfter not found returns val unchanged", "substringAfter", []any{"hello", "z"}, "hello"},
		{"substringBefore found", "substringBefore", []any{"hello", "l"}, "he"},
		{"substringBefore not found returns val unchanged", "substringBefore", []any{"hello", "z"}, "hello"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Given
			s := expressionclasses.String{}

			// When
			got, err := s.Call(tc.method, tc.args...)

			// Then
			if err != nil {
				t.Fatalf("String.%s(%#v): unexpected error %v", tc.method, tc.args, err)
			}
			if got != tc.want {
				t.Errorf("String.%s(%#v): got %#v, want %#v", tc.method, tc.args, got, tc.want)
			}
		})
	}
}

func TestString_Substring_NonStringReturnsEmptyString(t *testing.T) {
	t.Parallel()

	// Given
	s := expressionclasses.String{}

	// When
	got, err := s.Call("substring", 5, 0, 1)

	// Then
	if err != nil {
		t.Fatalf("String.substring(5, 0, 1): unexpected error %v", err)
	}
	if got != "" {
		t.Errorf("String.substring(5, 0, 1): got %#v, want empty string", got)
	}
}

func TestString_Join_NonStringArgIsError(t *testing.T) {
	t.Parallel()

	// Given
	s := expressionclasses.String{}

	// When
	_, err := s.Call("join", ",", "a", 5)

	// Then
	if err == nil {
		t.Errorf(`String.join(",", "a", 5): got nil error, want an error`)
	}
}

func TestString_Len_NonStringIsError(t *testing.T) {
	t.Parallel()

	// Given
	s := expressionclasses.String{}

	// When
	_, err := s.Call("len", 5)

	// Then
	if err == nil {
		t.Errorf(`String.len(5): got nil error, want an error`)
	}
}
