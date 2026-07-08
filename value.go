package oktaexpr

import "github.com/stevenewson/okta-expression-parser/values"

// Array represents an Okta Expression Language array literal, e.g. {1,2,3}.
type Array = values.Array

// Tuple represents a comma-joined group of operands produced by the comma
// operator, e.g. when passing multiple arguments to a class method call.
type Tuple = values.Tuple

func truthy(v any) bool                                 { return values.Truthy(v) }
func mergeOperands(a, b any) any                        { return values.MergeOperands(a, b) }
func spreadArgs(v any) []any                            { return values.SpreadArgs(v) }
func equalOperands(a, b any) bool                       { return values.EqualOperands(a, b) }
func compareOperands(op string, a, b any) (bool, error) { return values.CompareOperands(op, a, b) }
func addOperands(a, b any) (any, error)                 { return values.AddOperands(a, b) }
