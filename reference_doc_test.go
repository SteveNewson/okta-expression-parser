package oktaexpr_test

import (
	"testing"

	oktaexpr "github.com/stevenewson/okta-expression-parser"
	"github.com/stevenewson/okta-expression-parser/values"
)

// The tests in this file are checked directly against the examples in
// Okta's own reference documentation:
// https://developer.okta.com/docs/reference/okta-expression-language/
//
// This library implements the subset of Okta Expression Language used to
// evaluate group rule conditions (String, Arrays, Convert, Iso3166Convert,
// the isMemberOf* group functions, and Groups.getFilteredGroups), plus
// general-purpose constants, comparisons, boolean logic, the ternary
// operator, and the "+" operator. The reference page documents a much
// larger surface used elsewhere in Okta (profile mappings, OAuth/OIDC
// claims, session properties, app/org properties), which this library does
// not implement. The following are deliberately NOT covered here, since
// exercising them would just document a parse/lookup failure rather than
// verify a real behavior:
//
//   - Time, Manager/Assistant, and Directory/Workday functions, and
//     user.getGroups, hasDirectoryUser/hasWorkdayUser and friends: none of
//     these functions exist in this library.
//   - session.*, app.*/appuser.*, org.*, idpuser.*, and OAuth/OIDC custom
//     claim expressions (access.scope, etc.): these namespaces are
//     meaningful only outside group rules; a bare, non-"user."-prefixed
//     name always resolves to nil here (a preserved quirk of the source
//     grammar — see the README).
//   - user.getInternalProperty(...) and user.getLinkedObject(...): method
//     calls on "user.<path>" chains aren't supported; only plain attribute
//     access is.
//   - Array index syntax ({1,2,3}[0], user.arrayProperty[0]): the lexer has
//     no "[" token.
//   - The Elvis operator (?:) and the deprecated "matches" regex operator:
//     neither is implemented.
//   - The deprecated, un-namespaced legacy function aliases
//     (toUpperCase(...), substringBefore(...), etc.) and the bare
//     getFilteredGroups(...)/Groups.startsWith/endsWith/contains
//     group-claims functions: bare (non-"Class.method") function calls
//     aren't supported at all — see TestString_Call and friends in
//     expressionclasses for the namespaced equivalents this library does
//     implement.
//   - CSV-string-to-array coercion ("Arrays.flatten('10, 20, 30, 40')" and
//     the general "You can use CSV as an input parameter for all Arrays*
//     functions" note): only literal {...} arrays (or already-Array-typed
//     profile values) are accepted; a bare string argument is rejected.
//   - String.append: documented as plain concatenation, but this port
//     deliberately reproduces the real (verified) Python source's actual
//     "$"-separated behavior instead — see TestString_Call's "append" case
//     in expressionclasses/string_test.go, and the profile-mapping samples
//     in the "Conditional samples" section, which rely on bare (non-"user.")
//     source-attribute names and so aren't resolvable through this
//     library's WithUserProfile.
//
// Three real bugs were found and fixed while writing these tests:
// String.len was entirely unimplemented, Arrays.size(NULL)/isEmpty(NULL)
// errored instead of returning 0/true, and Convert.toInt truncated a float
// instead of rounding to the nearest integer. Two documented but previously
// entirely unimplemented features were also added: the "+" operator and
// decimal float literals (e.g. 3.141).

