// Package expressionclasses implements the built-in "class" methods that an
// Okta expression can invoke, e.g. String.toUpperCase(...) or
// Arrays.contains(...).
package expressionclasses

// Class is implemented by any expression-language class (String, Arrays,
// etc.) that can invoke one of its methods by name with positional
// arguments.
type Class interface {
	Call(method string, args ...any) (any, error)
}

// Registry maps class names, as they appear in an expression (e.g.
// "String"), to their Class implementation. Passing a custom Registry to a
// parser lets callers add or override expression classes, mirroring the
// Python library's `expression_classes` module parameter.
type Registry map[string]Class

// Default returns a Registry containing the built-in expression classes:
// Arrays, String, Convert, Iso3166Convert, and Groups. Each call returns
// fresh instances, so Groups' per-parser data never leaks between parsers.
func Default() Registry {
	return Registry{
		"Arrays":         Arrays{},
		"String":         String{},
		"Convert":        Convert{},
		"Iso3166Convert": Iso3166Convert{},
		"Groups":         &Groups{},
	}
}
