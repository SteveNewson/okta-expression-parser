package expressionclasses_test

import (
	"testing"

	"github.com/stevenewson/okta-expression-parser/expressionclasses"
)

// The Python source's Convert.toInt/toNum are unreachable: both are
// decorated with @classmethod but their function signatures omit the
// implicit cls parameter, so every call raises "takes 1 positional argument
// but 2 were given". These tests verify this port's fix, since there is no
// working Python behavior to use as an oracle here.
func TestConvert_ToInt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		arg  any
		want int
	}{
		{"parses digit string", "42", 42},
		{"passes through int", 7, 7},
		{"truncates float", 3.9, 3},
		{"true is 1", true, 1},
		{"false is 0", false, 0},
		{"unparsable string falls back to 0", "not-a-number", 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Given
			c := expressionclasses.Convert{}

			// When
			got, err := c.Call("toInt", tc.arg)

			// Then
			if err != nil {
				t.Fatalf("Convert.toInt(%#v): unexpected error %v", tc.arg, err)
			}
			if got != tc.want {
				t.Errorf("Convert.toInt(%#v): got %#v, want %v", tc.arg, got, tc.want)
			}
		})
	}
}

func TestConvert_ToNum(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		arg  any
		want float64
	}{
		{"parses decimal string", "3.5", 3.5},
		{"passes through float", 2.5, 2.5},
		{"converts int", 4, 4.0},
		{"unparsable string falls back to 0", "not-a-number", 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Given
			c := expressionclasses.Convert{}

			// When
			got, err := c.Call("toNum", tc.arg)

			// Then
			if err != nil {
				t.Fatalf("Convert.toNum(%#v): unexpected error %v", tc.arg, err)
			}
			if got != tc.want {
				t.Errorf("Convert.toNum(%#v): got %#v, want %v", tc.arg, got, tc.want)
			}
		})
	}
}

func TestConvert_ToInt_NilIsError(t *testing.T) {
	t.Parallel()

	// Given
	c := expressionclasses.Convert{}

	// When
	_, err := c.Call("toInt", nil)

	// Then
	if err == nil {
		t.Errorf("Convert.toInt(nil): got nil error, want an error")
	}
}
