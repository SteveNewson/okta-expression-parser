# GitHub Copilot Instructions

These instructions define how GitHub Copilot should assist with this Go project. The goal is to ensure consistent, high-quality code generation aligned with Go idioms, the chosen architecture, and our team's best practices.

## 🧠 Context

- **Project Type**: Go library (no `main` package, no CLI, no UI)
- **Language**: Go
- **Framework / Libraries**: none — standard library only
- **Architecture**: Hand-written recursive-descent lexer/parser/evaluator. `oktaexpr` (root package) exposes `Parser`/`New`/`Option`; `expressionclasses` implements the pluggable `String`/`Arrays`/`Convert`/`Iso3166Convert`/`Groups` classes behind a `Class` interface and `Registry`; `values` holds the shared runtime value model (`Array`, `Tuple`, truthiness, equality, comparison, `TypeName`).

## 🔧 General Guidelines

- Follow idiomatic Go conventions (<https://go.dev/doc/effective_go>).
- Use named functions over long anonymous ones.
- Organize logic into small, composable functions.
- Prefer interfaces for dependencies to enable mocking and testing, and define them at the consumer, not the implementation.
- Use `gofmt` or `goimports` to enforce formatting.
- Avoid unnecessary abstraction; keep things simple and readable.
- Use `context.Context` for request-scoped values and cancellation, if a future API ever needs it — nothing in this library currently does I/O or blocks, so don't introduce `context.Context` params speculatively.

## 📦 Library Design Principles

This is a library, imported by other Go modules (e.g. `okta-rules`, pinned to
a specific commit rather than a tagged release) — not an application with
its own entry point. That changes what "good" looks like:

- **No I/O, no side effects.** Never call `fmt.Println`/`log.Print`/`os.Exit`/`os.Stdout`/`os.Stderr` from library code. A caller embedding this parser in a CLI, a web service, or a batch job should see identical behavior; printing or exiting from inside the library takes that control away from them.
- **Never panic for expected failures.** A malformed expression, a bad argument type, or a lookup miss is an ordinary error, not a panic. Return `error` (or a typed error like `ErrUnexpectedType`) and let the caller decide what to do. Reserve panics for genuine programmer errors that indicate a bug in this package itself.
- **Keep the public API minimal.** Only export what callers actually need (`Parser`, `New`, `Option`, the `With*` constructors, the typed `Parse*` accessors, `Array`/`Tuple`, `ErrUnexpectedType`). Prefer unexported helpers; it's much cheaper to export something later than to remove it once something depends on it.
- **Functional options for configuration**, matching the existing `Option func(*Parser)` pattern — see `WithGroupIDs`, `WithUserProfile`, `WithGroupData`, `WithExpressionClasses` in `parser.go`. Add new configuration the same way rather than growing `New`'s parameter list or adding setter methods.
- **No global/package-level mutable state.** Every `New(...)` call must produce an independent `Parser`; nothing about one caller's configuration (profile data, group IDs, custom classes) should leak into another's. `expressionclasses.Default()` returning fresh `Class` instances each call is the existing example to follow.
- **Every exported identifier gets a doc comment**, starting with its own name (`// Parser evaluates ...`, `// WithGroupIDs sets ...`), per <https://go.dev/blog/godoc>. This is already consistent throughout the package — keep it that way for anything new.
- **Don't add dependencies.** The module has zero non-stdlib dependencies today; that's deliberate for a small, embeddable parser. Reach for the standard library first, and treat adding any external dependency as something to flag and justify explicitly, not something to do incidentally.
- **Exported signatures are a compatibility contract.** Since consumers pin a commit rather than a semver tag, a change to an exported function/method signature or type breaks them the moment they next `go get`. Prefer additive changes (a new `Option`, a new typed accessor) over breaking ones; if a breaking change is unavoidable, call it out clearly (e.g. in the commit message and `README.md`'s "Deviations" section, which already documents intentional behavior changes).

## 📁 File Structure

The repo is intentionally flat — resist adding directory layers that don't earn their keep:

```text
oktaexpr (root package)
  parser.go            — Parser, Option, New, With* configuration
  lexer.go             — tokenizer
  grammar.go           — recursive-descent parser/evaluator
  value.go             — thin re-exports/wrappers around the values package
  typed_parse.go        — ParseBool/ParseString/ParseInt/ParseFloat64/ParseArray, ErrUnexpectedType
expressionclasses/
  registry.go          — Class interface, Registry, Default()
  string.go, arrays.go, convert.go, iso3166convert.go, groups.go
values/
  values.go            — Array, Tuple, Truthy, EqualOperands, CompareOperands, AddOperands, TypeName
```

- New expression classes (if ever needed) belong in `expressionclasses/` as a new file plus a `Class` implementation registered in `Default()`.
- New shared value semantics (equality, ordering, coercion) belong in `values/`, not duplicated in `grammar.go` or an `expressionclasses` file.

## 🧶 Patterns

### ✅ Patterns to Follow

- Return errors, don't panic or log — see "No I/O, no side effects" above.
- Use dependency injection via constructors/functional options (avoid global state).
- When a behavior deliberately differs from the Python source this library ports, or from Okta's own docs, document *why* in the README's "Deviations" section and/or a code comment — see the existing entries for the pattern to follow.

### 🚫 Patterns to Avoid

- Don't use global state unless absolutely required.
- Don't hardcode configuration that should be an `Option`.
- Don't panic or exit in library code; return errors instead.
- Don't add a CLI, a `main` package, or any UI/TUI code to this repo — that belongs in a consuming application, not here.

## 🧪 Testing Guidelines

- Use the standard `testing` package only. Avoid assertion libraries (e.g. `testify`); use plain `if` statements so failure messages stay meaningful and specific.
- Place tests alongside the code they test using Go's `_test.go` convention (e.g. `expressionclasses/string_test.go`).
- Prefer **black-box tests** (`package oktaexpr_test`, `package expressionclasses_test`) for anything exercising the public API — see `parser_test.go` and `expressionclasses/*_test.go`. Use the internal package name (`package oktaexpr`) only when a test genuinely needs an unexported symbol, as `lexer_test.go` does for `tokenize`.
- Include table-driven tests for functions with many input variants.
- When checking behavior against Okta's own public reference docs or the ported Python source, say so directly in the test name/comment (see `reference_doc_test.go` and the README's "Deviations from the Python source" section) — that context matters for future maintainers deciding whether a "wrong-looking" result is a bug or a deliberate, verified deviation.

## 🔁 Iteration & Review

- Review Copilot output before committing.
- Refactor generated code to ensure readability and testability.
- Use comments to give Copilot context for better suggestions.
- Regenerate parts that are unidiomatic or too complex.

## 📚 References

- [Go Style Guide](https://google.github.io/styleguide/go/)
- [Effective Go](https://go.dev/doc/effective_go)
- [Godoc: documenting Go code](https://go.dev/blog/godoc)
- [Okta Expression Language reference](https://developer.okta.com/docs/reference/okta-expression-language/) (the ground truth this library is checked against, alongside the Python source it ports)

# Go Testing Principles & Rules

This document outlines the testing standards for this project, based on the principles of "The Power of Go: Tests" by John Arundel. Use these rules to guide all AI-generated code and manual refactoring.

## General Philosophy
* **Tests are code:** Treat test code with the same care as production code. It should be readable, maintainable, and clean.
* **Clarity over Cleverness:** A test should be so simple that it is obviously correct. Avoid complex logic or abstractions inside test functions.
* **Standard Library First:** Favor the built-in `testing` package. Avoid assertion libraries (like `testify`); use plain `if` statements for comparisons to keep error messages meaningful.

## Test Execution & Parallelism
* **Parallelize by Default:** Every test should start with `t.Parallel()` unless it relies on shared state or global resources that make parallelization impossible.
* **Subtests:** When using `t.Run`, the subtest should also call `t.Parallel()` if the parent test is parallel.

## Table-Driven Tests
* **Preferred Pattern:** Use table-driven tests for functions with multiple edge cases or inputs.
* **Structure:** Define a slice of anonymous structs (usually named `tests` or `cases`).
* **Naming:** Each test case should have a `name` string field to describe the scenario.
* **Execution:** Iterate over the cases and use `t.Run(tc.name, ...)` to execute them.

```go
func TestCalculate(t *testing.T) {
    t.Parallel()
    tests := []struct {
        name  string
        input int
        want  int
    }{
        {"positive numbers", 1, 2},
        {"negative numbers", -1, 0},
    }
    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            t.Parallel()
            got := Calculate(tc.input)
            if got != tc.want {
                t.Errorf("Calculate(%d): got %d, want %d", tc.input, got, tc.want)
            }
        })
    }
}
```

## Test Structure & Style
* **Flat Tests Preference:** Avoid nested subtests (`t.Run`) unless you are explicitly writing parameterized/table-driven tests. Create individual, distinctly named flat test functions instead (e.g., `TestCalculate_PositiveNumbers`).
* **Given / When / Then:** Structure every test using the AAA (Arrange, Act, Assert) or "Given / When / Then" pattern using code comments.
  * `// Given`: Setup state, instantiate dependencies, define inputs.
  * `// When`: The action being tested. Constrain this section to **a single method call**.
  * `// Then`: Assertions verifying the behavior.

```go
func TestCalculate_PositiveNumbers(t *testing.T) {
    t.Parallel()
    
    // Given
    input := 1
    want := 2
    
    // When
    got := Calculate(input)
    
    // Then
    if got != want {
        t.Errorf("Calculate(%d): got %d, want %d", input, got, want)
    }
}
```

## Error Messages
* **Informative Failures:** Error messages should follow the "got/want" pattern.
* **Context:** Include the input that caused the failure in the error message so the developer doesn't have to debug to find the culprit.
* **Format:** `t.Errorf("Subject(%v): got %v, want %v", input, got, want)`

## Minimalism in Mocking
* **Avoid Mocking Frameworks:** Don't use code generators for mocks unless the interface is massive.
* **Hand-written Fakes:** Use small, hand-written "fake" implementations or functional options for dependencies.
* **Interfaces at the Consumer:** Define interfaces where they are *used*, not where the implementation is defined, to keep them as small as possible.

## Package Layout
* **Black-box Testing:** Use the `_test` package suffix (e.g., `package oktaexpr_test`) for tests exercising the public API. This ensures you are testing the code as a user of the package.
* **Internal Tests:** Use the same package name (e.g., `package oktaexpr`) only when you need to test unexported functions or variables, such as the lexer's `tokenize`.

## Test Helpers
* **t.Helper():** When creating reusable helper functions (e.g., a setup function or a custom check), call `t.Helper()` at the start. This ensures that failure line numbers point to the caller of the helper, not the helper itself.
