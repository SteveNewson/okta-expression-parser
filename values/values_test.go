package values_test

import (
	"testing"

	"github.com/stevenewson/okta-expression-parser/values"
)

func TestTruthy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   any
		want bool
	}{
		{"nil is falsy", nil, false},
		{"true is truthy", true, true},
		{"false is falsy", false, false},
		{"zero int is falsy", 0, false},
		{"nonzero int is truthy", 1, true},
		{"negative int is truthy", -1, true},
		{"zero float is falsy", float64(0), false},
		{"nonzero float is truthy", float64(0.5), true},
		{"empty string is falsy", "", false},
		{"nonempty string is truthy", "x", true},
		{"empty array is falsy", values.Array{}, false},
		{"nonempty array is truthy", values.Array{1}, true},
		{"empty tuple is falsy", values.Tuple{}, false},
		{"nonempty tuple is truthy", values.Tuple{1}, true},
		{"empty map is falsy", map[string]any{}, false},
		{"nonempty map is truthy", map[string]any{"a": 1}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// When
			got := values.Truthy(tc.in)

			// Then
			if got != tc.want {
				t.Errorf("Truthy(%#v): got %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestEqualOperands(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a, b any
		want bool
	}{
		{"equal ints", 1, 1, true},
		{"unequal ints", 1, 2, false},
		{"equal strings", "a", "a", true},
		{"unequal strings", "a", "b", false},
		{"int vs string never equal", 1, "1", false},
		{"bool true equals int 1", true, 1, true},
		{"int 1 equals bool true", 1, true, true},
		{"bool false equals int 0", false, 0, true},
		{"bool true does not equal string true", true, "true", false},
		{"nil equals nil", nil, nil, true},
		{"nil does not equal false", nil, false, false},
		{"nil does not equal zero", nil, 0, false},
		{"equal arrays", values.Array{1, 2}, values.Array{1, 2}, true},
		{"unequal length arrays", values.Array{1, 2}, values.Array{1}, false},
		{"unequal arrays", values.Array{1, 2}, values.Array{1, 3}, false},
		{"array does not equal tuple", values.Array{1, 2}, values.Tuple{1, 2}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// When
			got := values.EqualOperands(tc.a, tc.b)

			// Then
			if got != tc.want {
				t.Errorf("EqualOperands(%#v, %#v): got %v, want %v", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func TestCompareOperands_MismatchedTypesReturnFalseWithoutError(t *testing.T) {
	t.Parallel()

	// Given: bool and int are NOT the same type for ordering, even though
	// EqualOperands treats them as numerically comparable.
	// When
	got, err := values.CompareOperands(">=", true, 1)

	// Then
	if err != nil {
		t.Fatalf("CompareOperands(true, 1): unexpected error %v", err)
	}
	if got != false {
		t.Errorf("CompareOperands(true, 1): got %v, want false", got)
	}
}

func TestCompareOperands_UnorderableSameTypeReturnsError(t *testing.T) {
	t.Parallel()

	// Given: two nils are the same type, but nil has no defined ordering.
	// When
	_, err := values.CompareOperands(">=", nil, nil)

	// Then
	if err == nil {
		t.Errorf("CompareOperands(nil, nil): got nil error, want an error")
	}
}

func TestCompareOperands_Ordering(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		op   string
		a, b any
		want bool
	}{
		{"int greater than", ">", 2, 1, true},
		{"int greater than false", ">", 1, 2, false},
		{"int greater than or equal, equal", ">=", 2, 2, true},
		{"int less than", "<", 1, 2, true},
		{"int less than or equal, equal", "<=", 2, 2, true},
		{"string lexicographic greater", ">", "b", "a", true},
		{"string lexicographic not greater", ">", "a", "b", false},
		{"float greater than", ">", 1.5, 1.0, true},
		{"bool greater than", ">", true, false, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// When
			got, err := values.CompareOperands(tc.op, tc.a, tc.b)

			// Then
			if err != nil {
				t.Fatalf("CompareOperands(%q, %#v, %#v): unexpected error %v", tc.op, tc.a, tc.b, err)
			}
			if got != tc.want {
				t.Errorf("CompareOperands(%q, %#v, %#v): got %v, want %v", tc.op, tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func TestAddOperands(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a, b any
		want any
	}{
		{"string concatenation", "Winston", "Churchill", "WinstonChurchill"},
		{"string concatenation with space", "Winston ", "Churchill", "Winston Churchill"},
		{"empty string concatenation", "", "a", "a"},
		{"int addition", 1, 2, 3},
		{"float addition", 1.5, 2.5, 4.0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// When
			got, err := values.AddOperands(tc.a, tc.b)

			// Then
			if err != nil {
				t.Fatalf("AddOperands(%#v, %#v): unexpected error %v", tc.a, tc.b, err)
			}
			if got != tc.want {
				t.Errorf("AddOperands(%#v, %#v): got %#v, want %#v", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func TestAddOperands_MismatchedTypesIsError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a, b any
	}{
		{"string and int", "a", 1},
		{"int and float", 1, 1.5},
		{"bool and bool", true, false},
		{"nil and string", nil, "a"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// When
			_, err := values.AddOperands(tc.a, tc.b)

			// Then
			if err == nil {
				t.Errorf("AddOperands(%#v, %#v): got nil error, want an error", tc.a, tc.b)
			}
		})
	}
}

func TestMergeOperands_BuildsFlatTuple(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a, b any
		want values.Tuple
	}{
		{"two scalars", 1, 2, values.Tuple{1, 2}},
		{"tuple then scalar", values.Tuple{1, 2}, 3, values.Tuple{1, 2, 3}},
		{"scalar then tuple", 1, values.Tuple{2, 3}, values.Tuple{1, 2, 3}},
		{"tuple then tuple", values.Tuple{1, 2}, values.Tuple{3, 4}, values.Tuple{1, 2, 3, 4}},
		{"array is not spread", values.Array{1, 2}, 3, values.Tuple{values.Array{1, 2}, 3}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// When
			got := values.MergeOperands(tc.a, tc.b)

			// Then
			gotTuple, ok := got.(values.Tuple)
			if !ok {
				t.Fatalf("MergeOperands(%#v, %#v): got %#v, want a Tuple", tc.a, tc.b, got)
			}
			if len(gotTuple) != len(tc.want) {
				t.Fatalf("MergeOperands(%#v, %#v): got %#v, want %#v", tc.a, tc.b, gotTuple, tc.want)
			}
			for i := range tc.want {
				if !values.EqualOperands(gotTuple[i], tc.want[i]) {
					t.Errorf("MergeOperands(%#v, %#v)[%d]: got %#v, want %#v", tc.a, tc.b, i, gotTuple[i], tc.want[i])
				}
			}
		})
	}
}

func TestSpreadArgs(t *testing.T) {
	t.Parallel()

	// Given
	tuple := values.Tuple{1, 2, 3}

	// When
	got := values.SpreadArgs(tuple)

	// Then
	if len(got) != 3 {
		t.Errorf("SpreadArgs(%#v): got %d args, want 3", tuple, len(got))
	}
}

func TestSpreadArgs_NonTupleBecomesSingleArg(t *testing.T) {
	t.Parallel()

	// When
	got := values.SpreadArgs(values.Array{1, 2})

	// Then
	if len(got) != 1 {
		t.Errorf("SpreadArgs(Array{1,2}): got %d args, want 1", len(got))
	}
}
