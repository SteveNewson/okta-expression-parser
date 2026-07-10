package oktaexpr_test

import (
	"testing"

	oktaexpr "github.com/stevenewson/okta-expression-parser"
)

func TestFormat_SingleNode(t *testing.T) {
	t.Parallel()

	node := oktaexpr.Comparison{Op: "==", Left: oktaexpr.PathExpr{RootUser: true, Hops: []oktaexpr.PathHop{oktaexpr.NameHop{Name: "department"}}}, Right: oktaexpr.Literal{Value: "Engineering"}}

	want := `user.department=="Engineering"`
	if got := oktaexpr.Format(node); got != want {
		t.Errorf("Format: got %q, want %q", got, want)
	}
}

func TestFormat_ORofAND(t *testing.T) {
	t.Parallel()

	base := oktaexpr.ClassCall{
		Class:  "Arrays",
		Method: "contains",
		Arg: oktaexpr.CommaList{Elements: []oktaexpr.Node{
			oktaexpr.ArrayLit{Elements: []oktaexpr.Node{oktaexpr.Literal{Value: "retail financial crime operations"}}},
			oktaexpr.ClassCall{Class: "String", Method: "toLowerCase", Arg: pathOf("department")},
		}},
	}
	divClause := oktaexpr.ClassCall{
		Class:  "Arrays",
		Method: "contains",
		Arg: oktaexpr.CommaList{Elements: []oktaexpr.Node{
			oktaexpr.ArrayLit{Elements: []oktaexpr.Node{oktaexpr.Literal{Value: "operations"}}},
			oktaexpr.ClassCall{Class: "String", Method: "toLowerCase", Arg: pathOf("division")},
		}},
	}
	deptClause := oktaexpr.ClassCall{
		Class:  "Arrays",
		Method: "contains",
		Arg: oktaexpr.CommaList{Elements: []oktaexpr.Node{
			oktaexpr.ArrayLit{Elements: []oktaexpr.Node{oktaexpr.Literal{Value: "shared services"}}},
			oktaexpr.ClassCall{Class: "String", Method: "toLowerCase", Arg: pathOf("department")},
		}},
	}

	node := oktaexpr.Logical{Op: "OR", Operands: []oktaexpr.Node{
		base,
		oktaexpr.Logical{Op: "AND", Operands: []oktaexpr.Node{divClause, deptClause}},
	}}

	want := `(
  Arrays.contains({"retail financial crime operations"}, String.toLowerCase(user.department))
  OR (
    Arrays.contains({"operations"}, String.toLowerCase(user.division))
    AND Arrays.contains({"shared services"}, String.toLowerCase(user.department))
  )
)`
	if got := oktaexpr.Format(node); got != want {
		t.Errorf("Format:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormat_NegatedChainExpands(t *testing.T) {
	t.Parallel()

	node := oktaexpr.Not{Operand: oktaexpr.Logical{Op: "AND", Operands: []oktaexpr.Node{
		oktaexpr.Comparison{Op: "==", Left: pathOf("a"), Right: oktaexpr.Literal{Value: "x"}},
		oktaexpr.Comparison{Op: "==", Left: pathOf("b"), Right: oktaexpr.Literal{Value: "y"}},
	}}}

	want := `!(
  user.a=="x"
  AND user.b=="y"
)`
	if got := oktaexpr.Format(node); got != want {
		t.Errorf("Format:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

// pathOf is a small test helper for a "user.<name>" PathExpr.
func pathOf(name string) oktaexpr.PathExpr {
	return oktaexpr.PathExpr{RootUser: true, Hops: []oktaexpr.PathHop{oktaexpr.NameHop{Name: name}}}
}
