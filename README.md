# Engine

`engine` coordinates manifest rendering across one or more renderer
implementations. It owns the shared render pipeline and provides the common
filter, transformer, and post-renderer hooks used by the renderer modules.

## Installation

```bash
go get github.com/k8s-manifest-kit/engine
```

## Quick start

```go
e, err := engine.New(
    engine.WithRenderer(myRenderer),
    engine.WithPostRenderer(myPostRenderer),
)
if err != nil {
    return err
}

objects, err := e.Render(ctx, render.WithValues(types.Values{
    "environment": "dev",
}))
```

Render-time options can add values, filters, transformers, and post-renderers.
The engine applies the configured pipeline in a deterministic order and
returns Kubernetes `unstructured.Unstructured` objects.

See [`docs/design.md`](docs/design.md), [`docs/development.md`](docs/development.md),
and [`AGENTS.md`](AGENTS.md) for architecture and development guidance.

## License

Apache License 2.0. See [LICENSE](LICENSE).
