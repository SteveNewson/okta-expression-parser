package oktaexpr

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/stevenewson/okta-expression-parser/expressionclasses"
)

// evalContext holds the evaluation environment for a single Eval call: the
// user profile, group memberships, and expression classes a Parser was
// configured with (see Parser's fields) — nothing here affects how an
// expression is parsed (see astparse.go), only how an already-built Node
// resolves against a specific user/org's data.
type evalContext struct {
	userProfile map[string]any
	groupIDs    []string
	groupData   map[string]any
	classes     expressionclasses.Registry
	strict      bool
}

// eval dispatches to the node-type-specific evaluator. Every node type
// astparse.go can produce is handled; a type outside that set (only
// possible from a hand-constructed Node) is a runtime error rather than a
// panic.
func (c *evalContext) eval(n Node) (any, error) {
	switch v := n.(type) {
	case Literal:
		return v.Value, nil
	case PathExpr:
		return c.evalPath(v)
	case MemberOfExpr:
		return c.evalMemberOf(v)
	case ClassCall:
		return c.evalClassCall(v)
	case ArrayLit:
		return c.evalArray(v)
	case CommaList:
		return c.evalCommaList(v)
	case Ternary:
		return c.evalTernary(v)
	case Comparison:
		return c.evalComparison(v)
	case Additive:
		return c.evalAdditive(v)
	case Logical:
		return c.evalLogical(v)
	case Not:
		return c.evalNot(v)
	default:
		return nil, fmt.Errorf("unsupported AST node type %T", n)
	}
}

// evalPath resolves a PathExpr's root plus its chain of hops. See
// PathExpr's doc comment for the "bare name always nil" quirk (RootUser
// false leaves val nil throughout, since only a "user"-rooted chain ever
// resolves against real profile data) and NameHop's for strict mode.
func (c *evalContext) evalPath(n PathExpr) (any, error) {
	var val any
	if n.RootUser {
		val = any(c.userProfile)
	}

	for _, hop := range n.Hops {
		switch h := hop.(type) {
		case MemberOfHop:
			memberVal, err := c.evalMemberOf(*h.Call)
			if err != nil {
				return nil, err
			}
			val = memberVal
		case NameHop:
			m, ok := val.(map[string]any)
			if !ok {
				val = nil
				continue
			}
			res, exists := m[h.Name]
			if !exists {
				if c.strict {
					return nil, fmt.Errorf("property %q does not exist at character %d", h.Name, h.Pos)
				}
				val = nil
				continue
			}
			if s, ok := res.(string); ok {
				// The source strips literal quote characters from any
				// string resolved through a "." access; preserved here for
				// fidelity.
				res = strings.ReplaceAll(s, `"`, "")
			}
			val = res
		}
	}
	return val, nil
}

func (c *evalContext) evalMemberOf(n MemberOfExpr) (any, error) {
	argVal, err := c.eval(n.Arg)
	if err != nil {
		return nil, err
	}

	switch n.Kind {
	case MemberOf:
		return c.evalIsMemberOf(argVal)
	case MemberOfAny:
		return c.evalIsMemberOfAny(argVal)
	case MemberOfName:
		return c.evalIsMemberOfGroupData(argVal, func(name, target string) bool { return name == target })
	case MemberOfGroupStartsWith:
		return c.evalIsMemberOfGroupData(argVal, strings.HasPrefix)
	case MemberOfGroupContains:
		return c.evalIsMemberOfGroupData(argVal, func(name, target string) bool { return strings.Contains(name, target) })
	case MemberOfGroupNameRegex:
		return c.evalIsMemberOfGroupNameRegex(argVal)
	default:
		return nil, fmt.Errorf("unhandled isMemberOf builtin %q", memberOfNames[n.Kind])
	}
}

