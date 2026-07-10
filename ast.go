package oktaexpr

import (
	"fmt"
	"strconv"
	"strings"
)

// Node is any Okta Expression Language AST node, as returned by
// (*Parser).Parse and consumed by (*Parser).Eval. The concrete types are
// Literal, PathExpr, MemberOfExpr, ClassCall, ArrayLit, CommaList, Ternary,
// Comparison, Additive, Logical, and Not.
//
// Node is deliberately not a sealed interface: callers can construct any of
// these types directly (e.g. to synthesize a replacement clause when
// rewriting a parsed expression) rather than only ever receiving them from
// Parse.
type Node interface {
	// String returns expr's canonical, valid Okta Expression Language
	// re-serialization. This is not necessarily byte-identical to whatever
	// text originally parsed to this node: parenthesization is re-derived
	// from operator precedence (and the operand/condition typing rule
	// below), not preserved verbatim.
	String() string
}

// Literal is a constant: an int, float64, string, bool, or nil (an Okta
// expression "null").
type Literal struct {
	Value any
}

func (n Literal) String() string {
	switch v := n.Value.(type) {
	case nil:
		return "null"
	case bool:
		if v {
			return "true"
		}
		return "false"
	case int:
		return strconv.Itoa(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case string:
		return quoteLiteral(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// quoteLiteral double-quotes s for re-serialization, escaping backslashes
// and double quotes so the result is always a valid string literal
// regardless of what s contains.
func quoteLiteral(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '"' || c == '\\' {
			b.WriteByte('\\')
		}
		b.WriteByte(c)
	}
	b.WriteByte('"')
	return b.String()
}

// PathHop is one "." access in a PathExpr's chain: either a plain property
// name (NameHop) or an embedded user.isMemberOfX(...) method call
// (MemberOfHop) — see PathExpr's doc comment.
type PathHop interface {
	hopString() string
}

// NameHop is a plain ".name" property access. Pos is the name token's
// position in the original source (0 for a hand-constructed node), used
// only to reproduce strict mode's "property does not exist at character
// %d" error text.
type NameHop struct {
	Name string
	Pos  int
}

func (h NameHop) hopString() string { return h.Name }

// MemberOfHop is an embedded user.isMemberOfX(...) call reached mid-chain,
// e.g. the "isMemberOfGroupName(...)" part of
// "user.isMemberOfGroupName(\"x\")". Unlike the bare isMemberOf* form
// (MemberOfExpr used directly as a Node, always condition-typed), this
// spelling is operand-typed, since it's reached through the path-chain
// production rather than parseAtom's condition-typed dispatch.
type MemberOfHop struct {
	Call *MemberOfExpr
}

func (h MemberOfHop) hopString() string { return h.Call.String() }

// PathExpr is a "." chain: "user.a.b.c", or a bare name (RootUser false),
// which — per a preserved quirk of the source grammar — always evaluates
// to nil regardless of Hops, since only "user"-rooted chains ever resolve
// against real profile data. RootName is the bare name's own text, kept
// only so String() can re-serialize it; it plays no role in evaluation.
type PathExpr struct {
	RootUser bool
	RootName string
	Hops     []PathHop
}

func (n PathExpr) String() string {
	var b strings.Builder
	if n.RootUser {
		b.WriteString("user")
	} else {
		b.WriteString(n.RootName)
	}
	for _, h := range n.Hops {
		b.WriteByte('.')
		b.WriteString(h.hopString())
	}
	return b.String()
}

// MemberOfKind identifies which isMemberOf* builtin a MemberOfExpr invokes.
type MemberOfKind int

const (
	MemberOf MemberOfKind = iota
	MemberOfAny
	MemberOfName
	MemberOfGroupStartsWith
	MemberOfGroupContains
	MemberOfGroupNameRegex
)

// memberOfNames maps each MemberOfKind to its Okta Expression Language
// function name.
var memberOfNames = map[MemberOfKind]string{
	MemberOf:                "isMemberOfGroup",
	MemberOfAny:             "isMemberOfAnyGroup",
	MemberOfName:            "isMemberOfGroupName",
	MemberOfGroupStartsWith: "isMemberOfGroupNameStartsWith",
	MemberOfGroupContains:   "isMemberOfGroupNameContains",
	MemberOfGroupNameRegex:  "isMemberOfGroupNameRegex",
}

// MemberOfExpr is a call to one of the isMemberOf* group-membership
// builtins. Used directly as a Node for the bare "isMemberOfGroup(...)"
// form (always condition-typed — see parseAtom); embedded in a PathExpr's
// Hops (via MemberOfHop) for the "user.isMemberOfGroupName(...)" method-call
// form (operand-typed).
type MemberOfExpr struct {
	Kind MemberOfKind
	Arg  Node
}

func (n MemberOfExpr) String() string {
	return fmt.Sprintf("%s(%s)", memberOfNames[n.Kind], n.Arg.String())
}

// ClassCall is a "Class.method(args)" expression, e.g.
// Arrays.contains({"a"}, user.department). Arg is nil for a call with no
// arguments, a single Node for one argument, or a CommaList for more than
// one.
type ClassCall struct {
	Class  string
	Method string
	Arg    Node
}

func (n ClassCall) String() string {
	arg := ""
	if n.Arg != nil {
		arg = n.Arg.String()
	}
	return fmt.Sprintf("%s.%s(%s)", n.Class, n.Method, arg)
}

// ArrayLit is a "{" operand "}" array literal, e.g. {1, 2, 3}.
type ArrayLit struct {
	Elements []Node
}

func (n ArrayLit) String() string {
	parts := make([]string, len(n.Elements))
	for i, e := range n.Elements {
		parts[i] = e.String()
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

// CommaList is a comma-joined group of operands, produced by the comma
// operator at any operand position (a ternary branch, array-literal
// contents, or a class-method/isMemberOf* call's argument list) — see
// mergeOperands/spreadArgs and parseOperandBranch's doc comment for the
// "binds to the innermost operand context" quirk this preserves. Every
// element is individually guaranteed operand-typed by construction.
type CommaList struct {
	Elements []Node
}

func (n CommaList) String() string {
	parts := make([]string, len(n.Elements))
	for i, e := range n.Elements {
		parts[i] = e.String()
	}
	return strings.Join(parts, ", ")
}

// Ternary is a "cond ? trueVal : falseVal" expression. Always
// operand-typed, regardless of what Cond, True, or False themselves are —
// a preserved quirk of the source grammar (see the README).
type Ternary struct {
	Cond, True, False Node
}

func (n Ternary) String() string {
	return fmt.Sprintf("%s ? %s : %s", n.Cond.String(), n.True.String(), n.False.String())
}

// Comparison is a "left OP right" relational expression: ==, !=, >, >=, <,
// or <=. Both operands must be operand-typed. Pos is the operator token's
// position in the original source (0 for a hand-constructed node), used
// only in the error a mismatched-type comparison (e.g. "1" > 2) produces.
type Comparison struct {
	Op          string
	Left, Right Node
	Pos         int
}

func (n Comparison) String() string {
	return fmt.Sprintf("%s%s%s", n.Left.String(), n.Op, n.Right.String())
}

// Additive is a "left + right" expression: string concatenation for two
// strings, arithmetic addition for two ints or two floats. Pos is the "+"
// token's position in the original source (0 for a hand-constructed
// node), used only in the error a mismatched-type "+" produces.
type Additive struct {
	Left, Right Node
	Pos         int
}

func (n Additive) String() string {
	return fmt.Sprintf("%s + %s", n.Left.String(), n.Right.String())
}

// Logical is a whole AND-chain or OR-chain ("a AND b AND c"), flattened
// into Operands — only ever constructed when at least one AND/OR token is
// present; a lone operand with no operator is returned unwrapped instead
// (see astparse.go), matching the source grammar's passthrough behavior.
// Op is "AND" or "OR". Always condition-typed.
type Logical struct {
	Op       string
	Operands []Node
}

func (n Logical) String() string {
	parts := make([]string, len(n.Operands))
	for i, o := range n.Operands {
		parts[i] = maybeParen(o)
	}
	return strings.Join(parts, " "+n.Op+" ")
}

// Not is a "!operand" (or "not operand") negation. Always condition-typed,
// even when Operand itself isn't — "!5" is condition-typed today, a real,
// preserved quirk of the source grammar.
type Not struct {
	Operand Node
}

func (n Not) String() string {
	return "!" + maybeParen(n.Operand)
}

// maybeParen wraps n's String() in parentheses when n is a construct whose
// own textual form could otherwise be mis-grouped by the enclosing
// operator — namely a nested Logical (AND inside OR, or vice versa) or a
// Ternary. Every other node type's own syntax (a call's parens, a
// comparison's tight binding, etc.) already disambiguates itself with no
// extra parens needed.
func maybeParen(n Node) string {
	switch n.(type) {
	case Logical, Ternary:
		return "(" + n.String() + ")"
	default:
		return n.String()
	}
}
