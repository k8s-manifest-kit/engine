package engine_test

import (
	"errors"
	"testing"

	utilmaps "github.com/k8s-manifest-kit/pkg/util/maps"
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
		g.Expect(errors.Is(err, utilmaps.ErrCycle)).To(BeTrue())
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
		g.Expect(errors.Is(err, utilmaps.ErrCycle)).To(BeTrue())
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
		g.Expect(errors.Is(err, utilmaps.ErrCycle)).To(BeTrue())
		g.Expect(err.Error()).To(ContainSubstring(`"cyclic.self" refers back to "cyclic"`))
		g.Expect(objects).To(BeNil())
	})

	t.Run("should reject cyclic values passed as pointer-based render options", func(t *testing.T) {
		g := NewWithT(t)

		cyclic := map[string]any{}
		cyclic["self"] = cyclic

		// Pointer-based render options take the same apply path as the
		// struct form and deep-clone their values while being applied, so
		// the cycle must be rejected before that clone runs too.
		renderer := new(mockRenderer)
		renderer.On("Name").Return("mock")

		e, err := engine.New(engine.WithRenderer(renderer))
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := e.Render(t.Context(), &render.Options{
			Values: types.Values{"cyclic": cyclic},
		})

		g.Expect(err).To(HaveOccurred())
		g.Expect(errors.Is(err, engine.ErrCyclicValues)).To(BeTrue())
		g.Expect(errors.Is(err, utilmaps.ErrCycle)).To(BeTrue())
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
		g.Expect(errors.Is(err, utilmaps.ErrCycle)).To(BeTrue())
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

func TestRenderTimeValues(t *testing.T) {

	t.Run("should pass render-time values to renderer", func(t *testing.T) {
		g := NewWithT(t)
		var capturedValues types.Values
		renderer := new(mockRenderer)
		renderer.On("Name").Return("mock")
		renderer.On("Process", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			capturedValues = args.Get(1).(types.Values)
		}).Return([]unstructured.Unstructured{makePod("test-pod")}, nil)

		e, err := engine.New(engine.WithRenderer(renderer))
		g.Expect(err).ToNot(HaveOccurred())

		renderValues := types.Values{
			"replicaCount": 3,
			"image": map[string]any{
				"tag": "v2.0",
			},
		}

		objects, err := e.Render(t.Context(), render.WithValues(renderValues))

		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(objects).Should(HaveLen(1))
		g.Expect(capturedValues).Should(Equal(renderValues))
	})

	t.Run("should pass empty map when no values provided", func(t *testing.T) {
		g := NewWithT(t)
		var capturedValues types.Values
		renderer := new(mockRenderer)
		renderer.On("Name").Return("mock")
		renderer.On("Process", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			capturedValues = args.Get(1).(types.Values)
		}).Return([]unstructured.Unstructured{makePod("test-pod")}, nil)

		e, err := engine.New(engine.WithRenderer(renderer))
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := e.Render(t.Context())

		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(objects).Should(HaveLen(1))
		g.Expect(capturedValues).Should(BeEmpty())
	})

	t.Run("should pass same values to multiple renderers", func(t *testing.T) {
		g := NewWithT(t)
		var capturedValues1, capturedValues2 types.Values

		renderer1 := new(mockRenderer)
		renderer1.On("Name").Return("renderer1")
		renderer1.On("Process", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			capturedValues1 = args.Get(1).(types.Values)
		}).Return([]unstructured.Unstructured{makePod("pod1")}, nil)

		renderer2 := new(mockRenderer)
		renderer2.On("Name").Return("renderer2")
		renderer2.On("Process", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			capturedValues2 = args.Get(1).(types.Values)
		}).Return([]unstructured.Unstructured{makePod("pod2")}, nil)

		e, err := engine.New(
			engine.WithRenderer(renderer1),
			engine.WithRenderer(renderer2),
		)
		g.Expect(err).ToNot(HaveOccurred())

		renderValues := types.Values{
			"env": "production",
		}

		objects, err := e.Render(t.Context(), render.WithValues(renderValues))

		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(objects).Should(HaveLen(2))
		g.Expect(capturedValues1).Should(Equal(renderValues))
		g.Expect(capturedValues2).Should(Equal(renderValues))
	})

	t.Run("should work with struct-based render.Options", func(t *testing.T) {
		g := NewWithT(t)
		var capturedValues types.Values
		renderer := new(mockRenderer)
		renderer.On("Name").Return("mock")
		renderer.On("Process", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			capturedValues = args.Get(1).(types.Values)
		}).Return([]unstructured.Unstructured{makePod("test-pod")}, nil)

		e, err := engine.New(engine.WithRenderer(renderer))
		g.Expect(err).ToNot(HaveOccurred())

		renderValues := types.Values{
			"key": "value",
		}

		objects, err := e.Render(t.Context(), render.Options{
			Values: renderValues,
		})

		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(objects).Should(HaveLen(1))
		g.Expect(capturedValues).Should(Equal(renderValues))
	})

}