// evalIsMemberOf implements isMemberOfGroup: true if arg is one of the
// parser's configured group IDs.
func (c *evalContext) evalIsMemberOf(arg any) (any, error) {
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

// evalIsMemberOfAny implements isMemberOfAnyGroup: true if any of arg's
// (possibly multiple, comma-joined) values is one of the configured group
// IDs.
func (c *evalContext) evalIsMemberOfAny(arg any) (any, error) {
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
// inconsistency exists in the Python source and is preserved rather than
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

// evalIsMemberOfGroupData implements the isMemberOfGroupName* family: true
// if match(groupName, arg) holds for any configured group's profile name.
func (c *evalContext) evalIsMemberOfGroupData(arg any, match func(groupName, target string) bool) (any, error) {
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

func (c *evalContext) evalIsMemberOfGroupNameRegex(arg any) (any, error) {
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

func (c *evalContext) evalClassCall(n ClassCall) (any, error) {
	class, ok := c.classes[n.Class]
	if !ok {
		return nil, fmt.Errorf("class %q has not been implemented in the parser", n.Class)
	}

	var args []any
	if n.Arg != nil {
		argVal, err := c.eval(n.Arg)
		if err != nil {
			return nil, err
		}
		args = spreadArgs(argVal)
	}

	result, err := class.Call(n.Method, args...)
	if err != nil {
		return nil, fmt.Errorf("%s.%s: %w", n.Class, n.Method, err)
	}
	return result, nil
}

func (c *evalContext) evalArray(n ArrayLit) (any, error) {
	elems := make(Array, len(n.Elements))
	for i, e := range n.Elements {
		v, err := c.eval(e)
		if err != nil {
			return nil, err
		}
		elems[i] = v
	}
	return elems, nil
}

// evalCommaList folds mergeOperands across its evaluated elements in
// order — identical to parseOperandBranch's incremental build in the old
// fused parser, just applied at Eval time instead of parse time.
func (c *evalContext) evalCommaList(n CommaList) (any, error) {
	var result any
	for i, e := range n.Elements {
		v, err := c.eval(e)
		if err != nil {
			return nil, err
		}
		if i == 0 {
			result = v
		} else {
			result = mergeOperands(result, v)
		}
	}
	return result, nil
}

// evalTernary evaluates Cond, True, and False unconditionally (no
// short-circuit — a preserved quirk, since the original bottom-up parser
// evaluated every subexpression as it reduced, before the ternary's own
// action ever ran) before picking a branch by Cond's truthiness.
func (c *evalContext) evalTernary(n Ternary) (any, error) {
	condVal, err := c.eval(n.Cond)
	if err != nil {
		return nil, err
	}
	trueVal, err := c.eval(n.True)
	if err != nil {
		return nil, err
	}
	falseVal, err := c.eval(n.False)
	if err != nil {
		return nil, err
	}

	if truthy(condVal) {
		return trueVal, nil
	}
	return falseVal, nil
}

func (c *evalContext) evalComparison(n Comparison) (any, error) {
	left, err := c.eval(n.Left)
	if err != nil {
		return nil, err
	}
	right, err := c.eval(n.Right)
	if err != nil {
		return nil, err
	}

	result, err := applyComparison(n.Op, left, right)
	if err != nil {
		return nil, fmt.Errorf("%s at character %d", err, n.Pos)
	}
	return result, nil
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

func (c *evalContext) evalAdditive(n Additive) (any, error) {
	left, err := c.eval(n.Left)
	if err != nil {
		return nil, err
	}
	right, err := c.eval(n.Right)
	if err != nil {
		return nil, err
	}

	result, err := addOperands(left, right)
	if err != nil {
		return nil, fmt.Errorf("%s at character %d", err, n.Pos)
	}
	return result, nil
}

// evalLogical evaluates every operand unconditionally, left to right,
// stopping at the first error — then reduces the results with the same
// truthy-chaining logic the old fused AND/OR loops applied incrementally.
// Evaluating every operand regardless of an earlier one's truthiness is
// the preserved "AND/OR never short-circuit" quirk; see the doc comment on
// Logical and the package README.
func (c *evalContext) evalLogical(n Logical) (any, error) {
	values := make([]any, len(n.Operands))
	for i, o := range n.Operands {
		v, err := c.eval(o)
		if err != nil {
			return nil, err
		}
		values[i] = v
	}

	result := values[0]
	for _, v := range values[1:] {
		switch n.Op {
		case "AND":
			if truthy(result) {
				result = v
			}
		case "OR":
			if !truthy(result) {
				result = v
			}
		}
	}
	return result, nil
}

func (c *evalContext) evalNot(n Not) (any, error) {
	v, err := c.eval(n.Operand)
	if err != nil {
		return nil, err
	}
	return !truthy(v), nil
}
