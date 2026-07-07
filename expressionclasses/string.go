package expressionclasses

import (
	"fmt"
	"strings"

	"github.com/stevenewson/okta-expression-parser/values"
)

// String implements the Okta Expression Language "String" class.
type String struct{}

func (String) Call(method string, args ...any) (any, error) {
	switch method {
	case "stringContains":
		return stringStringContains(args)
	case "startsWith":
		return stringStartsWith(args)
	case "toLowerCase":
		return stringToLowerCase(args)
	case "toUpperCase":
		return stringToUpperCase(args)
	case "append":
		return stringAppend(args)
	case "join":
		return stringJoin(args)
	case "removeSpaces":
		return stringRemoveSpaces(args)
	case "replace":
		return stringReplace(args)
	case "replaceFirst":
		return stringReplaceFirst(args)
	case "contains":
		return stringContains(args)
	case "stringSwitch":
		return stringSwitch(args)
	case "substring":
		return stringSubstring(args)
	case "substringAfter":
		return stringSubstringAfter(args)
	case "substringBefore":
		return stringSubstringBefore(args)
	default:
		return nil, fmt.Errorf("String has no method %q", method)
	}
}

// stringContains tests if a string contains another string. Unlike the
// contains method below, both arguments must genuinely be strings.
func stringStringContains(args []any) (any, error) {
	if len(args) != 2 {
		return nil, argCountError("String", "stringContains", "2", len(args))
	}
	strToTest, ok1 := args[0].(string)
	val, ok2 := args[1].(string)
	if !ok1 || !ok2 {
		return false, nil
	}
	return strings.Contains(strToTest, val), nil
}

// startsWith returns whether val starts with test.
func stringStartsWith(args []any) (any, error) {
	if len(args) != 2 {
		return nil, argCountError("String", "startsWith", "2", len(args))
	}
	test, ok := args[1].(string)
	if !ok {
		return nil, fmt.Errorf("String.startsWith: prefix must be a string, got %s", values.TypeName(args[1]))
	}
	return strings.HasPrefix(pyStr(args[0]), test), nil
}

// toLowerCase casts to lower case; non-strings pass through unchanged.
func stringToLowerCase(args []any) (any, error) {
	if len(args) != 1 {
		return nil, argCountError("String", "toLowerCase", "1", len(args))
	}
	if s, ok := args[0].(string); ok {
		return strings.ToLower(s), nil
	}
	return args[0], nil
}

// toUpperCase casts to upper case; non-strings pass through unchanged.
func stringToUpperCase(args []any) (any, error) {
	if len(args) != 1 {
		return nil, argCountError("String", "toUpperCase", "1", len(args))
	}
	if s, ok := args[0].(string); ok {
		return strings.ToUpper(s), nil
	}
	return args[0], nil
}

// append concatenates two strings with a literal "$" separator.
func stringAppend(args []any) (any, error) {
	if len(args) != 2 {
		return nil, argCountError("String", "append", "2", len(args))
	}
	return fmt.Sprintf("%s$%s", pyStr(args[0]), pyStr(args[1])), nil
}

// join returns a joined string using separator.
func stringJoin(args []any) (any, error) {
	if len(args) < 1 {
		return nil, argCountError("String", "join", "at least 1", len(args))
	}
	separator, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("string.join: separator must be a string, got %s", values.TypeName(args[0]))
	}
	parts := make([]string, len(args)-1)
	for i, v := range args[1:] {
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("string.join: argument %d is %s, not a string", i+1, values.TypeName(v))
		}
		parts[i] = s
	}
	return strings.Join(parts, separator), nil
}

// removeSpaces removes spaces from a string.
func stringRemoveSpaces(args []any) (any, error) {
	if len(args) != 1 {
		return nil, argCountError("String", "removeSpaces", "1", len(args))
	}
	return strings.ReplaceAll(pyStr(args[0]), " ", ""), nil
}

// replace replaces all occurrences of match with replacement in val.
func stringReplace(args []any) (any, error) {
	if len(args) != 3 {
		return nil, argCountError("String", "replace", "3", len(args))
	}
	return strings.ReplaceAll(pyStr(args[0]), pyStr(args[1]), pyStr(args[2])), nil
}

// replaceFirst replaces the first occurrence of match with replacement in val.
func stringReplaceFirst(args []any) (any, error) {
	if len(args) != 3 {
		return nil, argCountError("String", "replaceFirst", "3", len(args))
	}
	return strings.Replace(pyStr(args[0]), pyStr(args[1]), pyStr(args[2]), 1), nil
}

// contains returns whether val contains test, coercing both to strings.
func stringContains(args []any) (any, error) {
	if len(args) != 2 {
		return nil, argCountError("String", "contains", "2", len(args))
	}
	return strings.Contains(pyStr(args[0]), pyStr(args[1])), nil
}

// stringSwitch returns the value paired with the first key that appears as a
// substring of val, or default if none match. An odd number of key/value
// arguments returns false, matching the source library's behavior.
func stringSwitch(args []any) (any, error) {
	if len(args) < 2 {
		return nil, argCountError("String", "stringSwitch", "at least 2", len(args))
	}
	val, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("String.stringSwitch: val must be a string, got %s", values.TypeName(args[0]))
	}
	defaultVal := args[1]
	kvPairs := args[2:]

	if len(kvPairs)%2 != 0 {
		return false, nil
	}

	for i := 0; i < len(kvPairs); i += 2 {
		key, ok := kvPairs[i].(string)
		if !ok {
			return nil, fmt.Errorf("String.stringSwitch: key %d must be a string, got %s", i/2, values.TypeName(kvPairs[i]))
		}
		if strings.Contains(val, key) {
			return kvPairs[i+1], nil
		}
	}

	return defaultVal, nil
}

// substring returns a substring of val starting at index start and ending at
// index end. If end is beyond the length of val, it is clamped to the last
// character (matching the source library's quirky end=-1 fallback) rather
// than the end of the string.
func stringSubstring(args []any) (any, error) {
	if len(args) != 3 {
		return nil, argCountError("String", "substring", "3", len(args))
	}
	val, ok := args[0].(string)
	if !ok {
		return "", nil
	}
	start, err := toInt(args[1])
	if err != nil {
		return nil, fmt.Errorf("string.substring: %w", err)
	}
	end, err := toInt(args[2])
	if err != nil {
		return nil, fmt.Errorf("string.substring: %w", err)
	}
	if end > len([]rune(val)) {
		end = -1
	}
	return pySlice(val, start, end), nil
}

// substringAfter returns the substring of val starting at the first
// occurrence of search. If search is not found, val is returned unchanged.
func stringSubstringAfter(args []any) (any, error) {
	if len(args) != 2 {
		return nil, argCountError("String", "substringAfter", "2", len(args))
	}
	val := pyStr(args[0])
	search := pyStr(args[1])
	idx := strings.Index(val, search)
	if idx < 0 {
		return args[0], nil
	}
	return val[idx:], nil
}

// substringBefore returns the substring of val before the first occurrence
// of search. If search is not found, val is returned unchanged.
func stringSubstringBefore(args []any) (any, error) {
	if len(args) != 2 {
		return nil, argCountError("String", "substringBefore", "2", len(args))
	}
	val := pyStr(args[0])
	search := pyStr(args[1])
	idx := strings.Index(val, search)
	if idx < 0 {
		return args[0], nil
	}
	return val[:idx], nil
}
