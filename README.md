# okta-expression-parser

A Go library for evaluating [Okta Expression Language](https://developer.okta.com/docs/reference/okta-expression-language/)
expressions against a user profile and group memberships — a boolean or
value result from a string like `user.department == "Engineering"`.

This is a Go port of the Python library
[mathewmoon/okta-expression-parser](https://github.com/mathewmoon/okta-expression-parser).
Parsing builds a real AST (`ast.go`) via a hand-written recursive-descent
parser (`astparse.go`); evaluating that AST (`asteval.go`) mirrors the
semantics of the source's embedded grammar-rule actions.

## Usage

```go
import oktaexpr "github.com/stevenewson/okta-expression-parser"

p := oktaexpr.New(
	oktaexpr.WithGroupIDs([]string{"00g1"}),
	oktaexpr.WithUserProfile(map[string]any{
		"location": "US",
		"department": map[string]any{"name": "Engineering"},
	}),
)

node, err := p.Parse(`user.location == "US" and isMemberOfGroup("00g1")`)
result, err := p.Eval(node)
// result == true, err == nil
```

`Parse` builds an `oktaexpr.Node` — a real AST, not an evaluated result —
purely from the expression text; it doesn't need (or use) any of `p`'s
configured profile/groups/classes. `Eval` evaluates a `Node` against `p`'s
configuration. Splitting the two lets a caller inspect or rewrite a parsed
expression (every `Node` type has exported fields, and isn't a sealed
interface, so new nodes can be constructed directly too) before evaluating
it, or evaluate the same parsed `Node` against several different `Parser`
configurations without re-parsing.

`Eval` returns `any`: a `bool`, `int`, `float64`, `string`, `nil`,
`oktaexpr.Array`, `oktaexpr.Tuple`, or `map[string]any`, depending on the
expression. Most callers know ahead of time which type they expect — e.g.
an Okta group rule is always a boolean — so `Parser` also has typed
accessors that do the type assertion for you and return `ErrUnexpectedType`
(checkable with `errors.Is`) if the expression evaluated fine but produced a
different type. `EvalBool`, `EvalString`, `EvalInt`, `EvalFloat64`, and
`EvalArray` all take a `Node`; `ParseBool`, `ParseString`, `ParseInt`,
`ParseFloat64`, and `ParseArray` are convenience one-liners over
`Parse`+the matching `EvalX`, for the common case of evaluating an
expression once with no need for the intermediate `Node`:

```go
member, err := p.ParseBool(`isMemberOfGroup("00g1")`)
if errors.Is(err, oktaexpr.ErrUnexpectedType) {
	// the expression parsed and evaluated, but didn't produce a bool
}
```

None of the typed accessors coerce between numeric types — `ParseInt`/
`EvalInt` on an expression that evaluates to a `float64` (e.g.
`Convert.toNum(...)`) return `ErrUnexpectedType` rather than truncating,
matching the language's own type-strictness elsewhere (see the note on
relational operators below).

## The AST

`Parse` returns one of: `Literal`, `PathExpr` (a `user.a.b.c` chain, or a
bare name — see its doc comment for a preserved quirk), `MemberOfExpr` (the
`isMemberOfGroup` family), `ClassCall` (`Class.method(args)`), `ArrayLit`,
`CommaList` (a comma-joined operand group), `Ternary`, `Comparison`,
`Additive` (`+`), `Logical` (a whole `AND`/`OR` chain), or `Not`. Every type
has exported fields and a `String()` method producing a canonical
re-serialization (not necessarily byte-identical to whatever text
originally parsed to it — parenthesization is re-derived from operator
precedence, not preserved verbatim). `Format(node)` produces multi-line,
indented output instead, for reading a large generated expression.

Building the AST is a pure function of the expression text: it doesn't
consult `WithUserProfile`/`WithGroupIDs`/`WithGroupData`/
`WithExpressionClasses` at all, only `Eval` does — so `Parse` never needs a
specific `Parser`'s configuration, only `Eval` does.

## Deviations from the Python source

The Python source has no real test suite — `tests/static_test.py` is an
unasserted manual script, and `tests/request.py` exercises an unrelated HTTP
server. This port's test suite (and the ground truth it's checked against)
was built by running many expressions against the real Python library and
recording its actual behavior. That process surfaced several bugs, one of
which this port deliberately does not reproduce, and a couple of judgment
calls worth calling out explicitly:

- **Lexer keyword word-boundaries (fixed).** The Python lexer matches
  keywords like `and`, `or`, `not`, `ne`, `eq`, `true`, `null` via regex
  alternation with no word-boundary anchor, so an identifier that merely
  *starts with* a keyword is misread as that keyword followed by a truncated
  name — `user.nested` genuinely fails to parse in the Python library
  because `nested` starts with `ne`. This port scans each identifier
  greedily first and only then checks for an exact keyword match, which
  fixes this while changing nothing else.

