package expressionclasses_test

import (
	"testing"

	"github.com/stevenewson/okta-expression-parser/expressionclasses"
)

func TestIso3166Convert_Call(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		method string
		arg    string
		want   any
	}{
		{"toAlpha2 from alpha3", "toAlpha2", "USA", "US"},
		{"toAlpha3 from alpha2", "toAlpha3", "US", "USA"},
		{"toNumeric from alpha2", "toNumeric", "US", "840"},
		{"toName from alpha2", "toName", "US", "United States of America"},
		{"toAlpha2 from numeric", "toAlpha2", "840", "US"},
		{"toAlpha2 from name", "toAlpha2", "United States of America", "US"},
		{"toAlpha2 is case insensitive", "toAlpha2", "usa", "US"},
		{"unknown identifier returns nil", "toAlpha2", "ZZZZZZ", nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Given
			iso := expressionclasses.Iso3166Convert{}

			// When
			got, err := iso.Call(tc.method, tc.arg)

			// Then
			if err != nil {
				t.Fatalf("Iso3166Convert.%s(%q): unexpected error %v", tc.method, tc.arg, err)
			}
			if got != tc.want {
				t.Errorf("Iso3166Convert.%s(%q): got %#v, want %#v", tc.method, tc.arg, got, tc.want)
			}
		})
	}
}

func TestIso3166Convert_NonStringReturnsNil(t *testing.T) {
	t.Parallel()

	// Given
	iso := expressionclasses.Iso3166Convert{}

	// When
	got, err := iso.Call("toAlpha2", 5)

	// Then
	if err != nil {
		t.Fatalf("Iso3166Convert.toAlpha2(5): unexpected error %v", err)
	}
	if got != nil {
		t.Errorf("Iso3166Convert.toAlpha2(5): got %#v, want nil", got)
	}
}
