package expressionclasses

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/stevenewson/okta-expression-parser/values"
)

// toInt mirrors Python's int() builtin for the operand types this language
// produces well enough for internal parameter coercion (e.g. array indices,
// substring bounds). Unlike Convert.toInt, an unparsable string is an error
// here rather than silently coerced to 0, since these call sites need a real
// number to proceed.
func toInt(v any) (int, error) {
	switch t := v.(type) {
	case int:
		return t, nil
	case bool:
		if t {
			return 1, nil
		}
		return 0, nil
	case float64:
		return int(t), nil
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(t))
		if err != nil {
			return 0, fmt.Errorf("cannot convert %q to int", t)
		}
		return n, nil
	default:
		return 0, fmt.Errorf("cannot convert %s to int", values.TypeName(v))
	}
}

// pyStr mirrors Python's str() builtin for the operand types this language
// produces.
func pyStr(v any) string {
	switch t := v.(type) {
	case nil:
		return "None"
	case bool:
		if t {
			return "True"
		}
		return "False"
	case string:
		return t
	case int:
		return strconv.Itoa(t)
	case float64:
		return strconv.FormatFloat(t, 'g', -1, 64)
	default:
		return fmt.Sprintf("%v", t)
	}
}

// pySliceBound clamps a Python-style (possibly negative) slice index to
// [0, length], matching Python's slicing semantics.
func pySliceBound(idx, length int) int {
	if idx < 0 {
		idx += length
		if idx < 0 {
			idx = 0
		}
	}
	if idx > length {
		idx = length
	}
	return idx
}

// pySlice returns s[start:end] using Python's slicing semantics (negative
// indices count from the end, out-of-range bounds are clamped).
func pySlice(s string, start, end int) string {
	runes := []rune(s)
	n := len(runes)
	start = pySliceBound(start, n)
	end = pySliceBound(end, n)
	if start >= end {
		return ""
	}
	return string(runes[start:end])
}

func argCountError(class, method string, want string, got int) error {
	return fmt.Errorf("%s.%s expects %s argument(s), got %d", class, method, want, got)
}
