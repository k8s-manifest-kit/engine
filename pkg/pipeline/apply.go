package pipeline

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/k8s-manifest-kit/engine/pkg/filter"
	"github.com/k8s-manifest-kit/engine/pkg/transformer"
	"github.com/k8s-manifest-kit/engine/pkg/types"
)

// ApplyFilters applies a series of filters to objects, returning only those that match all filters.
// Returns Error with detailed context if any filter fails.
func ApplyFilters(
	ctx context.Context,
	objects []unstructured.Unstructured,
	filters []types.Filter,
) ([]unstructured.Unstructured, error) {
	if len(filters) == 0 {
		return objects, nil
	}

	filtered := make([]unstructured.Unstructured, 0, len(objects))

	for _, obj := range objects {
		matches := true
		for _, f := range filters {
			ok, err := f(ctx, obj)
			if err != nil {
				// filter.Wrap already returns a typed Error
				return nil, filter.Wrap(obj, err)
			}
			if !ok {
				matches = false

				break
			}
		}

		if matches {
			filtered = append(filtered, obj)
		}
	}

	return filtered, nil
}

// ApplyTransformers applies a series of transformers to objects, transforming each object sequentially.
// Returns Error with detailed context if any transformer fails.
func ApplyTransformers(
	ctx context.Context,
	objects []unstructured.Unstructured,
	transformers []types.Transformer,
) ([]unstructured.Unstructured, error) {
	if len(transformers) == 0 {
		return objects, nil
	}

	transformed := make([]unstructured.Unstructured, 0, len(objects))

	for _, obj := range objects {
		result := obj
		for _, t := range transformers {
			r, err := t(ctx, result)
			if err != nil {
				// transformer.Wrap already returns a typed Error
				return nil, transformer.Wrap(obj, err)
			}
			result = r
		}

		transformed = append(transformed, result)
	}

	return transformed, nil
}

// Apply executes a filter and transformer pipeline on the given objects.
// It applies filters first, then transformers, returning the transformed objects.
// Callers should wrap returned errors with appropriate context.
func Apply(
	ctx context.Context,
	objects []unstructured.Unstructured,
	filters []types.Filter,
	transformers []types.Transformer,
) ([]unstructured.Unstructured, error) {
	// Apply filters
	filtered, err := ApplyFilters(ctx, objects, filters)
	if err != nil {
		return nil, fmt.Errorf("filter error: %w", err)
	}

	// Apply transformers
	transformed, err := ApplyTransformers(ctx, filtered, transformers)
	if err != nil {
		return nil, fmt.Errorf("transformer error: %w", err)
	}

	return transformed, nil
}
