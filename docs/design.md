# Design Document: Engine

## 1. Introduction

This document outlines the design of the **Engine** component of the k8s-manifest-kit ecosystem. The Engine is responsible for orchestrating the rendering pipeline, coordinating renderers, and applying filters and transformers to Kubernetes manifest objects.

The Engine provides a robust and flexible three-level filter/transformer pipeline with a functional options pattern for configuration.

**Part of the [k8s-manifest-kit](https://github.com/k8s-manifest-kit) organization.**

## 2. High-Level Architecture

The Engine orchestrates the rendering process by coordinating `Renderer` instances and applying filters and transformers at three distinct stages:

1. **Renderer-specific**: Applied within each renderer's `Process()` method
2. **Engine-level**: Applied to aggregated results from all renderers
3. **Render-time**: Applied to a specific `Render()` call, merged with engine-level filters/transformers

```
┌────────────────────┐
│ Engine             │
│ Configuration      │
└──────────┬─────────┘
           │
           ├──► Renderer 1 ──► Process + Renderer F/T ──┐
           │                                            │
           ├──► Renderer 2 ──► Process + Renderer F/T ──┼──► Aggregate
           │                                            │    Objects
           └──► Renderer N ──► Process + Renderer F/T ──┘
                                                         │
                                                         ▼
                                              Engine-Level Filters
                                                         │
                                                         ▼
                                           Engine-Level Transformers
                                                         │
                                                         ▼
                                              Render-Time Filters
                                                         │
                                                         ▼
                                           Render-Time Transformers
                                                         │
                                                         ▼
                                                   Final Objects
```

## 3. Core Concepts

### 3.1. Package Structure

```
engine/
├── pkg/
│   ├── types/           # Core type definitions
│   │   ├── types.go     # Renderer, Filter, Transformer
│   │   └── annotations.go # Source annotation constants
│   ├── engine.go        # Engine implementation
│   ├── engine_option.go # Functional options
│   ├── engine_test.go   # Engine tests
│   ├── pipeline/        # Pipeline execution
│   │   ├── apply.go     # ApplyFilters, ApplyTransformers, Apply
│   │   └── apply_test.go
│   ├── filter/          # Filter implementations and composition
│   │   ├── compose.go   # Filter composition (Or, And, Not, If)
│   │   ├── error.go     # FilterError type
│   │   ├── jq/          # JQ-based filtering
│   │   └── meta/        # Metadata-based filters
│   │       ├── annotations/  # Annotation filters
│   │       ├── gvk/         # GroupVersionKind filters
│   │       ├── labels/      # Label filters
│   │       ├── name/        # Name filters
│   │       └── namespace/   # Namespace filters
│   └── transformer/     # Transformer implementations and composition
│       ├── compose.go   # Transformer composition (Chain, If, Switch)
│       ├── error.go     # TransformerError type
│       ├── jq/          # JQ-based transformation
│       └── meta/        # Metadata-based transformers
│           ├── annotations/  # Annotation transformers
│           ├── labels/       # Label transformers
│           ├── name/         # Name transformers
│           └── namespace/    # Namespace transformers
```

### 3.2. Core Types (pkg/types/types.go)

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

### 3.3. Engine (pkg/engine.go)

The `Engine` struct manages the rendering pipeline:

```go
type Engine struct {
    options engineOptions
}

// New creates a new Engine with the given options.
func New(opts ...EngineOption) *Engine

// Render processes all registered renderers and applies filters/transformers.
func (e *Engine) Render(ctx context.Context, opts ...RenderOption) ([]unstructured.Unstructured, error)
```

**Rendering Pipeline:**

1. Collect render-time values from `Render()` options
2. Process each renderer sequentially via `renderer.Process(ctx, values)`
3. Aggregate all objects from all renderers
4. Apply engine-level filters (configured via `New()`)
5. Apply engine-level transformers (configured via `New()`)
6. Apply render-time filters (passed to `Render()`)
7. Apply render-time transformers (passed to `Render()`)

**Render-Time Values:**

Render-time values are passed to all renderers via the `values` parameter in `Process()`. Renderers that support dynamic values deep merge these values with Source-level values, with render-time values taking precedence.

## 4. Configuration Pattern

The Engine uses the **functional options pattern** with dual support:

1. **Function-based options**: `WithRenderer(r)`, `WithFilter(f)`, `WithTransformer(t)`
2. **Struct-based options**: Direct struct literals for bulk configuration

All options implement the `Option[T]` interface from `github.com/k8s-manifest-kit/pkg/util`:

```go
type Option[T any] interface {
    ApplyTo(target *T)
}
```

### 4.1. Engine Options

```go
// Engine configuration
e := engine.New(
    engine.WithRenderer(helmRenderer),
    engine.WithFilter(appsV1Filter),
    engine.WithTransformer(labelTransformer),
)

// Or using struct-based options
e := engine.New(&engine.EngineOptions{
    Renderers: []types.Renderer{helmRenderer},
    Filters: []types.Filter{appsV1Filter},
    Transformers: []types.Transformer{labelTransformer},
})
```

### 4.2. Render-Time Options

```go
// Function-based
objects, err := e.Render(ctx,
    engine.WithRenderFilter(namespaceFilter),
    engine.WithRenderTransformer(annotationTransformer),
    engine.WithValues(map[string]any{
        "replicaCount": 3,
        "image": map[string]any{
            "tag": "v2.0",
        },
    }),
)

// Struct-based
objects, err := e.Render(ctx, engine.RenderOptions{
    Filters: []types.Filter{namespaceFilter},
    Transformers: []types.Transformer{annotationTransformer},
    Values: map[string]any{
        "replicaCount": 3,
        "image": map[string]any{
            "tag": "v2.0",
        },
    },
})
```

## 5. Three-Level Filtering/Transformation

The Engine supports filtering and transformation at three distinct stages:

### 5.1. Renderer-Specific (Earliest)

Applied by individual renderers during their `Process()` method, before results are returned.

```go
// Renderer-specific filters/transformers are configured on the renderer
helmRenderer, err := helm.New(
    []helm.Source{...},
    helm.WithFilter(onlyDeploymentsFilter),         // Applied by Helm only
    helm.WithTransformer(addHelmLabelsTransformer), // Applied by Helm only
)
```

**Use when**: You want filtering/transformation specific to one renderer's output.

### 5.2. Engine-Level (Middle)

Applied to aggregated results from all renderers on every `Render()` call.

```go
e := engine.New(
    engine.WithRenderer(helmRenderer),
    engine.WithFilter(namespaceFilter),      // Applied to ALL renders
    engine.WithTransformer(addCommonLabels), // Applied to ALL renders
)
```

**Use when**: You want consistent filtering/transformation across all renders.

### 5.3. Render-Time (Latest)

Applied to a single `Render()` call, merged with engine-level filters/transformers. Render-time values are also passed to renderers at this stage.

```go
objects, err := e.Render(ctx,
    engine.WithRenderFilter(kindFilter),               // Applied only to this render
    engine.WithRenderTransformer(envLabelTransformer), // Applied only to this render
    engine.WithValues(map[string]any{                  // Passed to renderers for this render
        "replicaCount": 3,
        "image": map[string]any{
            "tag": "v2.0",
        },
    }),
)
```

**Use when**:
- You need one-off filtering/transformation for a specific operation
- You need to override renderer values for a specific render call

**Important**:
- Render-time filters/transformers are *additive* - they append to engine-level options
- Render-time values deep merge with Source-level values (where supported by renderer)

### 5.4. Execution Order

```
1. Renderer processes inputs + applies renderer-specific F/T
2. Engine aggregates all renderer results
3. Engine applies engine-level filters
4. Engine applies render-time filters (merged)
5. Engine applies engine-level transformers
6. Engine applies render-time transformers (merged)
7. Returns final objects
```

## 6. Filters and Transformers

Filters and transformers are implemented as constructor functions that return `types.Filter` or `types.Transformer` closures. The library provides composition functions for building complex logic.

### 6.1. Filter Composition (pkg/filter)

Combinators for building complex filter logic:

```go
// Boolean Logic
func Or(filters ...types.Filter) types.Filter   // Any filter must pass
func And(filters ...types.Filter) types.Filter  // All filters must pass
func Not(filter types.Filter) types.Filter      // Inverts filter result

// Conditional
func If(condition types.Filter, then types.Filter) types.Filter  // Apply 'then' only if condition passes

// Usage: Complex namespace and kind filtering
filter := filter.Or(
    filter.And(
        namespace.Filter("production"),
        gvk.Filter(appsv1.SchemeGroupVersion.WithKind("Deployment")),
    ),
    filter.And(
        namespace.Filter("staging"),
        gvk.Filter(corev1.SchemeGroupVersion.WithKind("Service")),
    ),
)
```

**Features:**
* Short-circuit evaluation for performance
* Arbitrary nesting depth
* Clear, readable filter logic
* Composable with all filter types

### 6.2. Transformer Composition (pkg/transformer)

Combinators for building complex transformation pipelines:

```go
// Sequential Execution
func Chain(transformers ...types.Transformer) types.Transformer  // Apply transformers in sequence

// Conditional Transformation
func If(condition types.Filter, transformer types.Transformer) types.Transformer  // Apply only if condition passes

// Multi-branch Logic
type Case struct {
    When types.Filter
    Then types.Transformer
}
func Switch(cases []Case, defaultTransformer types.Transformer) types.Transformer  // First matching case wins

// Usage: Environment-specific transformations
transformer := transformer.Switch(
    []transformer.Case{
        {
            When: namespace.Filter("production"),
            Then: transformer.Chain(
                labels.Set(map[string]string{"env": "prod"}),
                annotations.Set(map[string]string{"tier": "critical"}),
            ),
        },
        {
            When: namespace.Filter("staging"),
            Then: labels.Set(map[string]string{"env": "staging"}),
        },
    },
    labels.Set(map[string]string{"env": "dev"}), // default
)
```

**Features:**
* Lazy evaluation - transformers only execute when conditions match
* Early exit in Switch - first matching case wins
* Composable with all transformer types
* Type-safe Case definitions

### 6.3. Built-in Filters and Transformers

The Engine includes comprehensive built-in filters and transformers for common operations:

**Filters:**
- Namespace: `namespace.Filter()`, `namespace.Exclude()`
- Labels: `labels.HasLabel()`, `labels.MatchLabels()`, `labels.Selector()`
- Name: `name.Exact()`, `name.Prefix()`, `name.Suffix()`, `name.Regex()`
- Annotations: `annotations.HasAnnotation()`, `annotations.MatchAnnotations()`
- GVK: `gvk.Filter()`
- JQ: `jq.Filter(expression)`

**Transformers:**
- Namespace: `namespace.Set()`, `namespace.EnsureDefault()`
- Name: `name.SetPrefix()`, `name.SetSuffix()`, `name.Replace()`
- Labels: `labels.Transform()`, `labels.Remove()`, `labels.RemoveIf()`
- Annotations: `annotations.Transform()`, `annotations.Remove()`, `annotations.RemoveIf()`
- JQ: `jq.Transform(expression)`

See the respective package documentation for detailed usage.

## 7. Filter and Transformer Logic

### 7.1. Filter Logic (AND Semantics)

Multiple filters are combined with **AND logic** - an object must pass ALL filters to be kept.

```go
engine.New(
    engine.WithFilter(namespaceFilter),  // Must pass this
    engine.WithFilter(kindFilter),        // AND must pass this
)
```

Implementation in `pipeline.ApplyFilters()` returns false as soon as any filter rejects an object.

### 7.2. Transformer Chaining

Transformers are applied **sequentially** - the output of one becomes the input to the next.

```go
engine.New(
    engine.WithTransformer(labels.Set(map[string]string{"env": "prod"})),
    engine.WithTransformer(annotations.Set(map[string]string{"version": "1.0"})),
)
```

**Order matters!** Implementation in `pipeline.ApplyTransformers()` processes transformers in sequence.

## 8. Pipeline Execution (pkg/pipeline)

### 8.1. Filter/Transformer Application

```go
// Apply filters with AND logic
func ApplyFilters(ctx context.Context, objects []unstructured.Unstructured, filters []types.Filter) ([]unstructured.Unstructured, error)

// Apply transformers in sequence
func ApplyTransformers(ctx context.Context, objects []unstructured.Unstructured, transformers []types.Transformer) ([]unstructured.Unstructured, error)

// Apply both filters and transformers
func Apply(ctx context.Context, objects []unstructured.Unstructured, filters []types.Filter, transformers []types.Transformer) ([]unstructured.Unstructured, error)
```

## 9. Error Handling

### 9.1. Typed Errors

The Engine provides typed errors for filter and transformer failures:

**FilterError (pkg/filter/error.go):**
```go
type FilterError struct {
    Object unstructured.Unstructured  // The object that failed filtering
    Err    error                       // The underlying error
}
```

**TransformerError (pkg/transformer/error.go):**
```go
type TransformerError struct {
    Object unstructured.Unstructured  // The object that failed transformation
    Err    error                       // The underlying error
}
```

### 9.2. Error Handling Conventions

* Errors are wrapped using `fmt.Errorf` with `%w` for proper error chain propagation
* Context is passed through the entire pipeline for cancellation support
* First error encountered stops processing and is returned immediately
* Use `errors.As()` to extract typed errors from error chains
* Use `errors.Is()` to check for specific underlying errors

## 10. Design Principles

1. **Type Safety**: Compile-time type safety for renderer inputs via typed `Source` structs
2. **Modularity**: Each component is independent and self-contained
3. **Flexibility**: Three-level F/T allows precise control over processing
4. **Consistency**: All components follow the same pattern and interface
5. **Extensibility**: Easy to add new renderers, filters, and transformers
6. **Error Handling**: Explicit error handling with wrapped errors for debugging
7. **Context Propagation**: Full support for cancellation and timeouts
8. **Functional Options**: Dual pattern support (function-based and struct-based)
9. **Composability**: Filters and transformers compose naturally
10. **Minimal Dependencies**: Custom implementations to reduce external dependencies

## 11. Dependencies

The Engine depends on:
- **k8s-manifest-kit/pkg**: Shared utilities (caching, merging, JQ, Kubernetes object utilities)
- **k8s.io/apimachinery**: Kubernetes API machinery for `unstructured.Unstructured`
- **k8s.io/api**: Kubernetes API types for GVK filtering

Renderers are **not** part of the engine repository - they are separate modules in the k8s-manifest-kit organization that depend on the engine.

