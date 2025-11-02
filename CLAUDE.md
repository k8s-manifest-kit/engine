# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

The **Engine** component of k8s-manifest-kit orchestrates the rendering pipeline for Kubernetes manifests. It coordinates renderers and applies filters and transformers at three distinct stages of the pipeline.

**Part of the [k8s-manifest-kit](https://github.com/k8s-manifest-kit) organization.**

## Documentation

- **[README.md](README.md)** - Module overview and quick start
- **[docs/design.md](docs/design.md)** - Architecture, three-level pipeline, and design decisions
- **[docs/development.md](docs/development.md)** - Coding conventions, testing guidelines, and contribution guide

## Quick Reference

### Core Types (pkg/types)

```go
// Renderer is the interface that all concrete renderers implement.
type Renderer interface {
    Process(ctx context.Context, values map[string]any) ([]unstructured.Unstructured, error)
}

// Filter is a function that decides whether to keep an object.
type Filter func(ctx context.Context, object unstructured.Unstructured) (bool, error)

// Transformer is a function that transforms an object.
type Transformer func(ctx context.Context, object unstructured.Unstructured) (unstructured.Unstructured, error)
```

### Three-Level Pipeline

1. **Renderer-specific**: Filters/transformers applied inside each renderer's `Process()`
2. **Engine-level**: Filters/transformers applied to all renders via `engine.New()`
3. **Render-time**: Filters/transformers applied to a single `Render()` call

See [docs/design.md#5-three-level-filteringtransformation](docs/design.md#5-three-level-filteringtransformation) for details.

### Engine Usage

```go
// Create engine with renderers and engine-level F/T
e := engine.New(
    engine.WithRenderer(helmRenderer),
    engine.WithFilter(namespaceFilter),
    engine.WithTransformer(labelTransformer),
)

// Render with render-time F/T and values
objects, err := e.Render(ctx,
    engine.WithRenderFilter(kindFilter),
    engine.WithRenderTransformer(annotationTransformer),
    engine.WithValues(map[string]any{
        "replicaCount": 3,
    }),
)
```

### Filter Composition (pkg/filter/compose.go)

```go
// Boolean logic
filter.Or(filter1, filter2)     // Any must pass
filter.And(filter1, filter2)    // All must pass
filter.Not(filter1)              // Invert result

// Conditional
filter.If(condition, thenFilter) // Apply thenFilter only if condition passes
```

### Transformer Composition (pkg/transformer/compose.go)

```go
// Sequential execution
transformer.Chain(t1, t2, t3)    // Apply in sequence

// Conditional
transformer.If(condition, then)  // Apply only if condition passes

// Multi-branch
transformer.Switch([]transformer.Case{
    {When: filter1, Then: transformer1},
    {When: filter2, Then: transformer2},
}, defaultTransformer)
```

### Built-in Filters and Transformers

**Filters**:
- `namespace.Filter()`, `namespace.Exclude()`
- `labels.HasLabel()`, `labels.MatchLabels()`, `labels.Selector()`
- `name.Exact()`, `name.Prefix()`, `name.Suffix()`, `name.Regex()`
- `annotations.HasAnnotation()`, `annotations.MatchAnnotations()`
- `gvk.Filter()`
- `jq.Filter(expression)`

**Transformers**:
- `namespace.Set()`, `namespace.EnsureDefault()`
- `name.SetPrefix()`, `name.SetSuffix()`, `name.Replace()`
- `labels.Transform()`, `labels.Remove()`, `labels.RemoveIf()`
- `annotations.Transform()`, `annotations.Remove()`, `annotations.RemoveIf()`
- `jq.Transform(expression)`

## Development

**Run tests:**
```bash
make test
```

**Format and lint:**
```bash
make fmt
make lint
```

**Run benchmarks:**
```bash
go test -v ./pkg/... -run=^$ -bench=.
```

For detailed development information:
- **Build commands**: See [docs/development.md#setup-and-build](docs/development.md#setup-and-build)
- **Coding conventions**: See [docs/development.md#coding-conventions](docs/development.md#coding-conventions)
- **Testing guidelines**: See [docs/development.md#testing-guidelines](docs/development.md#testing-guidelines)
- **Adding filters/transformers**: See [docs/development.md#extensibility](docs/development.md#extensibility)
- **Code review guidelines**: See [docs/development.md#code-review-guidelines](docs/development.md#code-review-guidelines)

## Testing Conventions

- Use vanilla Gomega (dot import)
- All test data as package-level constants
- Benchmark naming: `Benchmark<Component><TestName>`
- Use `t.Context()` instead of `context.Background()`

See [docs/development.md#testing-guidelines](docs/development.md#testing-guidelines) for complete testing practices.

## Key Concepts

### Pipeline Execution Order

```
1. Renderer.Process() + renderer-specific F/T
2. Aggregate results from all renderers
3. Engine-level filters
4. Render-time filters (merged)
5. Engine-level transformers
6. Render-time transformers (merged)
7. Return final objects
```

### Filter Logic

- Multiple filters use **AND logic** - object must pass ALL filters
- Short-circuit evaluation for performance
- First filter rejection stops evaluation

### Transformer Chaining

- Transformers applied **sequentially** - output of one feeds into next
- Order matters! Document transformation order assumptions
- First error stops the chain

### Error Handling

- Use typed errors: `FilterError`, `TransformerError`
- Wrap errors with `fmt.Errorf` and `%w`
- Context propagation for cancellation
- First error stops processing and is returned

## Architecture

The Engine is designed around these principles:

1. **Three-Level Pipeline**: Renderer-specific → Engine-level → Render-time
2. **Composability**: Filters and transformers compose naturally
3. **Type Safety**: Compile-time type checking for options
4. **Functional Options**: Flexible configuration with dual patterns
5. **Context Propagation**: Full cancellation support
6. **Extensibility**: Easy to add new filters/transformers
7. **Immutability**: Transformers create new objects, never modify inputs
8. **Error Context**: Rich error information for debugging

See [docs/design.md#10-design-principles](docs/design.md#10-design-principles) for more details.

## Dependencies

- **k8s-manifest-kit/pkg**: Shared utilities (caching, merging, JQ, K8s object utils)
- **k8s.io/apimachinery**: Kubernetes API machinery
- **k8s.io/api**: Kubernetes API types

Renderers are **separate** modules that depend on the engine, not part of this repository.

