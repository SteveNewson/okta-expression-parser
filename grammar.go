package oktaexpr

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/stevenewson/okta-expression-parser/expressionclasses"
)

// evalContext holds the token stream and evaluation environment for a single
// parse. It implements a recursive-descent evaluator that mirrors the source
// sly/yacc grammar: rather than building an AST and evaluating it afterward,
// each grammar level evaluates its result immediately as it parses, exactly
// as the bottom-up parser's embedded reduction actions did. One consequence
// worth calling out: AND/OR/ternary never short-circuit parsing, because by
// the time a grammar rule fires all of its children are already evaluated;
// this port preserves that "always evaluate every subexpression" behavior.
type evalContext struct {
	toks        []token
	pos         int
	userProfile map[string]any
	groupIDs    []string
	groupData   map[string]any
	classes     expressionclasses.Registry
}

func (c *evalContext) peek() token {
	return c.toks[c.pos]
}

func (c *evalContext) advance() token {
	t := c.toks[c.pos]
	if c.pos < len(c.toks)-1 {
		c.pos++
	}
	return t
}

func (c *evalContext) expect(typ tokenType, desc string) (token, error) {
	tok := c.peek()
	if tok.typ != typ {
		return token{}, fmt.Errorf("expected %s but found %q at character %d", desc, tok.value, tok.pos)
	}
	return c.advance(), nil
}

// parse is the grammar's entry point: `result: condition | operand`.
func (c *evalContext) parse() (any, error) {
	val, _, err := c.parseExpr()
	if err != nil {
		return nil, err
	}
	if c.peek().typ != tokEOF {
		tok := c.peek()
		return nil, fmt.Errorf("unexpected token %q at character %d", tok.value, tok.pos)
	}
	return val, nil
}

// parseExpr parses a full condition-or-operand expression, including the
// ternary operator, which is the loosest-binding construct in the grammar.
// The returned bool reports whether the expression is "condition"-typed
// (produced by a comparison, AND/OR, NOT, or isMemberOf* builtin) as opposed
// to "operand"-typed. That distinction only matters for what can be used as
// a ternary branch or as a comparison's left/right side, exactly as in the
// source grammar, which defines nearly identical rules for both operand and
// condition types.
func (c *evalContext) parseExpr() (any, bool, error) {
	val, isCond, err := c.parseOr()
	if err != nil {
		return nil, false, err
	}
	if c.peek().typ != tokQuestion {
		return val, isCond, nil
	}
	c.advance()

	trueVal, err := c.parseOperandBranch()
	if err != nil {
		return nil, false, err
	}
	if _, err := c.expect(tokColon, "':'"); err != nil {
		return nil, false, err
	}
	falseVal, err := c.parseOperandBranch()
	if err != nil {
		return nil, false, err
	}

	// Both branches are always evaluated already (see the eager-evaluation
	// note on evalContext) before we pick one, matching the source.
	if truthy(val) {
		return trueVal, false, nil
	}
	return falseVal, false, nil
}

// parseOperandBranch parses an expression that must be operand-typed, used
// for ternary branches, array literal contents, and class-method/
// isMemberOf* call arguments. A nested ternary is allowed (it is always
// operand-typed), but a bare comparison, AND/OR, or isMemberOf* result is
// rejected, matching the source grammar's refusal to reduce a "condition"
// into an "operand".
//
// It also greedily chains a trailing comma into a Tuple via mergeOperands,
// matching the source grammar's "operand , operand" production. Because
// that production exists at every operand position, a comma inside a nested
// ternary branch binds to the innermost operand context rather than an
// enclosing call's argument list — e.g. in `f(true ? "a" : "b", "c")`, the
// ", \"c\"" is consumed by the false branch, not by f's argument list. That
// is surprising but is the source's real, verified behavior, so it's
// preserved here rather than "fixed".
func (c *evalContext) parseOperandBranch() (any, error) {
	val, err := c.parseOperandBranchItem()
	if err != nil {
		return nil, err
	}
	for c.peek().typ == tokComma {
		c.advance()
		rhs, err := c.parseOperandBranchItem()
		if err != nil {
			return nil, err
		}
		val = mergeOperands(val, rhs)
	}
	return val, nil
}

func (c *evalContext) parseOperandBranchItem() (any, error) {
	tok := c.peek()
	val, isCond, err := c.parseExpr()
	if err != nil {
		return nil, err
	}
	if isCond {
		return nil, fmt.Errorf("expected a value expression but found a boolean condition expression at character %d", tok.pos)
	}
	return val, nil
}

