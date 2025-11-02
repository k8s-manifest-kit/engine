# Development Guide: Engine

This document provides coding conventions, testing guidelines, and contribution practices for developing the Engine component of k8s-manifest-kit.

For architectural information and design decisions, see [design.md](design.md).

**Part of the [k8s-manifest-kit](https://github.com/k8s-manifest-kit) organization.**

## Table of Contents

1. [Setup and Build](#setup-and-build)
2. [Coding Conventions](#coding-conventions)
3. [Testing Guidelines](#testing-guidelines)
4. [Extensibility](#extensibility)
5. [Code Review Guidelines](#code-review-guidelines)

## Setup and Build

### Build Commands

```bash
# Run all tests
make test

# Format code
make fmt

# Run linter
make lint

# Fix linting issues automatically
make lint/fix

# Clean build artifacts and test cache
make clean

# Update dependencies
make deps
```

### Test Commands

```bash
# Run all tests with verbose output
go test -v ./...

# Run tests in a specific package
go test -v ./pkg/filter

# Run a specific test
go test -v ./pkg -run TestEngine

# Run benchmarks
go test -v ./pkg/... -run=^$ -bench=.
```

## Coding Conventions

### Functional Options Pattern

All struct initialization uses the functional options pattern for flexible, extensible configuration.

**Define Options as Interfaces:**
```go
type Option[T any] interface {
    ApplyTo(target *T)
}
```

**Provide Both Function-Based and Struct-Based Options:**
```go
// Function-based option
func WithRenderer(r types.Renderer) EngineOption {
    return util.FunctionalOption[engineOptions](func(e *engineOptions) {
        e.renderers = append(e.renderers, r)
    })
}

// Struct-based option for bulk configuration
type EngineOptions struct {
    Renderers    []types.Renderer
    Filters      []types.Filter
    Transformers []types.Transformer
}

func (opts EngineOptions) ApplyTo(e *engineOptions) {
    e.renderers = opts.Renderers
    e.filters = opts.Filters
    e.transformers = opts.Transformers
}
```

**Guidelines:**
- For slice/map fields in struct-based options, use the type directly (not pointers)
- Place all options and related methods in `*_option.go` files
- Provide both patterns to support different use cases

### Error Handling Conventions

* Errors are wrapped using `fmt.Errorf` with `%w` for proper error chain propagation
* Context is passed through the entire pipeline for cancellation support
* First error encountered stops processing and is returned immediately
* Use `errors.As()` to extract typed errors from error chains
* Use `errors.Is()` to check for specific underlying errors

### Package Organization

```
engine/
├── pkg/
│   ├── types/           # Core type definitions
│   ├── engine.go        # Engine implementation
│   ├── engine_option.go # Functional options
│   ├── engine_test.go   # Engine tests
│   ├── pipeline/        # Pipeline execution
│   ├── filter/          # Filter implementations
│   └── transformer/     # Transformer implementations
```

Each component follows the pattern:
- `component.go` - Main implementation
- `component_option.go` - Functional options (if needed)
- `component_test.go` - Tests

## Testing Guidelines

### Test Framework

- Use vanilla Gomega (not Ginkgo)
- Use dot imports for Gomega: `import . "github.com/onsi/gomega"`
- Prefer `Should` over `To`
- For error validation: `Should(HaveOccurred())` / `ShouldNot(HaveOccurred())`
- Use subtests (`t.Run`) for organizing related test cases
- Use `t.Context()` instead of `context.Background()` or `context.TODO()` (Go 1.24+)
- **Mocking**: testify/mock is welcome and recommended for mocking interfaces
  - "Vanilla Gomega" means not using Ginkgo's BDD style (`Describe`, `Context`, `It`)
  - Other testing tools like testify/mock are encouraged for mocking

**Example:**
```go
func TestEngine(t *testing.T) {
    g := NewWithT(t)
    ctx := t.Context()

    t.Run("should render correctly", func(t *testing.T) {
        result, err := engine.Render(ctx)
        g.Expect(err).ShouldNot(HaveOccurred())
        g.Expect(result).Should(HaveLen(3))
    })
}
```

### Test Data Organization

**String literals** (YAML, JSON, etc.) should be defined as package-level constants for reusability.

**Good - String Literals as Constants:**
```go
const testManifestYAML = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
`

func TestSomething(t *testing.T) {
    // Use testManifestYAML constant
}
```

**Bad - Inline String Literals:**
```go
func TestSomething(t *testing.T) {
    yaml := `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
`  // WRONG: inline string literal should be a constant
}
```

**Good - Object Builder Functions:**
```go
// Helper functions to create fresh test objects
func makePod(name string) unstructured.Unstructured {
    return unstructured.Unstructured{
        Object: map[string]any{
            "apiVersion": "v1",
            "kind":       "Pod",
            "metadata":   map[string]any{"name": name},
        },
    }
}

func TestSomething(t *testing.T) {
    pod := makePod("test-pod")  // Fresh object per test
}
```

**Rules:**
- **String literals** (YAML, JSON, raw strings): Define as package-level constants
- **Complex objects** (unstructured.Unstructured, structs): Use helper builder functions
  - Objects should be created fresh for each test to avoid shared state
  - Builder functions provide flexibility (parameterized values)
- Define constants at the top of test files, grouped by test scenario
- Use descriptive names that indicate purpose (e.g., `testConfigMapYAML`, `makePod`)
- Add comments to group related constants and functions
- This makes tests more readable and data reusable across tests

### Benchmark Naming

- Include component name in benchmark tests
- Format: `Benchmark<Component><TestName>`
- Examples: `BenchmarkEngineRender`, `BenchmarkFilterComposition`, `BenchmarkTransformerChain`

### Test Strategy

**Unit Tests**: Test each component in isolation
- Engine: Test rendering pipeline, filter/transformer application
- Filters: Test with matching and non-matching objects
- Transformers: Verify transformations are applied correctly
- Pipeline: Test filter AND logic and transformer chaining

**Integration Tests**: Test the full pipeline
- Multiple renderers with engine-level F/T
- Render-time options merging with engine-level
- Error handling throughout the pipeline

**Benchmark Tests**: Performance testing
- Named with component prefix: `BenchmarkEngineRender`, `BenchmarkFilterAnd`
- Test filter/transformer composition performance
- Measure pipeline overhead

**Test Patterns**:
- Use vanilla Gomega (no Ginkgo)
- Subtests via `t.Run()`
- Use `t.Context()` instead of `context.Background()`
- Mock renderers for engine tests to avoid external dependencies

## Extensibility

### Adding New Filter/Transformer

1. Define a constructor function that returns `types.Filter` or `types.Transformer`
2. If configuration is needed, accept parameters and return a closure
3. Add via `engine.WithFilter`/`engine.WithTransformer` for engine-level

**Example:**

```go
// pkg/filter/custom/custom.go
package custom

import (
    "context"
    "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
    "github.com/k8s-manifest-kit/engine/pkg/types"
)

func MyCustomFilter(threshold int) types.Filter {
    return func(ctx context.Context, obj unstructured.Unstructured) (bool, error) {
        // Custom logic using threshold
        return true, nil
    }
}

// Usage
filter := custom.MyCustomFilter(10)
e := engine.New(engine.WithFilter(filter))
```

### Adding Filter/Transformer Composition Functions

1. Create a new function in `pkg/filter/compose.go` or `pkg/transformer/compose.go`
2. Return a composed `types.Filter` or `types.Transformer`
3. Add tests demonstrating the composition

**Example:**

```go
// pkg/filter/compose.go

// All returns a filter that requires ALL objects in a collection to pass a filter.
func All(filter types.Filter) types.Filter {
    return func(ctx context.Context, obj unstructured.Unstructured) (bool, error) {
        // Custom composition logic
    }
}
```

## Code Review Guidelines

### Linter Rules

All code must pass `make lint` before submission. Key linter rules:

- **goconst**: Extract repeated string literals to constants
- **gosec**: No hardcoded secrets (use `//nolint:gosec` only for test data with comment explaining why)
- **staticcheck**: Follow all suggestions
- **Comment formatting**: All comments must end with periods

### Git Commit Conventions

**Commit Message Format:**
```
<type>: <subject>

<body>

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `refactor`: Code refactoring (no functional changes)
- `test`: Adding or updating tests
- `docs`: Documentation changes
- `build`: Build system or dependency changes
- `chore`: Maintenance tasks

**Subject:**
- Use imperative mood ("add feature" not "added feature")
- Don't capitalize first letter
- No period at the end
- Max 72 characters

**Body:**
- Explain what and why (not how)
- Separate from subject with blank line
- Wrap at 72 characters
- Use bullet points for multiple items

**Example:**
```
feat: add conditional transformer composition with Switch

This commit introduces Switch composition for transformers:

- Switch evaluates cases in order until a match is found
- Each case has a When (filter) and Then (transformer)
- Supports default transformer when no cases match
- Lazy evaluation for performance

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>
```

### Pull Request Checklist

Before submitting a PR:
- [ ] All tests pass (`make test`)
- [ ] Linter passes (`make lint`)
- [ ] Code formatted (`make fmt`)
- [ ] New tests added for new features
- [ ] Documentation updated (design.md or development.md as needed)
- [ ] All test data extracted to package-level constants
- [ ] Benchmark tests follow naming convention
- [ ] Error handling follows conventions
- [ ] Functional options pattern used for configuration

### Code Style

- **Function signatures**: Each parameter must have its own type declaration (never group parameters with same type)
- **Comments**: Explain *why*, not *what*. Focus on non-obvious behavior, edge cases, and relationships
- **Error wrapping**: Always use `fmt.Errorf` with `%w` for error chains
- **Context propagation**: Pass context through all layers for cancellation support
- **Zero values**: Leverage zero value semantics instead of pointers where appropriate

## Best Practices

### Filter and Transformer Design

1. **Single Responsibility**: Each filter/transformer should do one thing well
2. **Composability**: Design filters/transformers to compose naturally
3. **Immutability**: Never modify input objects; always create new ones in transformers
4. **Error Context**: Use typed errors (FilterError, TransformerError) for rich context
5. **Performance**: Consider short-circuit evaluation in composition
6. **Documentation**: Document expected input conditions and guarantees

### Engine Usage Patterns

1. **Engine-Level vs Render-Time**: Use engine-level for consistent behavior, render-time for one-off operations
2. **Filter Before Transform**: Apply filters early to reduce transformation work
3. **Compose Thoughtfully**: Order matters in transformer chains; document assumptions
4. **Error Handling**: Always handle errors from Render() - they provide rich context
5. **Context Cancellation**: Respect context cancellation throughout the pipeline

