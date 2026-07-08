package oktaexpr_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	oktaexpr "github.com/stevenewson/okta-expression-parser"
	"github.com/stevenewson/okta-expression-parser/expressionclasses"
	"github.com/stevenewson/okta-expression-parser/values"
)

// TestParse_Literals, TestParse_Comparisons, etc. below are table-driven
// against expected results verified against the real Python
// okta-expression-parser library (see the deviations noted in the README
// for the handful of cases where this port deliberately differs).

func TestParse_Literals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		expr string
		want any
	}{
		{"positive int", "1", 1},
		{"multi-digit int", "123", 123},
		{"float", "3.141", 3.141},
		{"double-quoted string", `"hello"`, "hello"},
		{"empty string", `""`, ""},
		{"true lowercase", "true", true},
		{"True capitalized", "True", true},
		{"TRUE uppercase", "TRUE", true},
		{"false lowercase", "false", false},
		{"null", "null", nil},
		{"NULL uppercase", "NULL", nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Given
			p := oktaexpr.New()

			// When
			got, err := p.Parse(tc.expr)

			// Then
			if err != nil {
				t.Fatalf("Parse(%q): unexpected error %v", tc.expr, err)
			}
			if !values.EqualOperands(got, tc.want) {
				t.Errorf("Parse(%q): got %#v, want %#v", tc.expr, got, tc.want)
			}
		})
	}
}

func TestParse_Comparisons(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		expr string
		want bool
	}{
		{"int equal", "1 == 1", true},
		{"int not equal via ==", "1 == 2", false},
		{"string equal", `"a" == "a"`, true},
		{"single-quoted string equal", `'a' == 'a'`, true},
		{"single- and double-quoted string equal", `'a' == "a"`, true},
		{"single-quoted string containing a double quote", `'a"b' == 'a"b'`, true},
		{"int not-equal true", "1 != 2", true},
		{"int not-equal false", "1 != 1", false},
		{"gt false", "1 > 2", false},
		{"gt true", "2 > 1", true},
		{"gte equal", "2 >= 2", true},
		{"lt true", "1 < 2", true},
		// The Python source's "<=" is dead code: GTE/GT/LT/LTE are checked
		// in an order (GTE, GT, LT, LTE) that means LT always wins over LTE
		// for a leading "<", so "<=" tokenizes as LT followed by a stray "="
		// and fails to parse at all. This port fixes the lexer to check
		// both two-character operators before falling back to the
		// single-character ones, so <= genuinely works.
		{"lte equal (fixed dead-code bug)", "2 <= 2", true},
		{"lte less", "1 <= 2", true},
		{"lte false", "2 <= 1", false},
		{"bool equal", "true == true", true},
		{"bool equal false", "false == false", true},
		{"null equal null", "null == null", true},
		{"string ordering greater", `"b" > "a"`, true},
		{"string ordering not greater", `"a" > "b"`, false},
		{"int vs string never equal", `1 == "1"`, false},
		{"bool true equals int one", "true == 1", true},
		{"int one equals bool true", "1 == true", true},
		{"bool does not equal string", `true == "true"`, false},
		{"null does not equal false", "null == false", false},
		{"null does not equal zero", "null == 0", false},
		// GT/GE/LT/LE require exact type equality, unlike ==: bool and int
		// are not the same type for ordering purposes even though True==1.
		{"bool vs int strict type mismatch for gte", "true >= 1", false},
		{"bool vs bool ordering", "true > false", true},
		{"bool vs bool ordering reverse", "false < true", true},
		{"int vs string mismatch for gt", `1 > "1"`, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Given
			p := oktaexpr.New()

			// When
			got, err := p.Parse(tc.expr)

			// Then
			if err != nil {
				t.Fatalf("Parse(%q): unexpected error %v", tc.expr, err)
			}
			if got != tc.want {
				t.Errorf("Parse(%q): got %#v, want %v", tc.expr, got, tc.want)
			}
		})
	}
}

func TestParse_Comparison_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		expr string
	}{
		{"nil has no ordering", "null >= null"},
		{"isMemberOf result cannot be compared with ==", `isMemberOfGroup("g1") == true`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Given
			p := oktaexpr.New(oktaexpr.WithGroupIDs([]string{"g1"}))

			// When
			_, err := p.Parse(tc.expr)

			// Then
			if err == nil {
				t.Errorf("Parse(%q): got nil error, want an error", tc.expr)
			}
		})
	}
}

