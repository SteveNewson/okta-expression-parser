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

// parseEval is Parse followed by Eval, for the typed ParseX convenience
// methods below — most callers evaluating a single expression once don't
// need the intermediate Node.
func (p *Parser) parseEval(expression string) (any, error) {
	node, err := p.Parse(expression)
	if err != nil {
		return nil, err
	}
	return p.Eval(node)
}

// ParseBool evaluates expression and requires the result to be a bool. If
// the expression evaluates successfully to any other type — including
// nil, since null is not a bool — it returns ErrUnexpectedType instead of
// requiring the caller to type-assert the result of Parse/Eval.
func (p *Parser) ParseBool(expression string) (bool, error) {
	result, err := p.parseEval(expression)
	if err != nil {
		return false, err
	}
	return p.requireBool(expression, result)
}

// EvalBool evaluates node (as returned by Parse) and requires the result
// to be a bool — the Node-accepting equivalent of ParseBool.
func (p *Parser) EvalBool(node Node) (bool, error) {
	result, err := p.Eval(node)
	if err != nil {
		return false, err
	}
	return p.requireBool(fmt.Sprintf("%v", node), result)
}

func (p *Parser) requireBool(expression string, result any) (bool, error) {
	b, ok := result.(bool)
	if !ok {
		return false, unexpectedTypeError(expression, "bool", result)
	}
	return b, nil
}

// ParseString evaluates expression and requires the result to be a string.
func (p *Parser) ParseString(expression string) (string, error) {
	result, err := p.parseEval(expression)
	if err != nil {
		return "", err
	}
	return p.requireString(expression, result)
}

// EvalString is the Node-accepting equivalent of ParseString.
func (p *Parser) EvalString(node Node) (string, error) {
	result, err := p.Eval(node)
	if err != nil {
		return "", err
	}
	return p.requireString(fmt.Sprintf("%v", node), result)
}

func (p *Parser) requireString(expression string, result any) (string, error) {
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
	result, err := p.parseEval(expression)
	if err != nil {
		return 0, err
	}
	return p.requireInt(expression, result)
}

// EvalInt is the Node-accepting equivalent of ParseInt.
func (p *Parser) EvalInt(node Node) (int, error) {
	result, err := p.Eval(node)
	if err != nil {
		return 0, err
	}
	return p.requireInt(fmt.Sprintf("%v", node), result)
}

func (p *Parser) requireInt(expression string, result any) (int, error) {
	n, ok := result.(int)
	if !ok {
		return 0, unexpectedTypeError(expression, "int", result)
	}
	return n, nil
}

// ParseFloat64 evaluates expression and requires the result to be a
// float64. It does not coerce an int result to a float64.
func (p *Parser) ParseFloat64(expression string) (float64, error) {
	result, err := p.parseEval(expression)
	if err != nil {
		return 0, err
	}
	return p.requireFloat64(expression, result)
}

// EvalFloat64 is the Node-accepting equivalent of ParseFloat64.
func (p *Parser) EvalFloat64(node Node) (float64, error) {
	result, err := p.Eval(node)
	if err != nil {
		return 0, err
	}
	return p.requireFloat64(fmt.Sprintf("%v", node), result)
}

func (p *Parser) requireFloat64(expression string, result any) (float64, error) {
	f, ok := result.(float64)
	if !ok {
		return 0, unexpectedTypeError(expression, "float64", result)
	}
	return f, nil
}

// ParseArray evaluates expression and requires the result to be an Array.
func (p *Parser) ParseArray(expression string) (Array, error) {
	result, err := p.parseEval(expression)
	if err != nil {
		return nil, err
	}
	return p.requireArray(expression, result)
}

// EvalArray is the Node-accepting equivalent of ParseArray.
func (p *Parser) EvalArray(node Node) (Array, error) {
	result, err := p.Eval(node)
	if err != nil {
		return nil, err
	}
	return p.requireArray(fmt.Sprintf("%v", node), result)
}

func (p *Parser) requireArray(expression string, result any) (Array, error) {
	arr, ok := result.(Array)
	if !ok {
		return nil, unexpectedTypeError(expression, "Array", result)
	}
	return arr, nil
}

func unexpectedTypeError(expression, want string, got any) error {
	return fmt.Errorf("expression %q evaluated to %s, want %s: %w", expression, values.TypeName(got), want, ErrUnexpectedType)
}
