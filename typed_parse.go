package oktaexpr

import (
	"errors"
	"fmt"

	"github.com/stevenewson/okta-expression-parser/values"
)

// ErrUnexpectedType is returned by the typed Parse* methods (ParseBool,
// ParseString, ParseInt, ParseFloat64, ParseArray) when an expression
// evaluates successfully but not to the type the caller asked for. It is
// always wrapped with details about the expression and the type actually
// produced, so callers should check for it with errors.Is rather than
// comparing errors directly.
var ErrUnexpectedType = errors.New("okta expression did not evaluate to the requested type")

// ParseBool evaluates expression and requires the result to be a bool. If
// the expression evaluates successfully to any other type — including
// nil, since null is not a bool — it returns ErrUnexpectedType instead of
// requiring the caller to type-assert the result of Parse.
func (p *Parser) ParseBool(expression string) (bool, error) {
	result, err := p.Parse(expression)
	if err != nil {
		return false, err
	}
	b, ok := result.(bool)
	if !ok {
		return false, unexpectedTypeError(expression, "bool", result)
	}
	return b, nil
}

// ParseString evaluates expression and requires the result to be a string.
func (p *Parser) ParseString(expression string) (string, error) {
	result, err := p.Parse(expression)
	if err != nil {
		return "", err
	}
	s, ok := result.(string)
	if !ok {
		return "", unexpectedTypeError(expression, "string", result)
	}
	return s, nil
}

// ParseInt evaluates expression and requires the result to be an int. It
// does not coerce a float64 result (e.g. from Convert.toNum) to an int,
// matching the language's own type-strictness for relational operators
// (see the README's note on GT/GE/LT/LE requiring exact type equality).
func (p *Parser) ParseInt(expression string) (int, error) {
	result, err := p.Parse(expression)
	if err != nil {
		return 0, err
	}
	n, ok := result.(int)
	if !ok {
		return 0, unexpectedTypeError(expression, "int", result)
	}
	return n, nil
}

// ParseFloat64 evaluates expression and requires the result to be a
// float64. It does not coerce an int result to a float64.
func (p *Parser) ParseFloat64(expression string) (float64, error) {
	result, err := p.Parse(expression)
	if err != nil {
		return 0, err
	}
	f, ok := result.(float64)
	if !ok {
		return 0, unexpectedTypeError(expression, "float64", result)
	}
	return f, nil
}

// ParseArray evaluates expression and requires the result to be an Array.
func (p *Parser) ParseArray(expression string) (Array, error) {
	result, err := p.Parse(expression)
	if err != nil {
		return nil, err
	}
	arr, ok := result.(Array)
	if !ok {
		return nil, unexpectedTypeError(expression, "Array", result)
	}
	return arr, nil
}

func unexpectedTypeError(expression, want string, got any) error {
	return fmt.Errorf("expression %q evaluated to %s, want %s: %w", expression, values.TypeName(got), want, ErrUnexpectedType)
}