func TestReferenceDoc_StringFunctions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		expr string
		want any
	}{
		{"String.join with separator", `String.join(",", "This", "is", "a", "test")`, "This,is,a,test"},
		{"String.join with empty separator", `String.join("", "This", "is", "a", "test")`, "Thisisatest"},
		{"String.len", `String.len("This")`, 4},
		{"String.removeSpaces", `String.removeSpaces("This is a test")`, "Thisisatest"},
		{"String.replace", `String.replace("This is a test", "is", "at")`, "That at a test"},
		{"String.replaceFirst", `String.replaceFirst("This is a test", "is", "at")`, "That is a test"},
		{"String.startsWith", `String.startsWith("Kiss", "K")`, true},
		{"String.stringContains true", `String.stringContains("This is a test", "test")`, true},
		{"String.stringContains false", `String.stringContains("This is a test", "doesn'tExist")`, false},
		{"String.stringSwitch no key matches", `String.stringSwitch("This is a test", "default", "key1", "value1")`, "default"},
		{"String.stringSwitch key matches", `String.stringSwitch("This is a test", "default", "test", "value1")`, "value1"},
		{"String.stringSwitch first match wins", `String.stringSwitch("First match wins", "default", "absent", "value1", "wins", "value2", "match", "value3")`, "value2"},
		{"String.stringSwitch matches by substring", `String.stringSwitch("Substrings count", "default", "ring", "value1")`, "value1"},
		{"String.substring", `String.substring("This is a test", 2, 9)`, "is is a"},
		{"String.substringAfter", `String.substringAfter("abc@okta.com", "@")`, "okta.com"},
		{"String.substringBefore", `String.substringBefore("abc@okta.com", "@")`, "abc"},
		{"String.toUpperCase", `String.toUpperCase("This")`, "THIS"},
		{"String.toLowerCase", `String.toLowerCase("ThiS")`, "this"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p := oktaexpr.New()

			got, err := p.Parse(tc.expr)
			if err != nil {
				t.Fatalf("Parse(%q): unexpected error %v", tc.expr, err)
			}
			if got != tc.want {
				t.Errorf("Parse(%q): got %#v, want %#v", tc.expr, got, tc.want)
			}
		})
	}
}

func TestReferenceDoc_ArrayFunctions(t *testing.T) {
	t.Parallel()

	t.Run("Arrays.add", func(t *testing.T) {
		t.Parallel()
		p := oktaexpr.New(oktaexpr.WithUserProfile(map[string]any{"arrayAttribute": values.Array{10, 20, 30}}))
		got, err := p.Parse(`Arrays.add(user.arrayAttribute, 40)`)
		if err != nil {
			t.Fatalf("unexpected error %v", err)
		}
		want := values.Array{10, 20, 30, 40}
		if !arrayEqual(got, want) {
			t.Errorf("got %#v, want %#v", got, want)
		}
	})

	t.Run("Arrays.remove", func(t *testing.T) {
		t.Parallel()
		p := oktaexpr.New(oktaexpr.WithUserProfile(map[string]any{"arrayAttribute": values.Array{10, 20, 30}}))
		got, err := p.Parse(`Arrays.remove(user.arrayAttribute, 20)`)
		if err != nil {
			t.Fatalf("unexpected error %v", err)
		}
		want := values.Array{10, 30}
		if !arrayEqual(got, want) {
			t.Errorf("got %#v, want %#v", got, want)
		}
	})

	t.Run("Arrays.clear", func(t *testing.T) {
		t.Parallel()
		p := oktaexpr.New(oktaexpr.WithUserProfile(map[string]any{"arrayAttribute": values.Array{10, 20, 30}}))
		got, err := p.Parse(`Arrays.clear(user.arrayAttribute)`)
		if err != nil {
			t.Fatalf("unexpected error %v", err)
		}
		if !arrayEqual(got, values.Array{}) {
			t.Errorf("got %#v, want {}", got)
		}
	})

	simple := []struct {
		name string
		expr string
		want any
	}{
		{"Arrays.get", `Arrays.get({0, 1, 2}, 0)`, 0},
		{"Arrays.contains true", `Arrays.contains({10, 20, 30}, 10)`, true},
		{"Arrays.contains false", `Arrays.contains({10, 20, 30}, 50)`, false},
		{"Arrays.size", `Arrays.size({10, 20, 30})`, 3},
		{"Arrays.size of NULL", `Arrays.size(NULL)`, 0},
		{"Arrays.isEmpty false", `Arrays.isEmpty({10, 20})`, false},
		{"Arrays.isEmpty of NULL", `Arrays.isEmpty(NULL)`, true},
		{"Arrays.toCsvString", `Arrays.toCsvString({"This", "is", " a ", "test"})`, "This,is, a ,test"},
	}
	for _, tc := range simple {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := oktaexpr.New()
			got, err := p.Parse(tc.expr)
			if err != nil {
				t.Fatalf("Parse(%q): unexpected error %v", tc.expr, err)
			}
			if got != tc.want {
				t.Errorf("Parse(%q): got %#v, want %#v", tc.expr, got, tc.want)
			}
		})
	}

	t.Run("Arrays.flatten multiple args and nested arrays", func(t *testing.T) {
		t.Parallel()
		p := oktaexpr.New()
		got, err := p.Parse(`Arrays.flatten(10, {20, 30}, 40)`)
		if err != nil {
			t.Fatalf("unexpected error %v", err)
		}
		want := values.Array{10, 20, 30, 40}
		if !arrayEqual(got, want) {
			t.Errorf("got %#v, want %#v", got, want)
		}
	})
}

