# Engine Design

## Purpose

The engine coordinates one or more manifest renderers and applies shared
pipeline hooks. Renderers return Kubernetes
`unstructured.Unstructured` objects; the engine aggregates those objects and
returns them to the caller.

## Public API

`engine.New(opts ...Option)` constructs an engine. Engine options configure a
renderer, filters, transformers, and post-renderers. A render call accepts
render options such as `render.WithValues`, `render.WithFilter`,
`render.WithTransformer`, and `render.WithPostRenderer`.

The shared value type is `types.Values`. A post-renderer has the shape:

```go
type PostRenderer func(
    context.Context,
    []unstructured.Unstructured,
) ([]unstructured.Unstructured, error)
```

## Pipeline

Each renderer performs its renderer-specific work, applies source selectors,
and can run source-specific post-renderers. The engine then combines the
renderer outputs and applies engine-level and render-time filters,
transformers, and post-renderers in that order.

Render values are passed through the render context. A renderer decides how
its source consumes those values; renderers that load static files may accept
values without changing file content.

`pipeline.ApplyPostRenderers` is the current helper for applying a post-render
chain. The older `Apply`, `ApplyFilters`, and `ApplyTransformers` helpers are
deprecated compatibility APIs.

## Errors and cancellation

Errors retain the failing renderer or pipeline stage where possible and are
wrapped with context. Context cancellation is propagated to renderer loading,
value resolution, and hook execution.

## Related documentation

- [`../AGENTS.md`](../AGENTS.md) — repository-specific development rules.
- [`development.md`](development.md) — commands and test conventions.
- [`../../pkg/docs/design.md`](../../pkg/docs/design.md) — shared support packages.
