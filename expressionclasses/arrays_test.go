package expressionclasses_test

import (
	"testing"

	"github.com/stevenewson/okta-expression-parser/expressionclasses"
	"github.com/stevenewson/okta-expression-parser/values"
)

func TestArrays_Contains(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		array values.Array
		val   any
		want  bool
	}{
		{"contains present int", values.Array{0, 1, 2}, 0, true},
		{"does not contain absent int", values.Array{0, 1, 2}, 5, false},
		{"contains present string", values.Array{"a", "b"}, "a", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Given
			arr := expressionclasses.Arrays{}

			// When
			got, err := arr.Call("contains", tc.array, tc.val)

			// Then
			if err != nil {
				t.Fatalf("Arrays.contains(%#v, %#v): unexpected error %v", tc.array, tc.val, err)
			}
			if got != tc.want {
				t.Errorf("Arrays.contains(%#v, %#v): got %v, want %v", tc.array, tc.val, got, tc.want)
			}
		})
	}
}

func TestArrays_Add(t *testing.T) {
	t.Parallel()

	// Given
	arr := expressionclasses.Arrays{}

	// When
	got, err := arr.Call("add", values.Array{0, 1}, 2)

	// Then
	if err != nil {
		t.Fatalf("Arrays.add: unexpected error %v", err)
	}
	want := values.Array{0, 1, 2}
	if !values.EqualOperands(got, want) {
		t.Errorf("Arrays.add({0,1}, 2): got %#v, want %#v", got, want)
	}
}

func TestArrays_Remove(t *testing.T) {
	t.Parallel()

	// Given
	arr := expressionclasses.Arrays{}

	// When
	got, err := arr.Call("remove", values.Array{0, 1, 2, 1}, 1)

	// Then
	if err != nil {
		t.Fatalf("Arrays.remove: unexpected error %v", err)
	}
	want := values.Array{0, 2}
	if !values.EqualOperands(got, want) {
		t.Errorf("Arrays.remove({0,1,2,1}, 1): got %#v, want %#v", got, want)
	}
}

func TestArrays_Clear(t *testing.T) {
	t.Parallel()

	// Given
	arr := expressionclasses.Arrays{}

	// When
	got, err := arr.Call("clear", values.Array{0, 1, 2})

	// Then
	if err != nil {
		t.Fatalf("Arrays.clear: unexpected error %v", err)
	}
	if !values.EqualOperands(got, values.Array{}) {
		t.Errorf("Arrays.clear({0,1,2}): got %#v, want an empty array", got)
	}
}

func TestArrays_Get(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		array values.Array
		index int
		want  any
	}{
		{"in range", values.Array{0, 1, 2}, 1, 1},
		{"out of range returns nil", values.Array{0, 1, 2}, 5, nil},
		{"negative index counts from end", values.Array{0, 1, 2}, -1, 2},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Given
			arr := expressionclasses.Arrays{}

			// When
			got, err := arr.Call("get", tc.array, tc.index)

			// Then
			if err != nil {
				t.Fatalf("Arrays.get(%#v, %d): unexpected error %v", tc.array, tc.index, err)
			}
			if !values.EqualOperands(got, tc.want) {
				t.Errorf("Arrays.get(%#v, %d): got %#v, want %#v", tc.array, tc.index, got, tc.want)
			}
		})
	}
}

func TestArrays_Flatten(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []any
		want values.Array
	}{
		{"flattens two arrays", []any{values.Array{1, 2}, values.Array{3, 4}}, values.Array{1, 2, 3, 4}},
		{"flattens array and scalar", []any{values.Array{1, 2}, 3}, values.Array{1, 2, 3}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Given
			arr := expressionclasses.Arrays{}

			// When
			got, err := arr.Call("flatten", tc.args...)

			// Then
			if err != nil {
				t.Fatalf("Arrays.flatten(%#v): unexpected error %v", tc.args, err)
			}
			if !values.EqualOperands(got, tc.want) {
				t.Errorf("Arrays.flatten(%#v): got %#v, want %#v", tc.args, got, tc.want)
			}
		})
	}
}

func TestArrays_Size(t *testing.T) {
	t.Parallel()

	// Given
	arr := expressionclasses.Arrays{}

	// When
	got, err := arr.Call("size", values.Array{0, 1, 2})

	// Then
	if err != nil {
		t.Fatalf("Arrays.size: unexpected error %v", err)
	}
	if got != 3 {
		t.Errorf("Arrays.size({0,1,2}): got %v, want 3", got)
	}
}

func TestArrays_IsEmpty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		array values.Array
		want  bool
	}{
		{"empty array is empty", values.Array{}, true},
		{"nonempty array is not empty", values.Array{1}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Given
			arr := expressionclasses.Arrays{}

			// When
			got, err := arr.Call("isEmpty", tc.array)

			// Then
			if err != nil {
				t.Fatalf("Arrays.isEmpty(%#v): unexpected error %v", tc.array, err)
			}
			if got != tc.want {
				t.Errorf("Arrays.isEmpty(%#v): got %v, want %v", tc.array, got, tc.want)
			}
		})
	}
}

func TestArrays_ToCsvString(t *testing.T) {
	t.Parallel()

	// Given
	arr := expressionclasses.Arrays{}

	// When
	got, err := arr.Call("toCsvString", values.Array{"a", "b", "c"})

	// Then
	if err != nil {
		t.Fatalf("Arrays.toCsvString: unexpected error %v", err)
	}
	if got != "a,b,c" {
		t.Errorf(`Arrays.toCsvString({"a","b","c"}): got %v, want "a,b,c"`, got)
	}
}

func TestArrays_ToCsvString_NonStringElementIsError(t *testing.T) {
	t.Parallel()

	// Given: the source library's version raises a Python TypeError for
	// this exact input (join requires str elements).
	arr := expressionclasses.Arrays{}

	// When
	_, err := arr.Call("toCsvString", values.Array{1, 2, 3})

	// Then
	if err == nil {
		t.Errorf("Arrays.toCsvString({1,2,3}): got nil error, want an error")
	}
}
