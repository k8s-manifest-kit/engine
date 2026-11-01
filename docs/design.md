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
transformers, and post-renderers in that order. Render-time hooks are
additive: they are appended to the engine-level chains for the duration of a
single `Render` call and never mutate the engine's own configuration.

Values are not transported through the render context. The engine calls
`Renderer.Process(ctx, values)` with the values as an explicit argument, so
each renderer decides how its source consumes them; renderers that load
static files may accept values without changing file content.

`pipeline.ApplyPostRenderers` is the helper for applying a post-render chain.

## Render-time values

Two built-in paths clone render-time values, and both are guarded by cycle
validation:

- Struct-based `render.Options` deep-clones its values while being applied,
  so the engine validates those values before that clone runs.
- After all options are applied, the engine validates the merged values and
  only then deep-clones them once per renderer.

The per-renderer clone gives every renderer an independently owned values
tree: a renderer that merges or mutates its copy cannot affect another
renderer's copy, the caller's input, or later renders. `render.WithValues`
does not clone; the values map is only copied at the per-renderer clone, so
callers must not mutate it while a render using it is in flight.

Cycle validation follows exactly the reachability of the deep clone —
nested `map[string]any` and `[]any` — and treats every other type as a leaf.
Shared references that do not form a cycle, such as the same nested map
appearing under two keys, remain valid. A cycle on that traversal is
rejected with `ErrCyclicValues` and a key path before any renderer runs,
instead of exhausting the goroutine stack inside the clone. This is a cycle
check for the built-in option paths, not general graph validation, and it
places no restrictions on value types. Custom `render.Option`
implementations that clone values inside their own `ApplyTo` are outside
these guarantees.

`types.Values.DeepClone` copies JSON-like trees. Non-JSON types such as
pointers and structs with pointer fields are shallow-copied, so isolation
for those types is the caller's responsibility.

## Annotations

Renderers record provenance on the objects they produce through the
annotation constants and setters in `pkg/types` (`AnnotationSourceType`,
`AnnotationSourcePath`, `AnnotationSourceFile`, `AnnotationContentHash`,
`AnnotationRenderOrigin`, `SetContentHash`, `SetRenderOrigin`).
`types.AnnotateObjects` centralizes the mechanical application of those
annotations across a batch of objects:

```go
types.AnnotateObjects(objects,
    types.WithSourceAnnotations(map[string]string{
        types.AnnotationSourceType: "helm",
        types.AnnotationSourcePath: chartPath,
    }),
    types.WithContentHash(true),
    types.WithRenderOrigin(types.RenderOriginLive),
)
```

The helper mutates caller-owned objects in place. Its three phases are
independent and always run in the order source annotations, content hash,
render origin, regardless of option order; each phase completes across the
whole batch before the next starts. Default option values skip their phase
without writing. Hashing covers the complete object at that point in the
phase order and is not idempotent; origin-only calls never rehash.
Package-provided options snapshot their input and are reusable across calls
and goroutines. Renderers keep their own cloning, provenance discovery, and
cache lifecycle and call the helper at their existing annotation points.

## Concurrency

A single engine may serve overlapping `Render` calls: per-request values and
render-time hooks are isolated as described above. The engine does not
synchronize user code, however. Filters, transformers, post-renderers, and
renderer implementations shared across overlapping renders must be safe for
concurrent use, and callers must not mutate engine configuration while
renders are in flight.

## Errors and cancellation

Errors retain the failing renderer or pipeline stage where possible and are
wrapped with context. Context cancellation is propagated to renderer loading,
value resolution, and hook execution. Cyclic render-time values are rejected
with `ErrCyclicValues` as described in [Render-time values](#render-time-values).

## Related documentation

- [`../AGENTS.md`](../AGENTS.md) — repository-specific development rules.
- [`development.md`](development.md) — commands and test conventions.
- [`../../pkg/docs/design.md`](../../pkg/docs/design.md) — shared support packages.
