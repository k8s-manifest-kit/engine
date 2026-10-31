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
- Put shared implementation helpers in `_support.go`. In test packages, place
  shared helpers and fixtures in `<component>_support_test.go`; small
  test-local helpers and scenario assertions stay beside their tests rather
  than being mechanically extracted (see [Test layout](#test-layout)).
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

### Test layout

- Colocate tests with implementation. A `test/` subdirectory is a separate
  Go package, not a cosmetic folder; use existing package boundaries.
- Organize larger suites by concern using `<component>_<concern>_test.go`;
  retain a cohesive `<component>_test.go` for core API behavior. Use
  `_metadata_test.go`, `_cache_test.go`, `_values_test.go`, and
  `_concurrency_test.go` where relevant, not as mandatory files for every
  package.
- Place benchmarks in `<component>_benchmark_test.go`, in the same package
  directory, retaining benchmark names and setup/timing behavior.
- Put shared helpers in `<component>_support_test.go` and substantial
  feature-specific fixture machinery in
  `<component>_<concern>_support_test.go`. Small test-local helpers and
  scenario assertions can remain beside their tests.
- Support files contain helper/fixture/type declarations, not test,
  benchmark, fuzz, or example entry points. `_support_test.go` is a naming
  convention; only `_test.go` has Go's test-file significance.
- Keep tests discoverable without excessive fragmentation: no hard line
  limit, mandatory one-file-per-test rule, shared cross-repository harness,
  fixture-storage migration, or automatic test-speed claim.
- Preserve build tags where tests use them; filenames alone do not exclude
  a test from the default suite, and this repository has no live
  integration targets.

For behavior fixes, add a regression test that fails without the fix and
asserts on fields and identities rather than only error counts. Existing
anchors include `TestRenderRejectsCyclicValues` in
`pkg/engine_values_test.go`, `TestRenderTimeValues` in
`pkg/engine_values_test.go`, and `TestRenderConcurrentRequests` in
`pkg/engine_concurrency_test.go`.

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