func TestParse_AdditiveOperator(t *testing.T) {
	t.Parallel()

	profile := map[string]any{"firstName": "Winston", "lastName": "Churchill"}

	tests := []struct {
		name string
		expr string
		want any
	}{
		{"string concatenation", `user.firstName + user.lastName`, "WinstonChurchill"},
		{"string concatenation with literal separator", `user.firstName + " " + user.lastName`, "Winston Churchill"},
		{"chained concatenation", `"a" + "b" + "c"`, "abc"},
		{"int addition", "1 + 2", 3},
		{"float addition", "1.5 + 2.5", 4.0},
		{"binds tighter than comparison", `user.firstName + user.lastName == "WinstonChurchill"`, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Given
			p := oktaexpr.New(oktaexpr.WithUserProfile(profile))

			// When
			got, err := p.Parse(tc.expr)

			// Then
			if err != nil {
				t.Fatalf("Parse(%q): unexpected error %v", tc.expr, err)
			}
			if got != tc.want {
				t.Errorf("Parse(%q): got %#v, want %#v", tc.expr, got, tc.want)
			}
		})
	}
}

func TestParse_AdditiveOperator_MismatchedTypesIsError(t *testing.T) {
	t.Parallel()

	p := oktaexpr.New()

	_, err := p.Parse(`1 + "a"`)
	if err == nil {
		t.Errorf(`Parse(1 + "a"): got nil error, want an error`)
	}
}

func TestParse_LogicalOperators(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		expr string
		want any
	}{
		{"and lowercase", "true and false", false},
		{"AND uppercase", "true AND false", false},
		{"double ampersand", "true && false", false},
		{"or lowercase", "true or false", true},
		{"double pipe", "true || false", true},
		{"not lowercase", "not true", false},
		{"NOT uppercase", "NOT false", true},
		{"bang", "!true", false},
		{"and binds tighter than or, left", "true and true or false", true},
		{"and binds tighter than or, right", "false or true and false", false},
		{"parens override precedence", "(true or false) and false", false},
		// Python's and/or return the actual short-circuited operand, not a
		// coerced bool; this port preserves that.
		{"and returns second truthy operand", "1 and 2", 2},
		{"and returns first falsy operand", "0 and 2", 0},
		{"or returns first truthy operand", "2 or 0", 2},
		{"or returns second operand when first falsy", "0 or 2", 2},
		{"not always yields strict bool: not comparison", "not 1 == 1", false},
		{"not binds tighter than and", "not true and false", false},
		{"not binds tighter than eq is false: not 1==2", "not 1 == 2", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Given
			p := oktaexpr.New()

			// When
			got, err := p.Parse(tc.expr)

			// Then
			if err != nil {
				t.Fatalf("Parse(%q): unexpected error %v", tc.expr, err)
			}
			if !values.EqualOperands(got, tc.want) {
				t.Errorf("Parse(%q): got %#v, want %#v", tc.expr, got, tc.want)
			}
		})
	}
}

func TestParse_LogicalOperators_EagerEvaluation(t *testing.T) {
	t.Parallel()

	// Given: the grammar evaluates every subexpression as it parses
	// (mirroring the source's bottom-up yacc reductions), so AND/OR/ternary
	// never short-circuit around an erroring subexpression, even when its
	// value wouldn't matter to the final result.
	tests := []struct {
		name string
		expr string
	}{
		{"and does not short-circuit an erroring rhs", "false and (null >= null)"},
		{"or does not short-circuit an erroring rhs", "true or (null >= null)"},
		{"ternary evaluates the branch not taken", "true ? 1 : (null >= null)"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Given
			p := oktaexpr.New()

			// When
			_, err := p.Parse(tc.expr)

			// Then
			if err == nil {
				t.Errorf("Parse(%q): got nil error, want an error from the unevaluated-looking branch", tc.expr)
			}
		})
	}
}

