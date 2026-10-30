# Engine Development

## Prerequisites

Use Go 1.26.8, as declared by the module and workspace configuration.

## Commands

Run commands from this repository directory:

```bash
make test
make test/race
make fmt
make lint
make lint/fix
make check
```

`make check` combines formatting, linting, and tests. The Makefile is the
source of truth for tool versions and command details.

## Package layout

- `engine.go` — engine construction and render orchestration.
- `render/` — render-time options and pipeline configuration.
- `pipeline/` — shared filter, transformer, and post-renderer application.
- `types/` — renderer, values, hook, and annotation contracts.
- `filter/`, `transformer/`, and `postrenderer/` — built-in hook utilities.

Keep public behavior changes synchronized with the renderer modules and the
examples module. Prefer the shared types and pipeline helpers over introducing
renderer-specific copies of common contracts.

## Testing

Tests use the standard Go test runner and Gomega where appropriate. Cover
option ordering, context cancellation, error wrapping, empty renderer sets,
and the order of filters, transformers, and post-renderers when changing the
pipeline.

`make test/race` re-runs the unit suite under the race detector. The
concurrency tests render overlapping requests against a single engine, so
run it whenever changing value cloning, render-time option merging, or
output aggregation.

See [`design.md`](design.md) and [`../AGENTS.md`](../AGENTS.md).
