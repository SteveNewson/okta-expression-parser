# okta-expression-parser

A Go library for evaluating [Okta Expression Language](https://developer.okta.com/docs/reference/okta-expression-language/)
expressions against a user profile and group memberships — a boolean or
value result from a string like `user.department == "Engineering"`.

This is a Go port of the Python library
[mathewmoon/okta-expression-parser](https://github.com/mathewmoon/okta-expression-parser).
Rather than generating a parser from a grammar file (the Python source uses
`sly`, a yacc-style parser generator), this port is a hand-written
recursive-descent parser that evaluates each expression as it parses,
mirroring the semantics of the source's embedded grammar-rule actions.

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

result, err := p.Parse(`user.location == "US" and isMemberOfGroup("00g1")`)
// result == true, err == nil
```

`Parse` returns `any`: a `bool`, `int`, `float64`, `string`, `nil`,
`oktaexpr.Array`, `oktaexpr.Tuple`, or `map[string]any`, depending on the
expression.

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