func TestParse_Ternary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		expr string
		want any
	}{
		{"true branch", "true ? 1 : 2", 1},
		{"false branch", "false ? 1 : 2", 2},
		{"condition from comparison", `1 == 1 ? "yes" : "no"`, "yes"},
		{"nested ternary in true branch", "true ? false ? 1 : 2 : 3", 2},
		{"and binds tighter than ternary", "true and false ? 1 : 2", 2},
		{"comparison and and bind tighter than ternary", "1 == 1 and 2 == 3 ? 1 : 2", 2},
		{"or binds tighter than ternary", "true or false ? 1 : 2", 1},
		{"truthy int as ternary condition", `1 ? "yes" : "no"`, "yes"},
		{"falsy int as ternary condition", `0 ? "yes" : "no"`, "no"},
		{"truthy string as ternary condition", `"str" ? "yes" : "no"`, "yes"},
		{"isMemberOf result as ternary condition", `isMemberOfGroup("g1") ? "y" : "n"`, "y"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Given
			p := oktaexpr.New(oktaexpr.WithGroupIDs([]string{"g1"}))

			// When
			got, err := p.Parse(tc.expr)

			// Then
			if err != nil {
				t.Fatalf("Parse(%q): unexpected error %v", tc.expr, err)
			}
			if !values.EqualOperands(got, tc.want) {
				t.Errorf("Parse(%q): got %#v, want %#v", tc.expr, got, tc.want)
			}
		})
	}
}

func TestParse_Ternary_BranchMustBeOperandTyped(t *testing.T) {
	t.Parallel()

	// Given: the grammar requires ternary branches to be "operand"-typed;
	// a bare comparison or parenthesized comparison is "condition"-typed and
	// is rejected even though it evaluates to a plain bool, matching a real,
	// verified restriction in the Python source (`1==1 ? 1 : 2==3` fails to
	// parse there too).
	tests := []struct {
		name string
		expr string
	}{
		{"bare comparison as false branch", "1 == 1 ? 1 : 2 == 3"},
		{"parenthesized comparison as true branch", "true ? (1==1) : 2"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Given
			p := oktaexpr.New()

			// When
			_, err := p.Parse(tc.expr)

			// Then
			if err == nil {
				t.Errorf("Parse(%q): got nil error, want an error", tc.expr)
			}
		})
	}
}

func TestParse_UserProfilePaths(t *testing.T) {
	t.Parallel()

	profile := map[string]any{
		"location": "US",
		"userName": "SU",
		"nested":   map[string]any{"a": map[string]any{"b": "c"}},
		"zero":     0,
	}

	tests := []struct {
		name string
		expr string
		want any
	}{
		{"single-level attribute", "user.location", "US"},
		{"comparison against attribute", `user.location == "US"`, true},
		{"comparison against attribute, single-quoted", `user.location == 'US'`, true},
		{"missing attribute is null", "user.missing == null", true},
		{"bare name (not user.) is always null, matching source quirk", "foo", nil},
		{"combined attribute comparison", `user.location == "US" and user.userName == "SU"`, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Given
			p := oktaexpr.New(oktaexpr.WithUserProfile(profile))

			// When
			got, err := p.Parse(tc.expr)

			// Then
			if err != nil {
				t.Fatalf("Parse(%q): unexpected error %v", tc.expr, err)
			}
			if !values.EqualOperands(got, tc.want) {
				t.Errorf("Parse(%q): got %#v, want %#v", tc.expr, got, tc.want)
			}
		})
	}
}

// TestParse_DeepUserProfilePaths verifies a deliberate deviation from the
// Python source: its grammar only supports a single "." hop after "user"
// (user.location works, but user.nested.a fails to parse, returning nil
// after logging a syntax error). This port instead resolves arbitrarily
// deep chains, matching real Okta Expression Language behavior.
func TestParse_DeepUserProfilePaths(t *testing.T) {
	t.Parallel()

	profile := map[string]any{
		"outer": map[string]any{"inner": map[string]any{"leaf": "value"}},
	}

	tests := []struct {
		name string
		expr string
		want any
	}{
		{"two levels deep", "user.outer.inner", map[string]any{"leaf": "value"}},
		{"three levels deep", "user.outer.inner.leaf", "value"},
		{"resolving through a non-map segment yields nil", "user.outer.inner.leaf.anything", nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Given
			p := oktaexpr.New(oktaexpr.WithUserProfile(profile))

			// When
			got, err := p.Parse(tc.expr)

			// Then
			if err != nil {
				t.Fatalf("Parse(%q): unexpected error %v", tc.expr, err)
			}
			// EqualOperands doesn't special-case map[string]any (path
			// resolution results, unlike expression values, aren't
			// compared by the language itself), so fall back to
			// reflect.DeepEqual here.
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Parse(%q): got %#v, want %#v", tc.expr, got, tc.want)
			}
		})
	}
}

