// Package values defines the runtime value model shared by the parser and
// the expression classes: the Array/Tuple container types plus the
// Python-flavored truthiness, equality, and ordering semantics that the
// source Okta Expression Language grammar is built on.
package values

import "fmt"

// Array represents an Okta Expression Language array literal, e.g. {1,2,3}.
type Array []any

// Tuple represents a comma-joined group of operands, used to pass multiple
// positional arguments to a class method call or to isMemberOfAnyGroup.
type Tuple []any

// Truthy mirrors Python's bool() coercion, since the source expression
// language's AND/OR/NOT/ternary operators are defined in terms of it.
func Truthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case int:
		return t != 0
	case float64:
		return t != 0
	case string:
		return t != ""
	case Array:
		return len(t) > 0
	case Tuple:
		return len(t) > 0
	case map[string]any:
		return len(t) > 0
	default:
		return true
	}
}

// MergeOperands implements the comma operator's flattening semantics: chained
// comma-separated operands build a single flat Tuple, matching how the
// Python grammar merges "operand , operand" productions.
func MergeOperands(a, b any) any {
	at, aIsTuple := a.(Tuple)
	bt, bIsTuple := b.(Tuple)

	switch {
	case aIsTuple && bIsTuple:
		out := make(Tuple, 0, len(at)+len(bt))
		out = append(out, at...)
		out = append(out, bt...)
		return out
	case aIsTuple && !bIsTuple:
		out := make(Tuple, 0, len(at)+1)
		out = append(out, at...)
		return append(out, b)
	case !aIsTuple && bIsTuple:
		out := make(Tuple, 0, len(bt)+1)
		out = append(out, a)
		return append(out, bt...)
	default:
		return Tuple{a, b}
	}
}

// SpreadArgs converts an operand into positional arguments for a class
// method call: a Tuple spreads into multiple arguments, anything else
// becomes a single argument.
func SpreadArgs(v any) []any {
	if t, ok := v.(Tuple); ok {
		return []any(t)
	}
	return []any{v}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// EqualOperands mirrors Python's == semantics for the value types this
// language produces, including the fact that bool is numerically comparable
// to int (True == 1).
func EqualOperands(a, b any) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}

	switch av := a.(type) {
	case bool:
		switch bv := b.(type) {
		case bool:
			return av == bv
		case int:
			return boolToInt(av) == bv
		}
		return false
	case int:
		switch bv := b.(type) {
		case int:
			return av == bv
		case bool:
			return av == boolToInt(bv)
		}
		return false
	case float64:
		bv, ok := b.(float64)
		return ok && av == bv
	case string:
		bv, ok := b.(string)
		return ok && av == bv
	case Array:
		bv, ok := b.(Array)
		if !ok || len(av) != len(bv) {
			return false
		}
		for i := range av {
			if !EqualOperands(av[i], bv[i]) {
				return false
			}
		}
		return true
	case Tuple:
		bv, ok := b.(Tuple)
		if !ok || len(av) != len(bv) {
			return false
		}
		for i := range av {
			if !EqualOperands(av[i], bv[i]) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// CompareOperands mirrors the source grammar's relational operators, which
// require the two operands to be of the exact same type (unlike ==, bool and
// int are NOT considered the same type here) and return false without error
// for mismatched types. Same-typed operands of a type with no defined
// ordering (nil, Array, Tuple, map) produce an error, matching the Python
// TypeError raised by an unsupported ">=" comparison.
func CompareOperands(op string, a, b any) (bool, error) {
	if !sameType(a, b) {
		return false, nil
	}

	switch av := a.(type) {
	case bool:
		return compareInts(op, boolToInt(av), boolToInt(b.(bool))), nil
	case int:
		return compareInts(op, av, b.(int)), nil
	case float64:
		return compareFloats(op, av, b.(float64)), nil
	case string:
		return compareStrings(op, av, b.(string)), nil
	default:
		return false, fmt.Errorf("'%s' not supported between instances of %s and %s", op, TypeName(a), TypeName(b))
	}
}

// AddOperands implements the "+" operator: string concatenation for two
// strings, arithmetic addition for two ints or two floats, matching the
// documented "Concatenate two strings" example and the fact that Integer and
// Number are both listed as ordinary constant types. Like CompareOperands,
// mismatched types (including bool, which is never treated as numeric here)
// are rejected rather than coerced.
func AddOperands(a, b any) (any, error) {
	switch av := a.(type) {
	case string:
		if bv, ok := b.(string); ok {
			return av + bv, nil
		}
	case int:
		if bv, ok := b.(int); ok {
			return av + bv, nil
		}
	case float64:
		if bv, ok := b.(float64); ok {
			return av + bv, nil
		}
	}
	return nil, fmt.Errorf("unsupported operand type(s) for +: %q and %q", TypeName(a), TypeName(b))
}

func sameType(a, b any) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	switch a.(type) {
	case bool:
		_, ok := b.(bool)
		return ok
	case int:
		_, ok := b.(int)
		return ok
	case float64:
		_, ok := b.(float64)
		return ok
	case string:
		_, ok := b.(string)
		return ok
	default:
		return TypeName(a) == TypeName(b)
	}
}

// TypeName returns a human-readable type name used in error messages.
func TypeName(v any) string {
	if v == nil {
		return "NoneType"
	}
	switch v.(type) {
	case Array:
		return "Array"
	case Tuple:
		return "Tuple"
	case map[string]any:
		return "map"
	default:
		return fmt.Sprintf("%T", v)
	}
}

func compareInts(op string, a, b int) bool {
	switch op {
	case ">":
		return a > b
	case ">=":
		return a >= b
	case "<":
		return a < b
	case "<=":
		return a <= b
	}
	return false
}

func compareFloats(op string, a, b float64) bool {
	switch op {
	case ">":
		return a > b
	case ">=":
		return a >= b
	case "<":
		return a < b
	case "<=":
		return a <= b
	}
	return false
}

func compareStrings(op string, a, b string) bool {
	switch op {
	case ">":
		return a > b
	case ">=":
		return a >= b
	case "<":
		return a < b
	case "<=":
		return a <= b
	}
	return false
}