- **`<=` is dead code (fixed).** The Python lexer checks token patterns in
  the order `GTE, GT, LT, LTE`. Because `LT`'s pattern (`<`) matches before
  `LTE`'s pattern (`<=`) is ever tried, every `<=` tokenizes as `<` followed
  by a stray `=`, which fails to parse. `<=` is effectively unreachable in
  the Python library. This port checks for the two-character operators
  before falling back to the single-character ones.

- **`Convert.toInt`/`Convert.toNum` are fixed.** In the Python source, both
  are decorated `@classmethod` but their function signatures omit the
  implicit `cls` parameter, so every call raises `TypeError: takes 1
  positional argument but 2 were given`. The class is entirely non-functional
  in the source. This port implements it as documented.

- **A single-string array literal is no longer exploded into characters
  (fixed).** `{"ab"}` in the Python source produces `['a', 'b']` — a
  two-element array of characters — because its array-literal reduction
  checks `isinstance(x, Sequence)`, which strings also satisfy. This port's
  array literal only spreads a comma-joined group of operands; a single
  string operand becomes a one-element array containing the whole string.

- **`user.<path>` supports arbitrary depth (fixed, deliberately).** The
  Python grammar only has one rule for a dotted path access
  (`path "." NAME`), so `user.location` works but `user.outer.inner` fails
  to parse — the real Okta Expression Language supports arbitrary nesting
  like `user.profile.department.name`. This port resolves chains of any
  depth. Resolving through a value that isn't a map (or a missing key)
  yields `nil` at any depth, rather than the Python source's one-level-only
  quirk of returning the last resolved value unchanged when a `.` access
  hits a non-dict.

- **A bare `NAME` (not part of a `user.` chain) always resolves to `nil`
  (preserved, not fixed).** This is a genuine quirk of the Python grammar's
  `path: NAME` rule, which looks the name up in an always-empty dict. Only
  `user` and `user.<path>` ever resolve to a real profile value; this is
  preserved for fidelity since "fixing" it would mean guessing at an
  unspecified intended behavior.

- **Ternary branches must be "operand"-typed, not "condition"-typed
  (preserved).** The grammar defines nearly parallel rules for two internal
  types, informally "operand" (a value) and "condition" (a boolean-ish
  result of a comparison, `and`/`or`, `not`, or an `isMemberOf*` builtin).
  A ternary's branches, and both sides of a comparison, must be
  operand-typed. This means `1 == 1 ? 1 : 2 == 3` and
  `true ? (1==1) : 2` are genuine parse errors — verified against the
  Python source, which rejects them too — even though every subexpression
  involved evaluates to a plain bool. This port replicates that
  restriction rather than loosening it.

- **A comma inside a nested ternary branch binds to the innermost operand
  context (preserved).** Because the comma operator and the ternary
  operator both exist at every "operand" position in the grammar, a
  trailing comma after a ternary embedded in a function call is consumed by
  the ternary's branch, not the enclosing call's argument list — e.g. in
  `f(true ? "a" : "b", "c")`, `"c"` becomes part of the false branch's
  value, not a third argument to `f`. Verified against the Python source,
  which does the same thing. See `TestParse_CommaGreedilyBindsToInnermostOperand`.

- **AND/OR/ternary never short-circuit (preserved).** The Python parser is
  bottom-up (yacc-generated): every subexpression is fully evaluated as it's
  reduced, before an enclosing AND/OR/ternary rule's action ever runs. So
  `false and (1 / 0)`-shaped expressions (using this library's equivalent of
  an error-producing operand) still raise, even though the `false` alone
  determines the AND's result. This port's recursive-descent evaluator
  preserves that by always evaluating both sides of AND/OR and both ternary
  branches.

- **`and`/`or` return the actual operand, not a coerced bool (preserved).**
  Mirroring Python's `and`/`or`, `1 and 2` evaluates to `2`, and
  `0 or 2` evaluates to `2` — not `true`/`false`. `not` always yields a
  strict bool.

- **Errors are never silently swallowed (changed, intentionally).** The
  Python parser is built on `sly`'s default error handling: a malformed
  expression logs a warning to stderr and `parse()` returns `None`,
  indistinguishable from a legitimate null result. `Parser.Parse` always
  returns a non-nil `error` for anything that fails to parse or evaluate.

- **`Groups.getFilteredGroups` and the `isMemberOfGroupName*` builtins use
  different group-data shapes (preserved, not reconciled).**
  `getFilteredGroups` looks a key up directly on each group's data
  (`groupData[id][key]`), while `isMemberOfGroupName`,
  `isMemberOfGroupNameStartsWith`, `isMemberOfGroupNameContains`, and
  `isMemberOfGroupNameRegex` expect a nested `profile.name`. That
  inconsistency exists in the Python source too; `WithGroupData` sets both,
  so callers should shape their group data for whichever feature(s) they use.