func arrayEqual(got any, want values.Array) bool {
	arr, ok := got.(values.Array)
	if !ok || len(arr) != len(want) {
		return false
	}
	for i := range arr {
		if arr[i] != want[i] {
			return false
		}
	}
	return true
}

func TestReferenceDoc_ConvertFunctions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		profile map[string]any
		expr    string
		want    any
	}{
		{"Convert.toInt from string", map[string]any{"val": "1234"}, `Convert.toInt(user.val)`, 1234},
		{"Convert.toInt rounds a float down", map[string]any{"val": 123.4}, `Convert.toInt(user.val)`, 123},
		{"Convert.toInt rounds a float up", map[string]any{"val": 123.6}, `Convert.toInt(user.val)`, 124},
		{"Convert.toNum from string", map[string]any{"val": "3.141"}, `Convert.toNum(user.val)`, 3.141},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p := oktaexpr.New(oktaexpr.WithUserProfile(tc.profile))

			got, err := p.Parse(tc.expr)
			if err != nil {
				t.Fatalf("Parse(%q): unexpected error %v", tc.expr, err)
			}
			if got != tc.want {
				t.Errorf("Parse(%q): got %#v, want %#v", tc.expr, got, tc.want)
			}
		})
	}
}

func TestReferenceDoc_Iso3166ConvertFunctions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		expr string
		want any
	}{
		{"toAlpha2", `Iso3166Convert.toAlpha2("IND")`, "IN"},
		{"toAlpha3", `Iso3166Convert.toAlpha3("840")`, "USA"},
		{"toNumeric", `Iso3166Convert.toNumeric("USA")`, "840"},
		{"toName", `Iso3166Convert.toName("IN")`, "India"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p := oktaexpr.New()

			got, err := p.Parse(tc.expr)
			if err != nil {
				t.Fatalf("Parse(%q): unexpected error %v", tc.expr, err)
			}
			if got != tc.want {
				t.Errorf("Parse(%q): got %#v, want %#v", tc.expr, got, tc.want)
			}
		})
	}
}

