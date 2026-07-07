package expressionclasses

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/stevenewson/okta-expression-parser/values"
)

// Convert implements the Okta Expression Language "Convert" class.
//
// The Python source's Convert.toInt/toNum were unreachable: both were
// declared as @classmethod but their function signatures omitted the
// implicit cls parameter, so any call raised "takes 1 positional argument
// but 2 were given". This port fixes that so the class behaves as its
// docstrings describe.
type Convert struct{}

func (Convert) Call(method string, args ...any) (any, error) {
	switch method {
	case "toInt":
		return convertToInt(args)
	case "toNum":
		return convertToNum(args)
	default:
		return nil, fmt.Errorf("Convert has no method %q", method)
	}
}

// toInt casts a value to an int. A string that doesn't parse as an integer
// yields 0, matching the source's ValueError fallback.
func convertToInt(args []any) (any, error) {
	if len(args) != 1 {
		return nil, argCountError("Convert", "toInt", "1", len(args))
	}
	switch v := args[0].(type) {
	case int:
		return v, nil
	case bool:
		return boolToInt(v), nil
	case float64:
		return int(v), nil
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return 0, nil
		}
		return n, nil
	default:
		return nil, fmt.Errorf("Convert.toInt: cannot convert %s to int", values.TypeName(v))
	}
}

// toNum casts a value to a float. A string that doesn't parse as a number
// yields 0, matching the source's ValueError fallback.
func convertToNum(args []any) (any, error) {
	if len(args) != 1 {
		return nil, argCountError("Convert", "toNum", "1", len(args))
	}
	switch v := args[0].(type) {
	case int:
		return float64(v), nil
	case bool:
		return float64(boolToInt(v)), nil
	case float64:
		return v, nil
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil {
			return float64(0), nil
		}
		return f, nil
	default:
		return nil, fmt.Errorf("Convert.toNum: cannot convert %s to float", values.TypeName(v))
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