- **Bugs found by testing against Okta's own public reference docs (fixed).**
  `reference_doc_test.go` checks this port's behavior against the worked
  examples in
  [Okta's Expression Language reference](https://developer.okta.com/docs/reference/okta-expression-language/)
  directly, rather than against the (partially broken) Python source. That
  surfaced four real bugs, now fixed: `String.len` was entirely
  unimplemented; `Arrays.size(NULL)` and `Arrays.isEmpty(NULL)` errored
  instead of returning `0`/`true`; `Convert.toInt` truncated a float instead
  of rounding to the nearest integer (`123.6` became `123`, not the
  documented `124`); and `String.substringAfter` included the delimiter
  itself in its result (`substringAfter("abc@okta.com", "@")` returned
  `"@okta.com"`, not the documented `"okta.com"`).

- **The `+` operator and decimal float literals (added).** Neither existed
  at all: the lexer had no token for `+`, and a `.` after a digit was always
  read as the start of a path-chain access, so `3.141` failed to tokenize.
  Both are now supported: `+` does string concatenation for two strings and
  arithmetic addition for two ints or two floats (mismatched types are a
  parse-time error, matching the strictness of the relational operators),
  and float literals like `3.141` parse as `float64`.

- **`String.stringContains` only worked when both arguments were strings
  (fixed).** The Python source's `stringContains(container, val)` is just
  `val in container` wrapped in a try/except that turns any `TypeError` into
  `False`. Since Python's `in` is polymorphic over strings *and*
  lists/sets, real production rules use
  `String.stringContains({"a", "b"}, user.department)` as an allow/deny-list
  membership check — relying on the exact same `in` behavior other rules use
  for substring checks. This port originally required both arguments to be
  Go strings and silently returned `false` for anything else (matching only
  the "non-string second argument" case, e.g. `stringContains("hello", 5)`,
  which really does throw in Python and so is genuinely `false`). That made
  the array-container idiom always evaluate to `false` — inverting the
  intent of any rule that negated it, e.g.
  `!String.stringContains({"a", "b"}, user.department)` always evaluating
  `true` regardless of `user.department`. `stringContains` now also accepts
  an array first argument and does a membership check against it.

## Strict property access

By default, accessing a `user.<name>` (or any other `.`-chained) property
that doesn't exist in the underlying map resolves to `nil`, matching the
Python source. That's indistinguishable from a genuinely blank/zero value,
so a rule built against a typo'd or never-exported attribute name silently
evaluates to a wrong-but-not-erroring result instead of surfacing the
mistake.

`WithStrict(true)` opts into failing evaluation instead, but only when the
key is truly absent from the map — a key that's present with a blank string,
zero, `false`, or `null` is not an error, since the point is to catch
missing *attributes*, not "falsy" values:

```go
p := oktaexpr.New(
	oktaexpr.WithStrict(true),
	oktaexpr.WithUserProfile(map[string]any{"department": "Engineering"}),
)

_, err := p.ParseBool(`user.managerEmail == "x@example.com"`)
// err != nil: "managerEmail" isn't a key in the profile map at all

_, err = p.ParseBool(`user.department == "Engineering"`)
// err == nil: the key exists, even though other keys are missing
```

Off by default so existing callers are unaffected.

## Coverage against Okta's public reference docs

This library implements the subset of Okta Expression Language needed to
evaluate group rule conditions: `String`, `Arrays`, `Convert`,
`Iso3166Convert`, the `isMemberOf*` group functions, and
`Groups.getFilteredGroups`, plus constants, comparisons, boolean logic, the
ternary operator, and `+`. `reference_doc_test.go` checks every applicable
example from the reference page directly against this library.

The reference page also documents a much larger surface used elsewhere in
Okta — profile mappings, OAuth/OIDC custom claims, session properties,
Time/Manager/Assistant/Directory/Workday functions, `user.getGroups`, array
index syntax (`{1,2,3}[0]`), the Elvis operator (`?:`), the deprecated
`matches` regex operator, deprecated un-namespaced function aliases
(`toUpperCase(...)`, `substringBefore(...)`, etc.), and CSV-string-to-array
coercion for `Arrays*` functions. None of that is implemented; see the
comment at the top of `reference_doc_test.go` for the full list and why each
is out of scope for a group-rule-evaluation library.

## Expression classes

`Arrays`, `String`, `Convert`, `Iso3166Convert`, and `Groups` are
implemented in the `expressionclasses` package behind a `Class` interface
and a `Registry` (a `map[string]Class`). Pass `WithExpressionClasses` to
`New` to override the default registry — for example, to swap in a
different implementation of `String`, or to change what `Groups.GroupData`
is initialized with.

Note the set of usable class *names* is fixed by the grammar (`String`,
`Arrays`, `Convert`, `Iso3166Convert`, `Groups`) — same as the Python
source. A custom registry can override what those five names do; it can't
introduce a sixth class name, since the lexer only recognizes those five as
`CLASS` tokens.
