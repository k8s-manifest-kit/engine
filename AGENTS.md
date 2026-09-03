# Agent Guide: engine

`engine` is the core orchestration module for k8s-manifest-kit. It combines heterogeneous renderers, passes render-time values to them, and applies filters, transformers, and post-renderers to the aggregated result.

## Documentation

- [README](README.md) — module overview and quick start.
- [Design](docs/design.md) — architecture and pipeline semantics.
- [Development](docs/development.md) — local workflow and testing conventions.

## Public API

The package is imported from `github.com/k8s-manifest-kit/engine/pkg`.

- `engine.New(engine.WithRenderer(renderer), ...)` creates an engine and validates renderer names.
- Engine options: `WithRenderer`, `WithFilter`, `WithTransformer`, and `WithPostRenderer`.
- `Engine.Render(ctx, ...)` accepts render-time options from `github.com/k8s-manifest-kit/engine/pkg/render`: `WithValues`, `WithFilter`, `WithTransformer`, and `WithPostRenderer`.
- `types.Renderer.Process` receives `types.Values` and must return a renderer name through `Name()`.
- `types.PostRenderer` receives the whole object batch and may validate, modify, reorder, or enrich it.

Render-time filters, transformers, and post-renderers are appended to the corresponding engine-level chains. Render-time values are deep-cloned for each renderer; renderers that support values merge them over source values.

## Pipeline

Each renderer owns its source-selection and renderer-level processing. The engine then:

1. Processes every registered renderer.
2. Aggregates the returned objects.
3. Applies engine-level and render-time filters.
4. Applies engine-level and render-time transformers.
5. Applies engine-level and render-time post-renderers.

Use `types.BuildPostRendererChain` and `pipeline.ApplyPostRenderers` when working directly with the pipeline package. The older `Apply`, `ApplyFilters`, and `ApplyTransformers` helpers are deprecated.

## Built-in packages

- `filter`: `Or`, `And`, `Not`, `If`, JQ, and metadata filters.
- `transformer`: `Chain`, `If`, `Switch`, JQ, and metadata transformers.
- `postrenderer`: batch operations such as `ApplyOrder`.
- `types`: renderer, value, filter, transformer, post-renderer, and annotation types.

## Development

Run commands from this directory:

```bash
make test
make fmt
make lint
make lint/fix
make check
```

Use `t.Context()` in tests, Gomega for assertions, and preserve error chains with `%w`. Add or update tests and the relevant documentation when changing pipeline behavior.

