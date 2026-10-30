package engine

import (
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/k8s-manifest-kit/engine/pkg/render"
	"github.com/k8s-manifest-kit/engine/pkg/types"
)

// ErrCyclicValues is returned when render-time values contain a reference
// cycle among nested maps or slices. The engine deep-clones render-time
// values with a shared helper that recurses without cycle detection, so
// cyclic values are rejected with this error instead of crashing the process.
var ErrCyclicValues = errors.New("cyclic render-time values")

// valuesRootPath names the top-level values map in reported key paths.
const valuesRootPath = "<root>"

// validateRenderValues returns an ErrCyclicValues error when values contain
// a reference cycle among the nested maps and slices the engine deep-clones.
// It follows the same reachability as the clone — map[string]any and []any —
// and treats every other type as a leaf, so valid values of any shape pass
// unchanged.
func validateRenderValues(values types.Values) error {
	if len(values) == 0 {
		return nil
	}

	checker := valuesCycleChecker{}

	return checker.checkMap(values)
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

// valuesNode identifies a map or slice during cycle detection: maps by their
// header pointer, slices by their data pointer and length. Two slices
// sharing a backing array but differing in length stay distinct, which
// matches the slice values the deep clone recurses into.
type valuesNode struct {
	pointer uintptr
	length  int
}

// valuesPathSegment is one segment of a values key path: a map key when
// isKey is set, a slice element index otherwise.
type valuesPathSegment struct {
	key   string
	index int
	isKey bool
}

// valuesCycleChecker detects reference cycles in render-time values.
type valuesCycleChecker struct {
	// ancestors holds the identity of every map and slice on the current
	// descent path, oldest first. The deep clone recurses forever exactly
	// when some node is reachable from itself, which makes it appear on
	// this path a second time.
	ancestors []valuesNode

	// segments holds the key path of the current descent path, oldest
	// first. It is only rendered into a string when a cycle is reported.
	segments []valuesPathSegment
}

// checkMap reports an error when a reference cycle is reachable through the
// given map. Keys are visited in sorted order so the reported cycle does not
// depend on map iteration order.
func (c *valuesCycleChecker) checkMap(m map[string]any) error {
	node := valuesNode{pointer: reflect.ValueOf(m).Pointer(), length: len(m)}
	if slices.Contains(c.ancestors, node) {
		return c.cycleError(node)
	}

	c.ancestors = append(c.ancestors, node)

	for _, key := range slices.Sorted(maps.Keys(m)) {
		c.segments = append(c.segments, valuesPathSegment{key: key, isKey: true})

		if err := c.checkValue(m[key]); err != nil {
			return err
		}

		c.segments = c.segments[:len(c.segments)-1]
	}

	c.ancestors = c.ancestors[:len(c.ancestors)-1]

	return nil
}

// checkSlice reports an error when a reference cycle is reachable through
// the given slice.
func (c *valuesCycleChecker) checkSlice(s []any) error {
	node := valuesNode{pointer: reflect.ValueOf(s).Pointer(), length: len(s)}
	if slices.Contains(c.ancestors, node) {
		return c.cycleError(node)
	}

	c.ancestors = append(c.ancestors, node)

	for i, element := range s {
		c.segments = append(c.segments, valuesPathSegment{index: i})

		if err := c.checkValue(element); err != nil {
			return err
		}

		c.segments = c.segments[:len(c.segments)-1]
	}

	c.ancestors = c.ancestors[:len(c.ancestors)-1]

	return nil
}

// checkValue reports an error when a reference cycle is reachable through
// the given value. Only map[string]any and []any are entered, matching the
// reachability of the engine's deep clone; every other type is a leaf.
func (c *valuesCycleChecker) checkValue(value any) error {
	switch v := value.(type) {
	case map[string]any:
		return c.checkMap(v)
	case []any:
		return c.checkSlice(v)
	default:
		return nil
	}
}

// cycleError describes the cycle closed by the given ancestor node: where
// the closing reference lives and where it points back to.
func (c *valuesCycleChecker) cycleError(node valuesNode) error {
	ancestor := slices.Index(c.ancestors, node)

	return fmt.Errorf(
		"%w: %q refers back to %q",
		ErrCyclicValues,
		c.path(len(c.segments)),
		c.path(ancestor),
	)
}

// path renders the first depth segments of the current descent path as a
// dotted key path with bracketed slice indexes. It runs only when a cycle is
// reported.
func (c *valuesCycleChecker) path(depth int) string {
	if depth == 0 {
		return valuesRootPath
	}

	parts := make([]string, 0, depth)

	for i, segment := range c.segments[:depth] {
		switch {
		case !segment.isKey:
			parts = append(parts, "["+strconv.Itoa(segment.index)+"]")
		case i > 0:
			parts = append(parts, "."+segment.key)
		default:
			parts = append(parts, segment.key)
		}
	}

	return strings.Join(parts, "")
}