func (c *evalContext) parseOr() (any, bool, error) {
	val, isCond, err := c.parseAnd()
	if err != nil {
		return nil, false, err
	}
	for c.peek().typ == tokOr {
		c.advance()
		rhs, _, err := c.parseAnd()
		if err != nil {
			return nil, false, err
		}
		if truthy(val) {
			// val unchanged
		} else {
			val = rhs
		}
		isCond = true
	}
	return val, isCond, nil
}

func (c *evalContext) parseAnd() (any, bool, error) {
	val, isCond, err := c.parseNot()
	if err != nil {
		return nil, false, err
	}
	for c.peek().typ == tokAnd {
		c.advance()
		rhs, _, err := c.parseNot()
		if err != nil {
			return nil, false, err
		}
		if truthy(val) {
			val = rhs
		}
		isCond = true
	}
	return val, isCond, nil
}

func (c *evalContext) parseNot() (any, bool, error) {
	if c.peek().typ == tokNot {
		c.advance()
		val, _, err := c.parseNot()
		if err != nil {
			return nil, false, err
		}
		return !truthy(val), true, nil
	}
	return c.parseComparison()
}

var comparisonOps = map[tokenType]string{
	tokEQ:  "==",
	tokNE:  "!=",
	tokGT:  ">",
	tokGTE: ">=",
	tokLT:  "<",
	tokLTE: "<=",
}

// parseComparison implements the grammar's nonassoc EQ/NE/GT/GE/LT/LE level.
// Both sides must be operand-typed, matching the source, which only ever
// defines these operators over "operand EQ operand" (never over a
// "condition").
func (c *evalContext) parseComparison() (any, bool, error) {
	lhsTok := c.peek()
	val, isCond, err := c.parseAdditive()
	if err != nil {
		return nil, false, err
	}

	opName, isComparison := comparisonOps[c.peek().typ]
	if !isComparison {
		return val, isCond, nil
	}
	if isCond {
		return nil, false, fmt.Errorf("comparison operators require a value expression on the left, found a boolean condition expression at character %d", lhsTok.pos)
	}
	opTok := c.advance()

	rhsTok := c.peek()
	rhs, rhsCond, err := c.parseAdditive()
	if err != nil {
		return nil, false, err
	}
	if rhsCond {
		return nil, false, fmt.Errorf("comparison operators require a value expression on the right, found a boolean condition expression at character %d", rhsTok.pos)
	}

	result, err := applyComparison(opName, val, rhs)
	if err != nil {
		return nil, false, fmt.Errorf("%s at character %d", err, opTok.pos)
	}
	return result, true, nil
}

func applyComparison(op string, a, b any) (bool, error) {
	switch op {
	case "==":
		return equalOperands(a, b), nil
	case "!=":
		return !equalOperands(a, b), nil
	default:
		return compareOperands(op, a, b)
	}
}

// parseAdditive implements the "+" operator: string concatenation, or
// arithmetic addition for two ints or two floats (see values.AddOperands).
// It binds tighter than comparisons but looser than a primary/isMemberOf*
// atom, e.g. user.firstName + user.lastName == "WinstonChurchill" parses as
// (user.firstName + user.lastName) == "WinstonChurchill". Both sides must be
// operand-typed, matching the strictness of the comparison operators above.
func (c *evalContext) parseAdditive() (any, bool, error) {
	lhsTok := c.peek()
	val, isCond, err := c.parseAtom()
	if err != nil {
		return nil, false, err
	}

	for c.peek().typ == tokPlus {
		if isCond {
			return nil, false, fmt.Errorf("'+' requires a value expression on the left, found a boolean condition expression at character %d", lhsTok.pos)
		}
		plusTok := c.advance()

		rhsTok := c.peek()
		rhs, rhsCond, err := c.parseAtom()
		if err != nil {
			return nil, false, err
		}
		if rhsCond {
			return nil, false, fmt.Errorf("'+' requires a value expression on the right, found a boolean condition expression at character %d", rhsTok.pos)
		}

		val, err = addOperands(val, rhs)
		if err != nil {
			return nil, false, fmt.Errorf("%s at character %d", err, plusTok.pos)
		}
	}
	return val, isCond, nil
}

