# go-newliner

A Go linter built on the `golang.org/x/tools/go/analysis` framework that
enforces blank-line formatting rules in Go code.

## Installation

```sh
go install github.com/carsonak/go-newliner/cmd/go-newliner@latest
```

## Usage

```sh
go-newliner ./...
```

All diagnostics include `SuggestedFixes` so editors and `go fix` can auto-insert
the missing blank lines.

## Rules

### Rule 1: Closing Curly Braces

A closing brace `}` of a block statement (`if`, `for`, `range`, `switch`, `select`, etc.) must be followed by exactly one blank line before the next statement.

```go
// Bad
if ok {
    doWork()
}
doMore() // diagnostic: "closing brace should be followed by a blank line"

// Good
if ok {
    doWork()
}

doMore()
```

**Exceptions:**

- **Defer cleanup** — If the `if` block checks any variable from the preceding assignment against `nil`, and is immediately followed by a `defer` that cleans up a variable from that same assignment, no blank line is required:

  ```go
  f, err := os.Open(name)
  if err != nil {
      return err
  }
  defer f.Close() // allowed: no blank line needed
  ```

  This works regardless of the error variable's name (`err`, `readErr`, etc.).

- **Trailing close** — If the next non-whitespace character is `}`, `]`, or `)`, no blank line is required:

  ```go
  if true {
      if true {
          fmt.Println("nested")
      }
  } // no blank line needed before outer }
  ```

### Rule 2: Declarations

A block of variable declarations (including short variable declarations `:=`) must be followed by exactly one blank line.

```go
// Bad
x := 1
fmt.Println(x) // diagnostic: "declaration should be followed by a blank line"

// Good
x := 1

fmt.Println(x)
```

Contiguous declarations are treated as a group — only the last one in the group is checked.

**Exception:**

- **Nil check** — If the next statement is an `if` that checks any variable from the declaration block against `nil`, no blank line is required:

  ```go
  x, err := doSomething()
  if err != nil { // allowed: no blank line needed
      return err
  }

  conn, connErr := dial()
  if connErr != nil { // also allowed
      return connErr
  }
  ```

### Rule 3: Goroutines

A block of `go` statements must be followed by exactly one blank line.

```go
// Bad
go serve()
fmt.Println("started") // diagnostic: "go statement should be followed by a blank line"

// Good
go serve()

fmt.Println("started")
```

**Exception:**

- **Trailing close** — If the next non-whitespace character is `}`, no blank line is required (defers to Rule 1).

## Integration

### As a library

Import the analyzer to embed it in a custom multi-checker:

```go
import "github.com/carsonak/go-newliner/analyzer"

// analyzer.Analyzer is an *analysis.Analyzer
```

### With golangci-lint

`go-newliner` can be used as a plugin with [golangci-lint](https://golangci-lint.run/contributing/new-linters/).

## Development

```sh
# Run tests
go test ./...

# Build
go build -o go-newliner ./cmd/go-newliner/
```