func TestParse_MemberOfGroup(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		expr     string
		groupIDs []string
		want     bool
	}{
		{"member", `isMemberOfGroup("00g1")`, []string{"00g1", "00g2"}, true},
		{"not a member", `isMemberOfGroup("00g3")`, []string{"00g1", "00g2"}, false},
		{"no configured groups is always false", `isMemberOfGroup("00g1")`, nil, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Given
			p := oktaexpr.New(oktaexpr.WithGroupIDs(tc.groupIDs))

			// When
			got, err := p.Parse(tc.expr)

			// Then
			if err != nil {
				t.Fatalf("Parse(%q): unexpected error %v", tc.expr, err)
			}
			if got != tc.want {
				t.Errorf("Parse(%q): got %v, want %v", tc.expr, got, tc.want)
			}
		})
	}
}

func TestParse_MemberOfAnyGroup(t *testing.T) {
	t.Parallel()

	groupIDs := []string{"00g1", "00g2"}

	tests := []struct {
		name string
		expr string
		want bool
	}{
		{"one of two matches", `isMemberOfAnyGroup("00g1", "00g3")`, true},
		{"none match", `isMemberOfAnyGroup("00g3", "00g4")`, false},
		{"single arg matches", `isMemberOfAnyGroup("00g1")`, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Given
			p := oktaexpr.New(oktaexpr.WithGroupIDs(groupIDs))

			// When
			got, err := p.Parse(tc.expr)

			// Then
			if err != nil {
				t.Fatalf("Parse(%q): unexpected error %v", tc.expr, err)
			}
			if got != tc.want {
				t.Errorf("Parse(%q): got %v, want %v", tc.expr, got, tc.want)
			}
		})
	}
}

func TestParse_MemberOfGroupNameFamily(t *testing.T) {
	t.Parallel()

	groupData := map[string]any{
		"00g1": map[string]any{"profile": map[string]any{"name": "Engineering Team"}},
		"00g2": map[string]any{"profile": map[string]any{"name": "Sales Team"}},
	}

	tests := []struct {
		name string
		expr string
		want bool
	}{
		{"exact name match", `isMemberOfGroupName("Engineering Team")`, true},
		{"exact name no match", `isMemberOfGroupName("Marketing Team")`, false},
		{"starts with match", `isMemberOfGroupNameStartsWith("Eng")`, true},
		{"starts with no match", `isMemberOfGroupNameStartsWith("Zzz")`, false},
		{"contains match", `isMemberOfGroupNameContains("neer")`, true},
		{"contains no match", `isMemberOfGroupNameContains("zzz")`, false},
		{"regex match", `isMemberOfGroupNameRegex("^Sales.*")`, true},
		{"regex no match", `isMemberOfGroupNameRegex("^Zzz.*")`, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Given
			p := oktaexpr.New(oktaexpr.WithGroupData(groupData))

			// When
			got, err := p.Parse(tc.expr)

			// Then
			if err != nil {
				t.Fatalf("Parse(%q): unexpected error %v", tc.expr, err)
			}
			if got != tc.want {
				t.Errorf("Parse(%q): got %v, want %v", tc.expr, got, tc.want)
			}
		})
	}
}

func TestParse_ArrayLiterals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		expr string
		want any
	}{
		{"contains present", "Arrays.contains({0,1,2}, 0)", true},
		{"contains absent", "Arrays.contains({0,1,2}, 5)", false},
		{"add appends", "Arrays.add({0,1}, 2)", values.Array{0, 1, 2}},
		{"size", "Arrays.size({0,1,2})", 3},
		{"single string element is not exploded into characters", `Arrays.size({"ab"})`, 1},
		{"single string element preserved whole", `Arrays.contains({"ab"}, "ab")`, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Given
			p := oktaexpr.New()

			// When
			got, err := p.Parse(tc.expr)

			// Then
			if err != nil {
				t.Fatalf("Parse(%q): unexpected error %v", tc.expr, err)
			}
			if !values.EqualOperands(got, tc.want) {
				t.Errorf("Parse(%q): got %#v, want %#v", tc.expr, got, tc.want)
			}
		})
	}
}

