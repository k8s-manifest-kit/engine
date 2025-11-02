package engine

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/k8s-manifest-kit/pkg/util/metrics"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/k8s-manifest-kit/engine/pkg/pipeline"
	"github.com/k8s-manifest-kit/engine/pkg/types"
)

// Engine represents the core manifest rendering and processing engine.
type Engine struct {
	options Options
}

// New creates a new Engine with the given options.
func New(opts ...Option) (*Engine, error) {
	options := Options{
		Renderers:    make([]types.Renderer, 0),
		Filters:      make([]types.Filter, 0),
		Transformers: make([]types.Transformer, 0),
	}

	for _, opt := range opts {
		opt.ApplyTo(&options)
	}

	for _, renderer := range options.Renderers {
		if err := types.ValidateRenderer(renderer); err != nil {
			return nil, fmt.Errorf("invalid renderer: %w", err)
		}
	}

	e := Engine{
		options: options,
	}

	return &e, nil
}

// Render processes all inputs associated with the registered renderer configurations
// and returns a consolidated slice of unstructured.Unstructured objects.
//
// The rendering pipeline has three stages for filters and transformers:
//  1. renderer-specific: Each renderer applies its own filters/transformers during Process()
//  2. engine-level: Filters/transformers configured via New() are applied to aggregated results
//  3. render-time: Filters/transformers passed via opts are merged with engine-level ones
//
// Render-time options are additive - they append to engine-level options.
// Render-time values are passed to all renderers and deep merged with Source-level values.
func (e *Engine) Render(ctx context.Context, opts ...RenderOption) ([]unstructured.Unstructured, error) {
	startTime := time.Now()

	// Initialize render options by cloning the engine's options
	renderOpts := RenderOptions{
		Filters:      slices.Clone(e.options.Filters),
		Transformers: slices.Clone(e.options.Transformers),
		Values:       make(map[string]any),
	}

	// Apply render options
	for _, opt := range opts {
		opt.ApplyTo(&renderOpts)
	}

	var allObjects []unstructured.Unstructured
	var err error

	// Process renderers in parallel or sequentially
	if e.options.Parallel {
		allObjects, err = e.renderParallel(ctx, renderOpts.Values)
	} else {
		allObjects, err = e.renderSequential(ctx, renderOpts.Values)
	}

	if err != nil {
		return nil, fmt.Errorf("rendering failed: %w", err)
	}

	// Apply filters
	filtered, err := pipeline.ApplyFilters(ctx, allObjects, renderOpts.Filters)
	if err != nil {
		return nil, fmt.Errorf("engine filter error: %w", err)
	}

	// Apply transformers
	transformed, err := pipeline.ApplyTransformers(ctx, filtered, renderOpts.Transformers)
	if err != nil {
		return nil, fmt.Errorf("engine transformer error: %w", err)
	}

	metrics.ObserveRender(ctx, time.Since(startTime), len(transformed))

	return transformed, nil
}

// processRenderer executes a single renderer with timing, metrics, and error handling.
func (e *Engine) processRenderer(
	ctx context.Context,
	renderer types.Renderer,
	values map[string]any,
) ([]unstructured.Unstructured, error) {
	startTime := time.Now()
	objects, err := renderer.Process(ctx, values)

	metrics.ObserveRenderer(ctx, renderer.Name(), time.Since(startTime), len(objects), err)

	if err != nil {
		return nil, fmt.Errorf(
			"error processing renderer %q (%T): %w",
			renderer.Name(),
			renderer,
			err,
		)
	}

	return objects, nil
}

// renderSequential processes renderers sequentially in order.
func (e *Engine) renderSequential(ctx context.Context, values map[string]any) ([]unstructured.Unstructured, error) {
	allObjects := make([]unstructured.Unstructured, 0)

	for _, renderer := range e.options.Renderers {
		objects, err := e.processRenderer(ctx, renderer, values)
		if err != nil {
			return nil, err
		}

		allObjects = append(allObjects, objects...)
	}

	return allObjects, nil
}

// renderParallel processes all renderers concurrently using goroutines.
// Results are collected in the original renderer order for consistent output.
func (e *Engine) renderParallel(ctx context.Context, values map[string]any) ([]unstructured.Unstructured, error) {
	type result struct {
		objects []unstructured.Unstructured
		err     error
	}

	results := make([]result, len(e.options.Renderers))
	var wg sync.WaitGroup

	for i, renderer := range e.options.Renderers {
		wg.Add(1)
		go func(idx int, r types.Renderer) {
			defer wg.Done()
			objects, err := e.processRenderer(ctx, r, values)
			results[idx] = result{
				objects: objects,
				err:     err,
			}
		}(i, renderer)
	}

	wg.Wait()

	// Collect results in original renderer order
	allObjects := make([]unstructured.Unstructured, 0)
	for _, res := range results {
		if res.err != nil {
			return nil, res.err
		}

		allObjects = append(allObjects, res.objects...)
	}

	return allObjects, nil
}
