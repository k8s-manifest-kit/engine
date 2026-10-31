# Engine Development

## Prerequisites

Use the Go version declared in this module's [`go.mod`](../go.mod). The
Makefile owns the commands and pins the tool versions, such as the
golangci-lint release; it is the source of truth for both.

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

These are separate targets. `make test` runs the unit suite, `make
test/race` re-runs it under the race detector, and `make fmt` formats the
code. `make check` runs lint and vulncheck only; it is not a substitute for
the test targets and does not verify formatting.

## Package layout

- `engine.go` — engine construction and render orchestration.
- `engine_support.go` — shared implementation helpers for the engine package.
- `render/` — render-time options and pipeline configuration.
- `pipeline/` — shared filter, transformer, and post-renderer application.
- `types/` — renderer, values, hook, and annotation contracts.
- `filter/`, `transformer/`, and `postrenderer/` — built-in hook utilities.

Keep public behavior changes synchronized with the renderer modules and the
examples module. Prefer the shared types and pipeline helpers over introducing
renderer-specific copies of common contracts.

## Style

- Keep package responsibilities and ownership of mutable state explicit.
  Introduce an abstraction only when it names a real responsibility or reuse,
  not to erase similar-looking renderer code.
- Give each function parameter its own type: `name string, namespace string`,
  not `name, namespace string`.
- Put shared implementation helpers in `_support.go` and test-only support in
  `_support_test.go`; keep cohesive feature files intact rather than
  mechanically extracting every small helper.
- Follow the formatter and import grouping configured in `.golangci.yml`.
  Tests live in external test packages (for example `package engine_test`)
  and exercise the public API.
- Wrap errors with operation and source context using `%w` where a cause
  exists, and check sentinel errors such as `ErrCyclicValues` with
  `errors.Is`. Validation errors should name the path, key, or type needed
  for diagnosis without echoing input contents.

### Multiline strings

- Use one descriptive, standalone, package-level `const name =` declaration
  with a raw string literal for every multiline string: fixtures, manifests,
  templates, and expected output.
- Never embed multiline string literals in function or method bodies, and
  never declare them as function-local constants.
- Never group multiline constants in `const (...)`. Related single-line
  scalar constants can still be grouped, and single-line inline strings are
  unaffected.
- Keep large literal content separate from execution logic for readability.
  Preserve meaningful content indentation; the declaration layout must not
  introduce unintended bytes.
- A function may format the named template into a local variable. The
  prohibition concerns where the literal is declared, not where its string
  value is used.

```go
const fixtureYAML = `apiVersion: v1
kind: ConfigMap
`
```

## Testing

Tests use Gomega for assertions and `t.Context()` for contexts. Cover option
ordering, context cancellation, error wrapping, empty renderer sets, and the
order of filters, transformers, and post-renderers when changing the
pipeline.

For behavior fixes, add a regression test that fails without the fix and
asserts on fields and identities rather than only error counts. Existing
anchors include `TestRenderRejectsCyclicValues` in
`pkg/engine_values_test.go`, `TestRenderTimeValues` in `pkg/engine_test.go`,
and `TestRenderConcurrentRequests` in `pkg/engine_concurrency_test.go`.

Concurrent shared-instance tests start workers behind a barrier, join all
workers, and assert from the test goroutine. Give each worker independently
owned inputs and result slots, make callbacks used across goroutines
thread-safe, and do not use sleeps to order workers or unprotected shared
counters. Cover both a successful cold render and warm concurrent renders.

The engine does not synchronize user code: filters, transformers,
post-renderers, and custom `types.Renderer` implementations used by
overlapping `Render` calls must be safe for concurrent use, and callers must
not mutate engine configuration while renders are in flight. The default test
suite runs offline; this repository has no live integration targets.

`make test/race` re-runs the unit suite under the race detector. The
concurrency tests render overlapping requests against a single engine, so
run it whenever changing value cloning, render-time option merging, or
output aggregation.

See [`design.md`](design.md) and [`../AGENTS.md`](../AGENTS.md).
