package expressionclasses

import (
	"fmt"
	"strings"

	"github.com/stevenewson/okta-expression-parser/values"
)

// Arrays implements the Okta Expression Language "Arrays" class.
type Arrays struct{}

func (Arrays) Call(method string, args ...any) (any, error) {
	switch method {
	case "contains":
		return arraysContains(args)
	case "add":
		return arraysAdd(args)
	case "remove":
		return arraysRemove(args)
	case "clear":
		return arraysClear(args)
	case "get":
		return arraysGet(args)
	case "flatten":
		return arraysFlatten(args)
	case "size":
		return arraysSize(args)
	case "isEmpty":
		return arraysIsEmpty(args)
	case "toCsvString":
		return arraysToCsvString(args)
	default:
		return nil, fmt.Errorf("Arrays has no method %q", method)
	}
}

func asArray(v any, class, method string) (values.Array, error) {
	arr, ok := v.(values.Array)
	if !ok {
		return nil, fmt.Errorf("%s.%s: expected an array, got %s", class, method, values.TypeName(v))
	}
	return arr, nil
}

// asArrayOrEmpty is asArray, but treats a null argument as an empty array,
// matching the documented Arrays.size(NULL) == 0 and Arrays.isEmpty(NULL) ==
// true, rather than erroring like every other Arrays* function does on a
// non-array argument.
func asArrayOrEmpty(v any, class, method string) (values.Array, error) {
	if v == nil {
		return values.Array{}, nil
	}
	return asArray(v, class, method)
}

// contains tests if a value exists in an expression's array.
func arraysContains(args []any) (any, error) {
	if len(args) != 2 {
		return nil, argCountError("Arrays", "contains", "2", len(args))
	}
	arr, err := asArray(args[0], "Arrays", "contains")
	if err != nil {
		return nil, err
	}
	for _, item := range arr {
		if values.EqualOperands(item, args[1]) {
			return true, nil
		}
	}
	return false, nil
}

// add appends an element to a list and returns the list.
func arraysAdd(args []any) (any, error) {
	if len(args) != 2 {
		return nil, argCountError("Arrays", "add", "2", len(args))
	}
	arr, err := asArray(args[0], "Arrays", "add")
	if err != nil {
		return nil, err
	}
	out := make(values.Array, 0, len(arr)+1)
	out = append(out, arr...)
	out = append(out, args[1])
	return out, nil
}

// remove removes all occurrences of a value from a list.
func arraysRemove(args []any) (any, error) {
	if len(args) != 2 {
		return nil, argCountError("Arrays", "remove", "2", len(args))
	}
	arr, err := asArray(args[0], "Arrays", "remove")
	if err != nil {
		return nil, err
	}
	out := make(values.Array, 0, len(arr))
	for _, item := range arr {
		if !values.EqualOperands(item, args[1]) {
			out = append(out, item)
		}
	}
	return out, nil
}

// clear returns an empty list.
func arraysClear(args []any) (any, error) {
	if len(args) != 1 {
		return nil, argCountError("Arrays", "clear", "1", len(args))
	}
	return values.Array{}, nil
}

// get returns the element at index. If index does not exist, returns nil.
func arraysGet(args []any) (any, error) {
	if len(args) != 2 {
		return nil, argCountError("Arrays", "get", "2", len(args))
	}
	arr, err := asArray(args[0], "Arrays", "get")
	if err != nil {
		return nil, err
	}
	idx, err := toInt(args[1])
	if err != nil {
		return nil, fmt.Errorf("arrays.get: %w", err)
	}
	if idx < 0 {
		idx += len(arr)
	}
	if idx < 0 || idx >= len(arr) {
		return nil, nil
	}
	return arr[idx], nil
}

// flatten returns a flattened list from all args.
func arraysFlatten(args []any) (any, error) {
	out := values.Array{}
	for _, row := range args {
		if arr, ok := row.(values.Array); ok {
			out = append(out, arr...)
		} else {
			out = append(out, row)
		}
	}
	return out, nil
}

// size returns the number of elements in the array.
func arraysSize(args []any) (any, error) {
	if len(args) != 1 {
		return nil, argCountError("Arrays", "size", "1", len(args))
	}
	arr, err := asArrayOrEmpty(args[0], "Arrays", "size")
	if err != nil {
		return nil, err
	}
	return len(arr), nil
}

// isEmpty returns whether an array is empty or not.
func arraysIsEmpty(args []any) (any, error) {
	if len(args) != 1 {
		return nil, argCountError("Arrays", "isEmpty", "1", len(args))
	}
	arr, err := asArrayOrEmpty(args[0], "Arrays", "isEmpty")
	if err != nil {
		return nil, err
	}
	return len(arr) == 0, nil
}

// toCsvString returns a comma delineated string from array elements.
func arraysToCsvString(args []any) (any, error) {
	if len(args) != 1 {
		return nil, argCountError("Arrays", "toCsvString", "1", len(args))
	}
	arr, err := asArray(args[0], "Arrays", "toCsvString")
	if err != nil {
		return nil, err
	}
	parts := make([]string, len(arr))
	for i, item := range arr {
		s, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("Arrays.toCsvString: element %d is %s, not a string", i, values.TypeName(item))
		}
		parts[i] = s
	}
	return strings.Join(parts, ","), nil
}