func TestParse_EmptyArrayLiteralIsAnError(t *testing.T) {
	t.Parallel()

	// Given: the grammar's array rule requires at least one operand inside
	// the braces; {} has no such production and is a genuine parse error,
	// matching the source.
	p := oktaexpr.New()

	// When
	_, err := p.Parse("Arrays.isEmpty({})")

	// Then
	if err == nil {
		t.Errorf("Parse(%q): got nil error, want an error", "Arrays.isEmpty({})")
	}
}

func TestParse_ClassMethodCalls(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		expr string
		want any
	}{
		{"String.stringContains", `String.stringContains("hello", "ell")`, true},
		{"String.toUpperCase", `String.toUpperCase("abc")`, "ABC"},
		{"String.join with multiple args", `String.join(",", "a", "b", "c")`, "a,b,c"},
		{"Iso3166Convert.toAlpha3", `Iso3166Convert.toAlpha3("US")`, "USA"},
		{"Iso3166Convert.toName", `Iso3166Convert.toName("US")`, "United States of America"},
		// Convert is fixed in this port; see convert_test.go for why the
		// Python source's version always raised instead.
		{"Convert.toInt now works", `Convert.toInt("5")`, 5},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Given
			p := oktaexpr.New()

			// When
			got, err := p.Parse(tc.expr)

			// Then
			if err != nil {
				t.Fatalf("Parse(%q): unexpected error %v", tc.expr, err)
			}
			if !values.EqualOperands(got, tc.want) {
				t.Errorf("Parse(%q): got %#v, want %#v", tc.expr, got, tc.want)
			}
		})
	}
}

// TestParse_CommaGreedilyBindsToInnermostOperand verifies a surprising but
// verified-real behavior of the source grammar: a comma inside a nested
// ternary branch is consumed by that branch (building a Tuple) rather than
// by an enclosing call's argument list.
func TestParse_CommaGreedilyBindsToInnermostOperand(t *testing.T) {
	t.Parallel()

	// Given: the ternary's false branch ("b", "c") greedily absorbs the
	// trailing ", \"c\"" that looks like it should belong to String.join's
	// argument list, so String.join only ever receives (",", "a") — matching
	// the verified Python behavior.
	p := oktaexpr.New()

	// When
	got, err := p.Parse(`String.join(",", true ? "a" : "b", "c")`)

	// Then
	if err != nil {
		t.Fatalf("Parse: unexpected error %v", err)
	}
	if got != "a" {
		t.Errorf(`Parse(String.join(",", true ? "a" : "b", "c")): got %#v, want "a"`, got)
	}
}

func TestParse_UnknownClassIsAnError(t *testing.T) {
	t.Parallel()

	p := oktaexpr.New()

	_, err := p.Parse(`NoSuchClass.method("x")`)
	if err == nil {
		t.Errorf("Parse with unknown class: got nil error, want an error")
	}
}

func TestParse_SyntaxErrorsReturnAnError(t *testing.T) {
	t.Parallel()

	// Given: unlike the Python source, which sometimes logs a warning to
	// stderr and silently returns None for a malformed expression, this
	// port always returns a non-nil error.
	tests := []struct {
		name string
		expr string
	}{
		{"empty expression", ""},
		{"incomplete comparison", "1 =="},
		{"unterminated string", `"unterminated`},
		{"bare ampersand", "1 & 1"},
		{"trailing garbage", "1 == 1 2"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Given
			p := oktaexpr.New()

			// When
			_, err := p.Parse(tc.expr)

			// Then
			if err == nil {
				t.Errorf("Parse(%q): got nil error, want an error", tc.expr)
			}
		})
	}
}