// TestReferenceDoc_GroupFunctions covers Groups.getFilteredGroups, using the
// exact group ID from the doc's example. The isMemberOf* group functions
// shown in the same table are already covered in depth by
// TestParse_MemberOfGroup, TestParse_MemberOfAnyGroup, and
// TestParse_MemberOfGroupNameFamily in parser_test.go.
func TestReferenceDoc_GroupFunctions(t *testing.T) {
	t.Parallel()

	groupData := map[string]any{
		"00gml2xHE3RYRx7cM0g3": map[string]any{"name": "Sales Team"},
	}
	p := oktaexpr.New(oktaexpr.WithGroupData(groupData))

	got, err := p.Parse(`Groups.getFilteredGroups({"00gml2xHE3RYRx7cM0g3"}, "group.name", 40)`)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	want := values.Array{"Sales Team"}
	if !arrayEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

// TestReferenceDoc_ConstantsAndOperators covers the "Constants and
// operators" table, except array-index syntax ({1,2,3}[0]) and the Elvis
// operator (?:), neither of which this library implements.
func TestReferenceDoc_ConstantsAndOperators(t *testing.T) {
	t.Parallel()

	literals := []struct {
		name string
		expr string
		want any
	}{
		{"string constant", `'Hello world'`, "Hello world"},
		{"integer constant", `1234`, 1234},
		{"number constant", `3.141`, 3.141},
		{"boolean constant", `true`, true},
	}
	for _, tc := range literals {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := oktaexpr.New()
			got, err := p.Parse(tc.expr)
			if err != nil {
				t.Fatalf("Parse(%q): unexpected error %v", tc.expr, err)
			}
			if got != tc.want {
				t.Errorf("Parse(%q): got %#v, want %#v", tc.expr, got, tc.want)
			}
		})
	}

	profile := map[string]any{"firstName": "Winston", "lastName": "Churchill", "groupCode": 123}

	concatAndTernary := []struct {
		name string
		expr string
		want any
	}{
		{"concatenate two strings", `user.firstName + user.lastName`, "WinstonChurchill"},
		{"concatenate two strings with space", `user.firstName + " " + user.lastName`, "Winston Churchill"},
		{"ternary true branch", `user.groupCode == 123 ? 'Sales' : 'Other'`, "Sales"},
	}
	for _, tc := range concatAndTernary {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := oktaexpr.New(oktaexpr.WithUserProfile(profile))
			got, err := p.Parse(tc.expr)
			if err != nil {
				t.Fatalf("Parse(%q): unexpected error %v", tc.expr, err)
			}
			if got != tc.want {
				t.Errorf("Parse(%q): got %#v, want %#v", tc.expr, got, tc.want)
			}
		})
	}

	t.Run("ternary false branch", func(t *testing.T) {
		t.Parallel()
		p := oktaexpr.New(oktaexpr.WithUserProfile(map[string]any{"groupCode": 456}))
		got, err := p.Parse(`user.groupCode == 123 ? 'Sales' : 'Other'`)
		if err != nil {
			t.Fatalf("unexpected error %v", err)
		}
		if got != "Other" {
			t.Errorf("got %#v, want %#v", got, "Other")
		}
	})
}

// TestReferenceDoc_GroupRuleSamples covers the "Group rules samples" table
// under "Conditional samples", except the two rows using the deprecated
// "matches" regex operator, which isn't implemented.
func TestReferenceDoc_GroupRuleSamples(t *testing.T) {
	t.Parallel()

	profile := map[string]any{
		"department":     "Sales-West",
		"city":           "San Francisco",
		"salary":         1500000,
		"isContractor":   false,
		"email":          "alice@example.com",
		"hasBadge":       true,
		"favoriteColors": values.Array{"blue", "green"},
	}

	tests := []struct {
		name string
		expr string
		want bool
	}{
		{`IF String.stringContains(user.department, "Sales")`, `String.stringContains(user.department, "Sales")`, true},
		{`IF user.city == "San Francisco"`, `user.city == "San Francisco"`, true},
		{`IF user.salary >= 1000000`, `user.salary >= 1000000`, true},
		{`IF !user.isContractor`, `!user.isContractor`, true},
		{`IF user.salary > 1000000 AND !user.isContractor`, `user.salary > 1000000 AND !user.isContractor`, true},
		// From "Expressions in group rules": allowed forms of String/Arrays/user
		// expressions in a group rule condition.
		{"user.hasBadge", `user.hasBadge`, true},
		{`String.stringContains(user.email, "@example.com")`, `String.stringContains(user.email, "@example.com")`, true},
		{`Arrays.contains(user.favoriteColors, "blue")`, `Arrays.contains(user.favoriteColors, "blue")`, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p := oktaexpr.New(oktaexpr.WithUserProfile(profile))

			got, err := p.Parse(tc.expr)
			if err != nil {
				t.Fatalf("Parse(%q): unexpected error %v", tc.expr, err)
			}
			if got != tc.want {
				t.Errorf("Parse(%q): got %#v, want %v", tc.expr, got, tc.want)
			}
		})
	}
}

