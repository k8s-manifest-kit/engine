package engine

import (
	"maps"

	"github.com/k8s-manifest-kit/pkg/util"

	"github.com/k8s-manifest-kit/engine/pkg/types"
)

// RenderOptions represents the processing options for rendering.
type RenderOptions struct {
	// Filters are render-time filters applied only to this specific Render() call.
	// These are merged with (appended to) engine-level filters.
	Filters []types.Filter

	// Transformers are render-time transformers applied only to this specific Render() call.
	// These are merged with (appended to) engine-level transformers.
	Transformers []types.Transformer

	// Values are render-time values passed to all renderers during this specific Render() call.
	// These values are deep merged with Source-level values, with render-time values taking precedence.
	Values map[string]any
}

// ApplyTo implements the Option interface for RenderOptions.
func (opts RenderOptions) ApplyTo(target *RenderOptions) {
	target.Filters = append(target.Filters, opts.Filters...)
	target.Transformers = append(target.Transformers, opts.Transformers...)

	if opts.Values != nil {
		target.Values = maps.Clone(opts.Values)
	}
}

// Options represents the processing options for the engine.
type Options struct {
	// Filters are engine-level filters applied to all renders.
	Filters []types.Filter

	// Transformers are engine-level transformers applied to all renders.
	Transformers []types.Transformer

	// Values are values passed to renderers (used internally during rendering).
	Values map[string]any

	// Renderers are the manifest sources to process (e.g., Helm, Kustomize, YAML).
	Renderers []types.Renderer

	// Parallel enables parallel execution of renderers.
	Parallel bool
}

// ApplyTo implements the Option interface for Options.
func (opts Options) ApplyTo(target *Options) {
	target.Renderers = append(target.Renderers, opts.Renderers...)
	target.Filters = append(target.Filters, opts.Filters...)
	target.Transformers = append(target.Transformers, opts.Transformers...)
	target.Parallel = opts.Parallel

	if opts.Values != nil {
		target.Values = maps.Clone(opts.Values)
	}
}

// Option is a generic option for Options.
type Option = util.Option[Options]

// RenderOption is a generic option for RenderOptions.
type RenderOption = util.Option[RenderOptions]

// WithRenderer adds a configured renderer to the engine.
// Can only be used during engine creation.
func WithRenderer(r types.Renderer) Option {
	return util.FunctionalOption[Options](func(o *Options) {
		o.Renderers = append(o.Renderers, r)
	})
}

// WithFilter adds an engine-level filter function to the processing chain.
// Engine-level filters are applied to aggregated results from all renderers on every Render() call.
// For renderer-specific filtering, use the renderer's WithFilter option (e.g., helm.WithFilter).
// For one-time filtering on a single Render() call, use WithRenderFilter.
func WithFilter(f types.Filter) Option {
	return util.FunctionalOption[Options](func(o *Options) {
		o.Filters = append(o.Filters, f)
	})
}

// WithTransformer adds an engine-level transformer function to the processing chain.
// Engine-level transformers are applied to aggregated results from all renderers on every Render() call.
// For renderer-specific transformation, use the renderer's WithTransformer option (e.g., helm.WithTransformer).
// For one-time transformation on a single Render() call, use WithRenderTransformer.
func WithTransformer(t types.Transformer) Option {
	return util.FunctionalOption[Options](func(o *Options) {
		o.Transformers = append(o.Transformers, t)
	})
}

// WithRenderFilter adds a render-time filter function for a single Render() call.
// Render-time filters are merged with (appended to) engine-level filters.
// Use this for one-off filtering that doesn't apply to all renders.
func WithRenderFilter(f types.Filter) RenderOption {
	return util.FunctionalOption[RenderOptions](func(o *RenderOptions) {
		o.Filters = append(o.Filters, f)
	})
}

// WithRenderTransformer adds a render-time transformer function for a single Render() call.
// Render-time transformers are merged with (appended to) engine-level transformers.
// Use this for one-off transformation that doesn't apply to all renders.
func WithRenderTransformer(t types.Transformer) RenderOption {
	return util.FunctionalOption[RenderOptions](func(o *RenderOptions) {
		o.Transformers = append(o.Transformers, t)
	})
}

// WithParallel enables or disables parallel execution of renderers.
// When enabled, all renderers execute concurrently using goroutines.
// When disabled (default), renderers execute sequentially.
// Parallel execution is beneficial for I/O-bound renderers (Helm OCI fetch, file reads).
func WithParallel(enabled bool) Option {
	return util.FunctionalOption[Options](func(o *Options) {
		o.Parallel = enabled
	})
}

// WithValues adds render-time values for a single Render() call.
// These values are passed to all renderers and deep merged with Source-level values,
// with render-time values taking precedence for conflicting keys.
// Renderers that support dynamic values (Helm, Kustomize, GoTemplate) will use these values.
// Renderers that don't support values (YAML, Mem) will ignore them.
func WithValues(values map[string]any) RenderOption {
	return util.FunctionalOption[RenderOptions](func(o *RenderOptions) {
		o.Values = values
	})
}
