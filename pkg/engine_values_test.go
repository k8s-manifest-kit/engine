package engine_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	engine "github.com/k8s-manifest-kit/engine/pkg"
	"github.com/k8s-manifest-kit/engine/pkg/render"
	"github.com/k8s-manifest-kit/engine/pkg/types"

	. "github.com/onsi/gomega"
)

// TestRenderRejectsCyclicValues verifies that Engine.Render rejects
// render-time values containing reference cycles with a descriptive error
// before any renderer runs, instead of crashing the process inside the
// values deep clone.
func TestRenderRejectsCyclicValues(t *testing.T) {

	t.Run("should reject a self-referential map value", func(t *testing.T) {
		g := NewWithT(t)

		cyclic := map[string]any{}
		cyclic["self"] = cyclic

		// Process is deliberately left unstubbed: a render with cyclic
		// values must fail before any renderer runs, and an unexpected mock
		// call fails the test.
		renderer := new(mockRenderer)
		renderer.On("Name").Return("mock")

		e, err := engine.New(engine.WithRenderer(renderer))
		g.Expect(err).ToNot(HaveOccurred())

		// Before the fix this exhausted the goroutine stack inside the
		// values deep clone; it must now fail with a descriptive error.
		objects, err := e.Render(t.Context(), render.WithValues(types.Values{
			"cyclic": cyclic,
		}))

		g.Expect(err).To(HaveOccurred())
		g.Expect(errors.Is(err, engine.ErrCyclicValues)).To(BeTrue())
		g.Expect(err.Error()).To(ContainSubstring(`"cyclic.self" refers back to "cyclic"`))
		g.Expect(objects).To(BeNil())
	})

	t.Run("should reject a self-referential slice value", func(t *testing.T) {
		g := NewWithT(t)

		cyclic := make([]any, 1)
		cyclic[0] = cyclic

		renderer := new(mockRenderer)
		renderer.On("Name").Return("mock")

		e, err := engine.New(engine.WithRenderer(renderer))
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := e.Render(t.Context(), render.WithValues(types.Values{
			"cyclic": cyclic,
		}))

		g.Expect(err).To(HaveOccurred())
		g.Expect(errors.Is(err, engine.ErrCyclicValues)).To(BeTrue())
		g.Expect(err.Error()).To(ContainSubstring(`"cyclic[0]" refers back to "cyclic"`))
		g.Expect(objects).To(BeNil())
	})

	t.Run("should reject cyclic values passed as struct-based render options", func(t *testing.T) {
		g := NewWithT(t)

		cyclic := map[string]any{}
		cyclic["self"] = cyclic

		// Struct-based render options deep-clone their values while being
		// applied, so the cycle must be rejected before that clone runs.
		renderer := new(mockRenderer)
		renderer.On("Name").Return("mock")

		e, err := engine.New(engine.WithRenderer(renderer))
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := e.Render(t.Context(), render.Options{
			Values: types.Values{"cyclic": cyclic},
		})

		g.Expect(err).To(HaveOccurred())
		g.Expect(errors.Is(err, engine.ErrCyclicValues)).To(BeTrue())
		g.Expect(err.Error()).To(ContainSubstring(`"cyclic.self" refers back to "cyclic"`))
		g.Expect(objects).To(BeNil())
	})

	t.Run("should reject a cycle nested in maps and slices", func(t *testing.T) {
		g := NewWithT(t)

		cyclic := map[string]any{}
		cyclic["self"] = cyclic

		values := types.Values{
			"items": []any{
				map[string]any{"nested": cyclic},
			},
		}

		renderer := new(mockRenderer)
		renderer.On("Name").Return("mock")

		e, err := engine.New(engine.WithRenderer(renderer))
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := e.Render(t.Context(), render.WithValues(values))

		g.Expect(err).To(HaveOccurred())
		g.Expect(errors.Is(err, engine.ErrCyclicValues)).To(BeTrue())
		g.Expect(err.Error()).To(ContainSubstring(`"items[0].nested.self" refers back to "items[0].nested"`))
		g.Expect(objects).To(BeNil())
	})

	t.Run("should keep the engine usable after rejecting cyclic values", func(t *testing.T) {
		g := NewWithT(t)

		cyclic := map[string]any{}
		cyclic["self"] = cyclic

		renderer := new(mockRenderer)
		renderer.On("Name").Return("mock")

		e, err := engine.New(engine.WithRenderer(renderer))
		g.Expect(err).ToNot(HaveOccurred())

		_, err = e.Render(t.Context(), render.WithValues(types.Values{"cyclic": cyclic}))
		g.Expect(err).To(HaveOccurred())

		// The process survives the rejection and later renders succeed.
		renderer.On("Process", mock.Anything, mock.Anything).Return([]unstructured.Unstructured{makePod("pod1")}, nil)

		objects, err := e.Render(t.Context(), render.WithValues(types.Values{"replicaCount": 3}))
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objectNames(objects)).To(ConsistOf("pod1"))
	})

	t.Run("should render shared references that do not form cycles", func(t *testing.T) {
		g := NewWithT(t)

		// The same nested map appears under two keys, and a slice holds a
		// prefix of itself: both share references without any node being
		// reachable from itself, so the values stay valid.
		shared := map[string]any{"tier": "web"}

		items := make([]any, 2)
		items[0] = shared
		items[1] = items[:1]

		values := types.Values{
			"alpha": shared,
			"beta":  shared,
			"items": items,
		}

		var captured types.Values
		renderer := new(mockRenderer)
		renderer.On("Name").Return("mock")
		renderer.On("Process", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			captured = args.Get(1).(types.Values)
		}).Return([]unstructured.Unstructured{makePod("pod1")}, nil)

		e, err := engine.New(engine.WithRenderer(renderer))
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := e.Render(t.Context(), render.WithValues(values))

		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objectNames(objects)).To(ConsistOf("pod1"))
		g.Expect(captured).To(Equal(values))
	})
}
