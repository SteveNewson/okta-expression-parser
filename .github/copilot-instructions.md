# GitHub Copilot Instructions

These instructions define how GitHub Copilot should assist with this Go project. The goal is to ensure consistent, high-quality code generation aligned with Go idioms, the chosen architecture, and our team's best practices.

## 🧠 Context

- **Project Type**: CLI Tool
- **Language**: Go
- **Framework / Libraries**: cobra, testify, charmbracelet/bubbles
- **Architecture**: Modular MVU (Model-View-Update) + Command Pattern

## 🔧 General Guidelines

- Follow idiomatic Go conventions (<https://go.dev/doc/effective_go>).
- Use named functions over long anonymous ones.
- Organize logic into small, composable functions.
- Prefer interfaces for dependencies to enable mocking and testing.
- Use `gofmt` or `goimports` to enforce formatting.
- Avoid unnecessary abstraction; keep things simple and readable.
- Use `context.Context` for request-scoped values and cancellation.

## 👾 TUI Guidelines

- **Component Structure:**
  - Each distinct UI element or view should generally be implemented as its own `bubble`.
  - Follow the standard `bubbles` pattern:
    - `Model`: Struct containing the component's state.
    - `Init()`: Returns the initial command (often `nil`).
    - `Update(msg tea.Msg)`: Handles incoming messages/events and updates the model. Returns `(tea.Model, tea.Cmd)`.
    - `View()`: Renders the component's UI as a string based on the current model state.
  - Keep `Update` functions focused; delegate complex logic to helper methods or separate functions.
  - Use `tea.BatchMsg` to batch multiple commands returned from `Update`.

- **State Management:**
  - Prefer local state within each component's `Model`.
  - For shared state or communication between components, use `tea.Msg` passing:
    - Parent components can pass messages down during their `Update`.
    - Child components can send messages up for the parent (or root) `Update` function to handle.
  - Avoid global state for TUI components. If necessary, inject shared dependencies (like services or data repositories) into the root TUI model during initialization.

- **Interaction & Messages:**
  - Define custom `tea.Msg` types (structs or simple types) for specific application events (e.g., `dataLoadedMsg`, `errorOccurredMsg`, `itemSelectedMsg`).
  - Use `tea.KeyMsg` for handling keyboard input within `Update`. Check `key.Type` or use `key.Matches`.
  - Commands (`tea.Cmd`) should be used for I/O operations (API calls, DB access, timers) to avoid blocking the `Update` loop. The results of these commands should be sent back as `tea.Msg`.

- **Styling:**
  - Use `lipgloss` for styling text, borders, layouts, etc.
  - Define reusable styles in `internal/util/styles.go` and reference them within component `View` methods.
  - Ensure styles adapt reasonably to different terminal sizes where possible.

- **Layout:**
  - Use `lipgloss` functions like `lipgloss.JoinVertical`, `lipgloss.JoinHorizontal`, and `lipgloss.Place` for arranging components.

## 📁 File Structure

Use this structure as a guide when creating or updating files:

```text
app/
  app.go
cmd/
  main.go
  firstcommand/
    firstcommand.go
  root/
    root.go
internal/
  controller/
  service/
  repository/
  model/
  config/
  middleware/
  utils/
pkg/
  logger/
  errors/
tests/
  integration/
```

- When creating commands, create a logical folder structure that mimic the command structure as much as possible. This should at least extend one directory deep.
- Don't over engineer the file structure. If a command is small or shares commonality with other commands, it can be placed in the same command folder.

## 🧶 Patterns

### ✅ Patterns to Follow

- Use **Clean Architecture** and **Repository Pattern**.
- Implement input validation using Go structs and validation tags (e.g., [go-playground/validator](https://github.com/go-playground/validator)).
- Use custom error types for wrapping and handling business logic errors.
- Logging should be handled via `charmbracelet/log`.
- Use dependency injection via constructors (avoid global state).
- Keep `main.go` minimal—delegate to `internal`.

### 🚫 Patterns to Avoid

- Don’t use global state unless absolutely required.
- Don’t hardcode config—use environment variables or config files.
- Don’t panic or exit in library code; return errors instead.
- Don’t expose secrets—use `.env` or secret managers.
- Avoid embedding business logic in HTTP handlers.

## 🧪 Testing Guidelines

- Use `testing` and [testify](https://github.com/stretchr/testify) for assertions and mocking.
- Place unit tests alongside the code they test using Go's standard `_test.go` convention (e.g., `internal/executor/executor_test.go`).
- Place integration tests under `tests/integration/`.
- Mock external services (e.g., DB, APIs) using interfaces and mocks for unit tests.
- Include table-driven tests for functions with many input variants.
- Follow TDD for core business logic.
- **Cache Persistence during Testing:** When making changes to the code or testing runs, do not truncate or delete records from the `cache` table unless the schema itself has changed. This prevents unnecessary and expensive redundant API/LLM calls.

## 🔁 Iteration & Review

- Review Copilot output before committing.
- Refactor generated code to ensure readability and testability.
- Use comments to give Copilot context for better suggestions.
- Regenerate parts that are unidiomatic or too complex.

## 📚 References

- [Go Style Guide](https://google.github.io/styleguide/go/)
- [Effective Go](https://go.dev/doc/effective_go)
- [Standard Go Project Layout](https://github.com/golang-standards/project-layout)
- [Testify](https://github.com/stretchr/testify)
- [Go Validator](https://github.com/go-playground/validator)
- [Charmbracelet Bubbletea Documentation](https://pkg.go.dev/github.com/charmbracelet/bubbletea)

# Go CLI Tool Principles & Rules

## The Main Wrapper
* **Keep `main` Small:** The `main()` function should be a thin wrapper. Its only jobs are to call a `Run`-style function and call `os.Exit()`.
* **No Logic in Main:** Business logic, flag parsing, and error handling should live in a testable function, not in `main`.

## The Run Pattern
* **Signature:** Implement a `Run()` function that accepts dependencies (like I/O streams) and returns an exit code.
* **Example:**
```go
func main() {
    os.Exit(Run(os.Args[1:], os.Stdout, os.Stderr))
}

func Run(args []string, stdout, stderr io.Writer) int {
    // Logic goes here
    return 0
}
```

## Testable I/O
* **Abstract the Streams:** Never use `fmt.Println()` or `os.Stdout` directly inside your tool's logic.
* **Inject Writers:** Pass `io.Writer` (for stdout/stderr) and `io.Reader` (for stdin) into your functions. This allows tests to capture output using `bytes.Buffer`.
* **Silent by Default:** Tools should not print anything unless requested or necessary for the tool's primary function.

## Flag Handling
* **Standard Library:** Use the `flag` package unless complex subcommands require an external library.
* **Scoped FlagSets:** In the `Run()` function, create a new `flag.NewFlagSet` rather than using the global `flag` variables. This prevents state leakage between tests.
* **Usage Messages:** Always provide helpful usage strings for flags.

## Error Handling & Exit Codes
* **Errors to Stderr:** All error messages must be sent to the `stderr` stream, never `stdout`.
* **Meaningful Exit Codes:** * `0`: Success.
    * `1`: General error / Catch-all.
    * `2`: Misuse of shell built-ins (or flag parsing errors).
* **Don't Panic:** Never use `panic()` for expected errors (file not found, invalid input). Return an error and handle it gracefully in `Run`.

## Project Structure
* **Separation:** Keep the CLI "plumbing" (parsing flags, environment variables) separate from the "library" logic (the actual work the tool does).
* **Internal Package:** If the tool is a standalone utility, keep logic in an `internal/` directory to prevent other projects from importing it as a library unless intended.

## Functional Options
* **Configuration:** For complex tools, use the "Functional Options" pattern to configure the tool's behavior. This keeps the API clean and extensible.

# Go Testing Principles & Rules

This document outlines the testing standards for this project, based on the principles of "The Power of Go: Tests" by John Arundel. Use these rules to guide all AI-generated code and manual refactoring.

## General Philosophy
* **Tests are code:** Treat test code with the same care as production code. It should be readable, maintainable, and clean.
* **Clarity over Cleverness:** A test should be so simple that it is obviously correct. Avoid complex logic or abstractions inside test functions.
* **Standard Library First:** Favor the built-in `testing` package. Avoid assertion libraries (like `testify`) unless explicitly instructed; use plain `if` statements for comparisons to keep error messages meaningful.

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
* **Black-box Testing:** Use the `_test` package suffix (e.g., `package stringutil_test`) for integration tests or when testing the public API. This ensures you are testing the code as a user of the package.
* **Internal Tests:** Use the same package name (e.g., `package stringutil`) only when you need to test unexported functions or variables.

## Test Helpers
* **t.Helper():** When creating reusable helper functions (e.g., a setup function or a custom check), call `t.Helper()` at the start. This ensures that failure line numbers point to the caller of the helper, not the helper itself.