// parseAtom parses the atoms that comparisons operate on: primaries,
// isMemberOf* builtins (always condition-typed), and parenthesized
// expressions (which preserve whatever type was inside, exactly like the
// source's separate "(" condition ")" and "(" operand ")" rules).
func (c *evalContext) parseAtom() (any, bool, error) {
	switch c.peek().typ {
	case tokLParen:
		c.advance()
		val, isCond, err := c.parseExpr()
		if err != nil {
			return nil, false, err
		}
		if _, err := c.expect(tokRParen, "')'"); err != nil {
			return nil, false, err
		}
		return val, isCond, nil
	case tokMemberOf, tokMemberOfAny, tokMemberOfName, tokMemberOfGroupStartsWith, tokMemberOfGroupContains, tokMemberOfGroupNameRegex:
		val, err := c.parseMemberOf()
		return val, true, err
	default:
		val, err := c.parsePrimary()
		return val, false, err
	}
}

func (c *evalContext) parsePrimary() (any, error) {
	tok := c.peek()
	switch tok.typ {
	case tokInt:
		c.advance()
		n, err := strconv.Atoi(tok.value)
		if err != nil {
			return nil, fmt.Errorf("invalid integer literal %q at character %d", tok.value, tok.pos)
		}
		return n, nil
	case tokFloat:
		c.advance()
		f, err := strconv.ParseFloat(tok.value, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid number literal %q at character %d", tok.value, tok.pos)
		}
		return f, nil
	case tokString:
		c.advance()
		return unquoteString(tok.value), nil
	case tokBool:
		c.advance()
		return strings.EqualFold(tok.value, "true"), nil
	case tokNull:
		c.advance()
		return nil, nil
	case tokLBrace:
		return c.parseArray()
	case tokUser:
		c.advance()
		return c.parsePathChain(any(c.userProfile))
	case tokName:
		c.advance()
		// A bare NAME that isn't part of a user.<path> chain always resolves
		// to nil. This mirrors a quirk in the source grammar's `path: NAME`
		// rule, which looks the name up in an always-empty dict; only
		// user.<name> chains ever resolve to a real profile value.
		return c.parsePathChain(nil)
	case tokClass:
		return c.parseClassCall()
	default:
		return nil, fmt.Errorf("unexpected token %q at character %d", tok.value, tok.pos)
	}
}

// parsePathChain resolves a (possibly empty) chain of ".name" accesses
// starting from root. The source grammar only supported a single "." hop
// after "user" or a bare name; this port deliberately extends that to
// arbitrary depth (e.g. user.profile.department.name) to match real Okta
// Expression Language behavior. When a hop can't be resolved (the current
// value isn't a map, or the key is missing), the result is nil, rather than
// the source's quirk of returning the last resolved value unchanged.
func (c *evalContext) parsePathChain(root any) (any, error) {
	val := root
	for c.peek().typ == tokDot {
		c.advance()
		nameTok, err := c.expect(tokName, "a property name")
		if err != nil {
			return nil, err
		}

		m, ok := val.(map[string]any)
		if !ok {
			val = nil
			continue
		}
		res, exists := m[nameTok.value]
		if !exists {
			val = nil
			continue
		}
		if s, ok := res.(string); ok {
			// The source strips literal quote characters from any string
			// resolved through a "." access; preserved here for fidelity.
			res = strings.ReplaceAll(s, `"`, "")
		}
		val = res
	}
	return val, nil
}

// parseArray parses a "{" operand "}" array literal. A single string
// operand becomes a one-element array (not exploded into characters, which
// is what the Python source does due to strings satisfying its
// isinstance(x, Sequence) check — a bug this port doesn't reproduce).
func (c *evalContext) parseArray() (any, error) {
	if _, err := c.expect(tokLBrace, "'{'"); err != nil {
		return nil, err
	}
	val, err := c.parseOperandBranch()
	if err != nil {
		return nil, err
	}
	if _, err := c.expect(tokRBrace, "'}'"); err != nil {
		return nil, err
	}
	if t, ok := val.(Tuple); ok {
		return Array(t), nil
	}
	return Array{val}, nil
}

func (c *evalContext) parseClassCall() (any, error) {
	classTok, err := c.expect(tokClass, "a class name")
	if err != nil {
		return nil, err
	}
	if _, err := c.expect(tokDot, "'.'"); err != nil {
		return nil, err
	}
	methodTok, err := c.expect(tokName, "a method name")
	if err != nil {
		return nil, err
	}
	if _, err := c.expect(tokLParen, "'('"); err != nil {
		return nil, err
	}
	argVal, err := c.parseOperandBranch()
	if err != nil {
		return nil, err
	}
	if _, err := c.expect(tokRParen, "')'"); err != nil {
		return nil, err
	}

	class, ok := c.classes[classTok.value]
	if !ok {
		return nil, fmt.Errorf("class %q has not been implemented in the parser", classTok.value)
	}
	result, err := class.Call(methodTok.value, spreadArgs(argVal)...)
	if err != nil {
		return nil, fmt.Errorf("%s.%s: %w", classTok.value, methodTok.value, err)
	}
	return result, nil
}