// TestReferenceDoc_NullAndBlankAttributes covers "Check for null and blank
// attributes".
func TestReferenceDoc_NullAndBlankAttributes(t *testing.T) {
	t.Parallel()

	t.Run("never populated attribute is null", func(t *testing.T) {
		t.Parallel()
		p := oktaexpr.New(oktaexpr.WithUserProfile(map[string]any{}))
		got, err := p.Parse(`user.employeeNumber == null`)
		if err != nil {
			t.Fatalf("unexpected error %v", err)
		}
		if got != true {
			t.Errorf("got %#v, want true", got)
		}
	})

	t.Run("populated then cleared attribute is an empty string, not null", func(t *testing.T) {
		t.Parallel()
		p := oktaexpr.New(oktaexpr.WithUserProfile(map[string]any{"employeeNumber": ""}))

		isNull, err := p.Parse(`user.employeeNumber == null`)
		if err != nil {
			t.Fatalf("unexpected error %v", err)
		}
		if isNull != false {
			t.Errorf(`user.employeeNumber == null: got %#v, want false`, isNull)
		}

		isEmpty, err := p.Parse(`user.employeeNumber == ""`)
		if err != nil {
			t.Fatalf("unexpected error %v", err)
		}
		if isEmpty != true {
			t.Errorf(`user.employeeNumber == "": got %#v, want true`, isEmpty)
		}
	})

	fallback := `user.employeeNumber != "" AND user.employeeNumber != null ? user.employeeNumber : user.nonEmployeeNumber`

	t.Run("fallback expression uses employeeNumber when set", func(t *testing.T) {
		t.Parallel()
		p := oktaexpr.New(oktaexpr.WithUserProfile(map[string]any{
			"employeeNumber":    "E-7",
			"nonEmployeeNumber": "N-42",
		}))
		got, err := p.Parse(fallback)
		if err != nil {
			t.Fatalf("unexpected error %v", err)
		}
		if got != "E-7" {
			t.Errorf("got %#v, want %#v", got, "E-7")
		}
	})

	t.Run("fallback expression uses nonEmployeeNumber when employeeNumber is blank", func(t *testing.T) {
		t.Parallel()
		p := oktaexpr.New(oktaexpr.WithUserProfile(map[string]any{
			"employeeNumber":    "",
			"nonEmployeeNumber": "N-42",
		}))
		got, err := p.Parse(fallback)
		if err != nil {
			t.Fatalf("unexpected error %v", err)
		}
		if got != "N-42" {
			t.Errorf("got %#v, want %#v", got, "N-42")
		}
	})
}

// TestReferenceDoc_PopularExpressions covers the subset of the "Popular
// expressions" table that doesn't rely on deprecated, un-namespaced
// function aliases (substring, toLowerCase, ...) or Workday/Active
// Directory functions, none of which this library implements.
func TestReferenceDoc_PopularExpressions(t *testing.T) {
	t.Parallel()

	profile := map[string]any{
		"firstName": "Winston",
		"lastName":  "Churchill",
		"email":     "winston.churchill@gmail.com",
		"login":     "winston.churchill@gmail.com",
	}

	tests := []struct {
		name string
		expr string
		want any
	}{
		{"Firstname", `user.firstName`, "Winston"},
		{"Firstname + Lastname", `user.firstName + user.lastName`, "WinstonChurchill"},
		{"Firstname + Lastname with separator", `user.firstName + "." + user.lastName`, "Winston.Churchill"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p := oktaexpr.New(oktaexpr.WithUserProfile(profile))

			got, err := p.Parse(tc.expr)
			if err != nil {
				t.Fatalf("Parse(%q): unexpected error %v", tc.expr, err)
			}
			if got != tc.want {
				t.Errorf("Parse(%q): got %#v, want %#v", tc.expr, got, tc.want)
			}
		})
	}
}
