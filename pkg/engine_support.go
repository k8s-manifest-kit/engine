package engine

import (
	"errors"
	"fmt"

	utilmaps "github.com/k8s-manifest-kit/pkg/util/maps"

	"github.com/k8s-manifest-kit/engine/pkg/render"
	"github.com/k8s-manifest-kit/engine/pkg/types"
)

// ErrCyclicValues is returned when render-time values contain a reference
// cycle among nested maps or slices. The engine deep-clones render-time
// values with a shared helper that recurses without cycle detection, so
// cyclic values are rejected with this error instead of crashing the process.
var ErrCyclicValues = errors.New("cyclic render-time values")

// validateRenderValues returns an error wrapping both ErrCyclicValues and
// the shared maps.ErrCycle when values contain a reference cycle among the
// nested maps and slices the engine deep-clones. Detection is delegated to
// the shared maps.ValidateAcyclic helper, which follows the same
// reachability as the clone — map[string]any and []any — and treats every
// other type as a leaf, so valid values of any shape pass unchanged.
func validateRenderValues(values types.Values) error {
	if len(values) == 0 {
		return nil
	}

	if err := utilmaps.ValidateAcyclic(values); err != nil {
		return fmt.Errorf("%w: %w", ErrCyclicValues, err)
	}

	return nil
}

// renderOptionValues returns the render-time values the given option carries
// into a render, if any. Struct-based render.Options is the only built-in
// option that deep-clones its values while being applied, so the engine
// checks those values before that clone runs.
func renderOptionValues(opt render.Option) types.Values {
	switch o := opt.(type) {
	case render.Options:
		return o.Values
	case *render.Options:
		if o != nil {
			return o.Values
		}

		return nil
	default:
		return nil
	}
}
