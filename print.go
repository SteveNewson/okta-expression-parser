package oktaexpr

import "strings"

// Format returns a multi-line, indented rendering of node, for reading a
// large generated expression — e.g. one of the compound OR-of-AND clauses
// a consumer might build to patch an Okta group rule. Every AND/OR chain
// (a Logical node, or a Not directly wrapping one) gets one operand per
// line, indented one level deeper than its enclosing chain, with the
// connective and the next operand's opening "(" sharing a line:
//
//	(
//	  Arrays.contains({"retail financial crime operations"}, String.toLowerCase(user.department))
//	  OR (
//	    Arrays.contains({"operations"}, String.toLowerCase(user.division))
//	    AND Arrays.contains({"shared services"}, String.toLowerCase(user.department))
//	  )
//	)
//
// Every other node type renders on a single line via its own String().
func Format(node Node) string {
	var b strings.Builder
	writeNode(&b, node, 0, "")
	return b.String()
}

// writeNode writes node at the given indent level (in units of two
// spaces), prefixed by prefix (e.g. "OR " for a chain's non-first operand,
// or "!" for a negated chain) on the same line as node's own opening.
func writeNode(b *strings.Builder, node Node, indent int, prefix string) {
	b.WriteString(strings.Repeat("  ", indent))
	b.WriteString(prefix)

	openPrefix, op, operands, ok := unwrapCompound(node)
	if !ok {
		b.WriteString(node.String())
		return
	}

	b.WriteString(openPrefix)
	b.WriteString("(\n")
	for i, operand := range operands {
		p := ""
		if i > 0 {
			p = op + " "
		}
		writeNode(b, operand, indent+1, p)
		b.WriteString("\n")
	}
	b.WriteString(strings.Repeat("  ", indent))
	b.WriteString(")")
}

// unwrapCompound reports whether node is a chain Format should expand
// across multiple lines — a Logical, or a Not directly wrapping one (so a
// negated compound clause, e.g. "!(a AND b)", still expands rather than
// rendering as one long line) — and if so, the prefix its opening "("
// needs ("" for a bare Logical, "!" for the negated form), the chain's
// connective, and its operands.
func unwrapCompound(node Node) (openPrefix, op string, operands []Node, ok bool) {
	switch v := node.(type) {
	case Logical:
		return "", v.Op, v.Operands, true
	case Not:
		if inner, ok := v.Operand.(Logical); ok {
			return "!", inner.Op, inner.Operands, true
		}
	}
	return "", "", nil, false
}