// TestParse_StaticTestScenarios converts the scenarios from the Python
// source's tests/static_test.py (an unasserted manual script, not a real
// test) into real assertions.
func TestParse_StaticTestScenarios(t *testing.T) {
	t.Parallel()

	user := map[string]any{
		"groups":    values.Array{"00g1mf03t9hPrfpaO4h7"},
		"location":  "US",
		"phiaccess": "true",
		"booltest":  "true",
	}
	p := oktaexpr.New(
		oktaexpr.WithGroupIDs([]string{"00g1mf03t9hPrfpaO4h7"}),
		oktaexpr.WithUserProfile(user),
	)

	tests := []struct {
		name string
		expr string
		want any
	}{
		{"string class helper", `String.stringContains(user.location, "US")`, true},
		{"combined user attribute comparison", `user.location == "US" AND user.userName == "SU"`, false},
		{"group membership check", `isMemberOfAnyGroup("00g1mf03t9hPrfpaO4h7", "123456")`, true},
		{"array contains", `Arrays.contains({0,1,2}, 0)`, true},
		{"bool-shaped string comparison", `user.booltest == "true"`, true},
		{"mixed group check and/or with ints", `isMemberOfAnyGroup("1234","5678") && 1 || 1`, 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// When
			got, err := p.Parse(tc.expr)

			// Then
			if err != nil {
				t.Fatalf("Parse(%q): unexpected error %v", tc.expr, err)
			}
			if !values.EqualOperands(got, tc.want) {
				t.Errorf("Parse(%q): got %#v, want %#v", tc.expr, got, tc.want)
			}
		})
	}
}

// shoutClass is a custom expressionclasses.Class used to verify that
// WithExpressionClasses lets callers plug in their own implementation of a
// class, mirroring the Python library's `expression_classes` module
// parameter. Note the class NAME must still be one of the lexer's fixed set
// (String, Arrays, Convert, Iso3166Convert, Groups): the "CLASS" token is a
// fixed set of keywords in both the source grammar and this port, so a
// custom registry can override what those names do, but can't introduce an
// entirely new class name — that would require a grammar change, not just
// swapping out an implementation.
type shoutClass struct{}

func (shoutClass) Call(method string, args ...any) (any, error) {
	if method != "shout" {
		return nil, fmt.Errorf("String has no method %q", method)
	}
	s, _ := args[0].(string)
	return strings.ToUpper(s) + "!", nil
}

func TestParse_CustomExpressionClasses(t *testing.T) {
	t.Parallel()

	// Given
	p := oktaexpr.New(oktaexpr.WithExpressionClasses(expressionclasses.Registry{
		"String": shoutClass{},
	}))

	// When
	got, err := p.Parse(`String.shout("hi")`)

	// Then
	if err != nil {
		t.Fatalf("Parse: unexpected error %v", err)
	}
	if got != "HI!" {
		t.Errorf(`Parse(String.shout("hi")): got %#v, want "HI!"`, got)
	}
}

func TestParse_CustomExpressionClasses_DefaultClassesAreReplacedNotMerged(t *testing.T) {
	t.Parallel()

	// Given: WithExpressionClasses replaces the registry outright, so the
	// built-in Arrays class is no longer available even though String was
	// the only class actually being overridden.
	p := oktaexpr.New(oktaexpr.WithExpressionClasses(expressionclasses.Registry{
		"String": shoutClass{},
	}))

	// When
	_, err := p.Parse(`Arrays.size({1,2,3})`)

	// Then
	if err == nil {
		t.Errorf("Parse(Arrays.size(...)) with a replaced registry: got nil error, want an error")
	}
}

func TestParse_ClassCallSyntaxErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		expr string
	}{
		{"missing dot after class name", `String("x")`},
		{"missing method name", `String.("x")`},
		{"missing opening paren", `String.toUpperCase "x")`},
		{"missing closing paren", `String.toUpperCase("x"`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Given
			p := oktaexpr.New()

			// When
			_, err := p.Parse(tc.expr)

			// Then
			if err == nil {
				t.Errorf("Parse(%q): got nil error, want an error", tc.expr)
			}
		})
	}
}

func TestParse_MemberOfGroupNameRegex_InvalidPatternIsAnError(t *testing.T) {
	t.Parallel()

	// Given
	p := oktaexpr.New(oktaexpr.WithGroupData(map[string]any{
		"g1": map[string]any{"profile": map[string]any{"name": "Engineering"}},
	}))

	// When
	_, err := p.Parse(`isMemberOfGroupNameRegex("[")`)

	// Then
	if err == nil {
		t.Errorf(`Parse(isMemberOfGroupNameRegex("[")): got nil error, want an error`)
	}
}

func TestParse_MemberOfSyntaxErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		expr string
	}{
		{"missing opening paren", `isMemberOfGroup "x")`},
		{"missing closing paren", `isMemberOfGroup("x"`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Given
			p := oktaexpr.New()

			// When
			_, err := p.Parse(tc.expr)

			// Then
			if err == nil {
				t.Errorf("Parse(%q): got nil error, want an error", tc.expr)
			}
		})
	}
}
