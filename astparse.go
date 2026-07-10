package oktaexpr

import (
	"fmt"
	"strconv"
	"strings"
)

// astParser holds the token stream for a single Parse call and builds an
// AST from it — the same recursive-descent grammar this library has always
// used, except each parse* method here returns the Node it built rather
// than evaluating it immediately. Building the AST needs none of Parser's
// configured state
// (user profile, group IDs, group data, strict mode, expression-class
// registry): none of that affects how an expression is *structured*, only
// how a parsed Node is later *evaluated* (see asteval.go) — the registry
// lookup for a Class.method(...) call, for instance, only happens at Eval
// time, since the lexer's fixed keyword table already tells the parser
// which five names are legal CLASS tokens without consulting any registry.
//
// The bool each method returns alongside its Node reports whether the
// parsed construct is "condition"-typed (the result of a comparison,
// AND/OR, NOT, or a bare isMemberOf* call) as opposed to "operand"-typed (a
// plain value) — see parseExpr's doc comment. This is a parse-time
// legality concern (a ternary branch, a comparison operand, or a "+"
// operand must be operand-typed, or it's a genuine parse error) preserved
// exactly from grammar.go; Eval never needs to know a Node's operand/
// condition-ness once it's survived these checks.
type astParser struct {
	toks []token
	pos  int
}

func (c *astParser) peek() token {
	return c.toks[c.pos]
}

func (c *astParser) advance() token {
	t := c.toks[c.pos]
	if c.pos < len(c.toks)-1 {
		c.pos++
	}
	return t
}

func (c *astParser) expect(typ tokenType, desc string) (token, error) {
	tok := c.peek()
	if tok.typ != typ {
		return token{}, fmt.Errorf("expected %s but found %q at character %d", desc, tok.value, tok.pos)
	}
	return c.advance(), nil
}

// parseToAST tokenizes expression and builds its AST — Parse's
// implementation.
func parseToAST(expression string) (Node, error) {
	toks, err := tokenize(expression)
	if err != nil {
		return nil, err
	}
	c := &astParser{toks: toks}
	node, _, err := c.parseExpr()
	if err != nil {
		return nil, err
	}
	if c.peek().typ != tokEOF {
		tok := c.peek()
		return nil, fmt.Errorf("unexpected token %q at character %d", tok.value, tok.pos)
	}
	return node, nil
}

// parseExpr parses a full condition-or-operand expression, including the
// ternary operator, which is the loosest-binding construct in the grammar.
func (c *astParser) parseExpr() (Node, bool, error) {
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

	// A ternary is always operand-typed, regardless of its branches — see
	// Ternary's doc comment.
	return Ternary{Cond: val, True: trueVal, False: falseVal}, false, nil
}

// parseOperandBranch parses an expression that must be operand-typed, used
// for ternary branches, array literal contents, and class-method/
// isMemberOf* call arguments. It also greedily chains a trailing comma into
// a CommaList (matching the source grammar's "operand , operand"
// production, present at every operand position) — see CommaList's doc
// comment for the "binds to the innermost operand context" quirk this
// preserves.
func (c *astParser) parseOperandBranch() (Node, error) {
	val, err := c.parseOperandBranchItem()
	if err != nil {
		return nil, err
	}
	if c.peek().typ != tokComma {
		return val, nil
	}

	elements := []Node{val}
	for c.peek().typ == tokComma {
		c.advance()
		rhs, err := c.parseOperandBranchItem()
		if err != nil {
			return nil, err
		}
		elements = append(elements, rhs)
	}
	return CommaList{Elements: elements}, nil
}

