package oktaexpr_test

import (
	"errors"
	"strings"
	"testing"

	oktaexpr "github.com/stevenewson/okta-expression-parser"
	"github.com/stevenewson/okta-expression-parser/values"
)

func TestParseBool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		expr string
		want bool
	}{
		{"true literal", "true", true},
		{"comparison result", "1 == 1", true},
		{"false literal", "false", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Given
			p := oktaexpr.New()

			// When
			got, err := p.ParseBool(tc.expr)

			// Then
			if err != nil {
				t.Fatalf("ParseBool(%q): unexpected error %v", tc.expr, err)
			}
			if got != tc.want {
				t.Errorf("ParseBool(%q): got %v, want %v", tc.expr, got, tc.want)
			}
		})
	}
}

func TestParseBool_WrongTypeReturnsErrUnexpectedType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		expr     string
		wantType string // the actual type's name, which must appear in the error message
	}{
		{"int result", "1", "int"},
		{"string result", `"true"`, "string"},
		{"null result", "null", "NoneType"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Given
			p := oktaexpr.New()

			// When
			got, err := p.ParseBool(tc.expr)

			// Then
			if !errors.Is(err, oktaexpr.ErrUnexpectedType) {
				t.Fatalf("ParseBool(%q): got err %v, want an error wrapping ErrUnexpectedType", tc.expr, err)
			}
			if got != false {
				t.Errorf("ParseBool(%q): got %v on error, want the bool zero value false", tc.expr, got)
			}
			if !strings.Contains(err.Error(), tc.wantType) {
				t.Errorf("ParseBool(%q): error message %q does not mention the actual type %q", tc.expr, err.Error(), tc.wantType)
			}
		})
	}
}

func TestParseBool_PropagatesUnderlyingParseError(t *testing.T) {
	t.Parallel()

	// Given
	p := oktaexpr.New()

	// When
	_, err := p.ParseBool("1 ==")

	// Then: a genuine parse error should surface as-is, not as
	// ErrUnexpectedType.
	if err == nil {
		t.Fatalf(`ParseBool("1 =="): got nil error, want an error`)
	}
	if errors.Is(err, oktaexpr.ErrUnexpectedType) {
		t.Errorf(`ParseBool("1 =="): got ErrUnexpectedType, want the underlying parse error`)
	}
}

func TestParseString(t *testing.T) {
	t.Parallel()

	// Given
	p := oktaexpr.New()

	// When
	got, err := p.ParseString(`"hello"`)

	// Then
	if err != nil {
		t.Fatalf("ParseString: unexpected error %v", err)
	}
	if got != "hello" {
		t.Errorf(`ParseString("\"hello\""): got %q, want "hello"`, got)
	}
}

func TestParseString_WrongTypeReturnsErrUnexpectedType(t *testing.T) {
	t.Parallel()

	// Given
	p := oktaexpr.New()

	// When
	got, err := p.ParseString("true")

	// Then
	if !errors.Is(err, oktaexpr.ErrUnexpectedType) {
		t.Fatalf("ParseString(true): got err %v, want an error wrapping ErrUnexpectedType", err)
	}
	if got != "" {
		t.Errorf("ParseString(true): got %q on error, want the string zero value", got)
	}
	if !strings.Contains(err.Error(), "bool") {
		t.Errorf("ParseString(true): error message %q does not mention the actual type %q", err.Error(), "bool")
	}
}

func TestParseInt(t *testing.T) {
	t.Parallel()

	// Given
	p := oktaexpr.New()

	// When
	got, err := p.ParseInt("42")

	// Then
	if err != nil {
		t.Fatalf("ParseInt: unexpected error %v", err)
	}
	if got != 42 {
		t.Errorf("ParseInt(42): got %d, want 42", got)
	}
}

func TestParseInt_DoesNotCoerceFloat(t *testing.T) {
	t.Parallel()

	// Given: Convert.toNum always returns a float64, even for a
	// whole-number input; ParseInt should not silently coerce it.
	p := oktaexpr.New()

	// When
	got, err := p.ParseInt(`Convert.toNum("5")`)

	// Then
	if !errors.Is(err, oktaexpr.ErrUnexpectedType) {
		t.Fatalf(`ParseInt(Convert.toNum("5")): got err %v, want an error wrapping ErrUnexpectedType`, err)
	}
	if got != 0 {
		t.Errorf(`ParseInt(Convert.toNum("5")): got %d on error, want 0`, got)
	}
	if !strings.Contains(err.Error(), "float64") {
		t.Errorf(`ParseInt(Convert.toNum("5")): error message %q does not mention the actual type %q`, err.Error(), "float64")
	}
}

func TestParseFloat64(t *testing.T) {
	t.Parallel()

	// Given
	p := oktaexpr.New()

	// When
	got, err := p.ParseFloat64(`Convert.toNum("3.5")`)

	// Then
	if err != nil {
		t.Fatalf("ParseFloat64: unexpected error %v", err)
	}
	if got != 3.5 {
		t.Errorf("ParseFloat64: got %v, want 3.5", got)
	}
}

func TestParseFloat64_DoesNotCoerceInt(t *testing.T) {
	t.Parallel()

	// Given
	p := oktaexpr.New()

	// When
	got, err := p.ParseFloat64("5")

	// Then
	if !errors.Is(err, oktaexpr.ErrUnexpectedType) {
		t.Fatalf("ParseFloat64(5): got err %v, want an error wrapping ErrUnexpectedType", err)
	}
	if got != 0 {
		t.Errorf("ParseFloat64(5): got %v on error, want 0", got)
	}
	if !strings.Contains(err.Error(), "int") {
		t.Errorf("ParseFloat64(5): error message %q does not mention the actual type %q", err.Error(), "int")
	}
}

func TestParseArray(t *testing.T) {
	t.Parallel()

	// Given
	p := oktaexpr.New()

	// When
	got, err := p.ParseArray("Arrays.add({0,1}, 2)")

	// Then
	if err != nil {
		t.Fatalf("ParseArray: unexpected error %v", err)
	}
	want := values.Array{0, 1, 2}
	if !values.EqualOperands(got, want) {
		t.Errorf("ParseArray: got %#v, want %#v", got, want)
	}
}

func TestParseArray_WrongTypeReturnsErrUnexpectedType(t *testing.T) {
	t.Parallel()

	// Given
	p := oktaexpr.New()

	// When
	got, err := p.ParseArray("1")

	// Then
	if !errors.Is(err, oktaexpr.ErrUnexpectedType) {
		t.Fatalf("ParseArray(1): got err %v, want an error wrapping ErrUnexpectedType", err)
	}
	if got != nil {
		t.Errorf("ParseArray(1): got %#v on error, want nil", got)
	}
	if !strings.Contains(err.Error(), "int") {
		t.Errorf("ParseArray(1): error message %q does not mention the actual type %q", err.Error(), "int")
	}
}
