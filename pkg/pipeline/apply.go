package pipeline

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/k8s-manifest-kit/engine/pkg/types"
)

// ApplySourceSelectors evaluates all source selectors for a given source.
// Returns true if the source should be rendered (all selectors pass).
// Returns false if any selector rejects the source.
// Returns an error if any selector fails.
//
// S is the concrete source type for the renderer (e.g., helm.Source, mem.Source).
// The type parameter is inferred from the call site arguments.
func ApplySourceSelectors[S any](
	ctx context.Context,
	source S,
	selectors []func(context.Context, S) (bool, error),
) (bool, error) {
	for _, selector := range selectors {
		ok, err := selector(ctx, source)
		if err != nil {
			return false, fmt.Errorf("source selector error: %w", err)
		}

		if !ok {
			return false, nil
		}
	}

	return true, nil
}

// ApplyPostRenderers chains post-renderers sequentially, feeding each post-renderer's
// output into the next. Returns the final batch of objects.
//
// A failing post-renderer aborts the chain and returns an error of the form
// "post-renderer error at chain index %d: %w", where the index is the
// zero-based position in the supplied local chain. Chains built by
// types.BuildPostRendererChain compose adapted filters first, then
// transformers, then explicit post-renderers, so the index identifies the
// position in that composed order. The index is scoped to the call: separate
// ApplyPostRenderers calls index their own chains independently, and
// positions are not stable callback identifiers.
func ApplyPostRenderers(
	ctx context.Context,
	objects []unstructured.Unstructured,
	postRenderers []types.PostRenderer,
) ([]unstructured.Unstructured, error) {
	if len(postRenderers) == 0 {
		return objects, nil
	}

	result := objects
	for i, pr := range postRenderers {
		var err error

		result, err = pr(ctx, result)
		if err != nil {
			return nil, fmt.Errorf("post-renderer error at chain index %d: %w", i, err)
		}
	}

	return result, nil
}