func (c *astParser) parseOperandBranchItem() (Node, error) {
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

func (c *astParser) parseOr() (Node, bool, error) {
	return c.parseLogicalChain(tokOr, "OR", c.parseAnd)
}

func (c *astParser) parseAnd() (Node, bool, error) {
	return c.parseLogicalChain(tokAnd, "AND", c.parseNot)
}

// parseLogicalChain parses a chain of two or more operands joined purely by
// connective (AND or OR), calling next to parse each operand. A Logical
// node is only built when at least one connective token is actually
// present; with none, the single operand is returned unwrapped, matching
// the source grammar's passthrough behavior — a bare `parseAnd`/`parseOr`
// call with no operator found returns its child's own isCond unchanged.
func (c *astParser) parseLogicalChain(tok tokenType, op string, next func() (Node, bool, error)) (Node, bool, error) {
	val, isCond, err := next()
	if err != nil {
		return nil, false, err
	}
	if c.peek().typ != tok {
		return val, isCond, nil
	}

	operands := []Node{val}
	for c.peek().typ == tok {
		c.advance()
		rhs, _, err := next()
		if err != nil {
			return nil, false, err
		}
		operands = append(operands, rhs)
	}
	return Logical{Op: op, Operands: operands}, true, nil
}

func (c *astParser) parseNot() (Node, bool, error) {
	if c.peek().typ == tokNot {
		c.advance()
		val, _, err := c.parseNot()
		if err != nil {
			return nil, false, err
		}
		return Not{Operand: val}, true, nil
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

// parseComparison implements the grammar's nonassoc EQ/NE/GT/GE/LT/LE
// level. Both sides must be operand-typed, matching the source, which only
// ever defines these operators over "operand EQ operand" (never over a
// "condition").
func (c *astParser) parseComparison() (Node, bool, error) {
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

	return Comparison{Op: opName, Left: val, Right: rhs, Pos: opTok.pos}, true, nil
}

// parseAdditive implements the "+" operator: string concatenation, or
// arithmetic addition for two ints or two floats. It binds tighter than
// comparisons but looser than a primary/isMemberOf* atom. Both sides must
// be operand-typed, matching the strictness of the comparison operators
// above.
func (c *astParser) parseAdditive() (Node, bool, error) {
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

		val = Additive{Left: val, Right: rhs, Pos: plusTok.pos}
	}
	return val, isCond, nil
}

// parseAtom parses the atoms that comparisons operate on: primaries,
// isMemberOf* builtins (always condition-typed), and parenthesized
// expressions (which preserve whatever type was inside).
func (c *astParser) parseAtom() (Node, bool, error) {
	switch {
	case c.peek().typ == tokLParen:
		c.advance()
		val, isCond, err := c.parseExpr()
		if err != nil {
			return nil, false, err
		}
		if _, err := c.expect(tokRParen, "')'"); err != nil {
			return nil, false, err
		}
		return val, isCond, nil
	case isMemberOfToken(c.peek().typ):
		val, err := c.parseMemberOf()
		return val, true, err
	default:
		val, err := c.parsePrimary()
		return val, false, err
	}
}

func (c *astParser) parsePrimary() (Node, error) {
	tok := c.peek()
	switch tok.typ {
	case tokInt:
		c.advance()
		n, err := strconv.Atoi(tok.value)
		if err != nil {
			return nil, fmt.Errorf("invalid integer literal %q at character %d", tok.value, tok.pos)
		}
		return Literal{Value: n}, nil
	case tokFloat:
		c.advance()
		f, err := strconv.ParseFloat(tok.value, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid number literal %q at character %d", tok.value, tok.pos)
		}
		return Literal{Value: f}, nil
	case tokString:
		c.advance()
		return Literal{Value: unquoteString(tok.value)}, nil
	case tokBool:
		c.advance()
		return Literal{Value: strings.EqualFold(tok.value, "true")}, nil
	case tokNull:
		c.advance()
		return Literal{Value: nil}, nil
	case tokLBrace:
		return c.parseArray()
	case tokUser:
		c.advance()
		return c.parsePathChain(true, "")
	case tokName:
		// A bare NAME that isn't part of a user.<path> chain always
		// resolves to nil at eval time — see PathExpr's doc comment.
		c.advance()
		return c.parsePathChain(false, tok.value)
	case tokClass:
		return c.parseClassCall()
	default:
		return nil, fmt.Errorf("unexpected token %q at character %d", tok.value, tok.pos)
	}
}

// isMemberOfToken reports whether typ is one of the isMemberOf* builtin
// tokens, used both to dispatch the bare-call form in parseAtom and the
// user.isMemberOf...(...) method-call form in parsePathChain.
func isMemberOfToken(typ tokenType) bool {
	switch typ {
	case tokMemberOf, tokMemberOfAny, tokMemberOfName, tokMemberOfGroupStartsWith, tokMemberOfGroupContains, tokMemberOfGroupNameRegex:
		return true
	default:
		return false
	}
}

// parsePathChain builds a PathExpr from a (possibly empty) chain of
// ".name" (or ".isMemberOfX(...)") accesses following "user" (rootUser
// true) or a bare NAME (rootUser false, rootName its text).
func (c *astParser) parsePathChain(rootUser bool, rootName string) (Node, error) {
	path := PathExpr{RootUser: rootUser, RootName: rootName}

	for c.peek().typ == tokDot {
		c.advance()

		if isMemberOfToken(c.peek().typ) {
			call, err := c.parseMemberOf()
			if err != nil {
				return nil, err
			}
			memberOf, _ := call.(MemberOfExpr)
			path.Hops = append(path.Hops, MemberOfHop{Call: &memberOf})
			continue
		}

		nameTok, err := c.expect(tokName, "a property name")
		if err != nil {
			return nil, err
		}
		path.Hops = append(path.Hops, NameHop{Name: nameTok.value, Pos: nameTok.pos})
	}
	return path, nil
}

// parseArray parses a "{" operand "}" array literal.
func (c *astParser) parseArray() (Node, error) {
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

	if list, ok := val.(CommaList); ok {
		return ArrayLit{Elements: list.Elements}, nil
	}
	return ArrayLit{Elements: []Node{val}}, nil
}

func (c *astParser) parseClassCall() (Node, error) {
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

	return ClassCall{Class: classTok.value, Method: methodTok.value, Arg: argVal}, nil
}

func (c *astParser) parseMemberOf() (Node, error) {
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

	kind, ok := memberOfKindForToken[kindTok.typ]
	if !ok {
		return nil, fmt.Errorf("unhandled isMemberOf builtin %q", kindTok.value)
	}
	return MemberOfExpr{Kind: kind, Arg: argVal}, nil
}

var memberOfKindForToken = map[tokenType]MemberOfKind{
	tokMemberOf:                MemberOf,
	tokMemberOfAny:             MemberOfAny,
	tokMemberOfName:            MemberOfName,
	tokMemberOfGroupStartsWith: MemberOfGroupStartsWith,
	tokMemberOfGroupContains:   MemberOfGroupContains,
	tokMemberOfGroupNameRegex:  MemberOfGroupNameRegex,
}

func unquoteString(raw string) string {
	if len(raw) >= 2 && (raw[0] == '"' || raw[0] == '\'') {
		return raw[1 : len(raw)-1]
	}
	return raw
}