func (c *evalContext) parseMemberOf() (any, error) {
	kindTok := c.advance()
	if _, err := c.expect(tokLParen, "'('"); err != nil {
		return nil, err
	}
	argVal, err := c.parseOperandBranch()
	if err != nil {
		return nil, err
	}
	if _, err := c.expect(tokRParen, "')'"); err != nil {
		return nil, err
	}

	switch kindTok.typ {
	case tokMemberOf:
		return c.evalMemberOf(argVal)
	case tokMemberOfAny:
		return c.evalMemberOfAny(argVal)
	case tokMemberOfName:
		return c.evalMemberOfGroupData(argVal, func(name, target string) bool { return name == target })
	case tokMemberOfGroupStartsWith:
		return c.evalMemberOfGroupData(argVal, strings.HasPrefix)
	case tokMemberOfGroupContains:
		return c.evalMemberOfGroupData(argVal, func(name, target string) bool { return strings.Contains(name, target) })
	case tokMemberOfGroupNameRegex:
		return c.evalMemberOfGroupNameRegex(argVal)
	}
	return nil, fmt.Errorf("unhandled isMemberOf builtin %q", kindTok.value)
}

// evalMemberOf implements isMemberOfGroup: true if arg is one of the
// parser's configured group IDs.
func (c *evalContext) evalMemberOf(arg any) (any, error) {
	if len(c.groupIDs) == 0 {
		return false, nil
	}
	s, ok := arg.(string)
	if !ok {
		return false, nil
	}
	for _, id := range c.groupIDs {
		if id == s {
			return true, nil
		}
	}
	return false, nil
}

// evalMemberOfAny implements isMemberOfAnyGroup: true if any of arg's
// (possibly multiple, comma-joined) values is one of the configured group
// IDs.
func (c *evalContext) evalMemberOfAny(arg any) (any, error) {
	if len(c.groupIDs) == 0 {
		return false, nil
	}
	ids := make(map[string]bool, len(c.groupIDs))
	for _, id := range c.groupIDs {
		ids[id] = true
	}
	for _, v := range spreadArgs(arg) {
		if s, ok := v.(string); ok && ids[s] {
			return true, nil
		}
	}
	return false, nil
}

// groupProfileName extracts group["profile"]["name"] from a group_data
// entry. Note this shape (nested under "profile") differs from what
// expressionclasses.Groups.getFilteredGroups expects (a flat map); that
// inconsistency exists in the source library and is preserved rather than
// papered over, since it's the caller's group_data to shape either way.
func groupProfileName(group any) (string, bool) {
	m, ok := group.(map[string]any)
	if !ok {
		return "", false
	}
	profile, ok := m["profile"].(map[string]any)
	if !ok {
		return "", false
	}
	name, ok := profile["name"].(string)
	return name, ok
}

// evalMemberOfGroupData implements the isMemberOfGroupName* family: true if
// match(groupName, arg) holds for any configured group's profile name.
func (c *evalContext) evalMemberOfGroupData(arg any, match func(groupName, target string) bool) (any, error) {
	target, ok := arg.(string)
	if !ok {
		return false, nil
	}
	for _, group := range c.groupData {
		if name, ok := groupProfileName(group); ok && match(name, target) {
			return true, nil
		}
	}
	return false, nil
}

func (c *evalContext) evalMemberOfGroupNameRegex(arg any) (any, error) {
	pattern, ok := arg.(string)
	if !ok {
		return false, nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("isMemberOfGroupNameRegex: invalid regular expression %q: %w", pattern, err)
	}
	for _, group := range c.groupData {
		name, ok := groupProfileName(group)
		if !ok {
			continue
		}
		if loc := re.FindStringIndex(name); loc != nil && loc[0] == 0 {
			return true, nil
		}
	}
	return false, nil
}

func unquoteString(raw string) string {
	if len(raw) >= 2 && (raw[0] == '"' || raw[0] == '\'') {
		return raw[1 : len(raw)-1]
	}
	return raw
}
