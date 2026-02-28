package source

import (
	"context"

	"github.com/k8s-manifest-kit/engine/pkg/types"
)

// Selector creates a type-safe SourceSelector for a specific source type S.
// The selector applies only to sources of type S; all other source types are
// intentionally unaffected (returns true, nil). This allows composing multiple
// type-specific selectors across renderers without cross-type interference.
//
// If S is an interface type, the selector matches any source implementing that
// interface — not just a single concrete type. Use concrete types for precise
// matching.
func Selector[S any](fn func(ctx context.Context, source S) (bool, error)) types.SourceSelector {
	return func(ctx context.Context, s types.Source) (bool, error) {
		src, ok := s.(S)
		if !ok {
			return true, nil
		}

		return fn(ctx, src)
	}
}
