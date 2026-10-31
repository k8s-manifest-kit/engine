package engine_test

import (
	"context"
	"errors"
	"testing"

	"github.com/k8s-manifest-kit/pkg/util/k8s"
	"github.com/stretchr/testify/mock"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	engine "github.com/k8s-manifest-kit/engine/pkg"
	"github.com/k8s-manifest-kit/engine/pkg/render"
	"github.com/k8s-manifest-kit/engine/pkg/types"

	. "github.com/onsi/gomega"
)

func TestNew(t *testing.T) {

	t.Run("should create empty engine", func(t *testing.T) {
		g := NewWithT(t)
		e, err := engine.New()
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(e).ToNot(BeNil())
	})

	t.Run("should create engine with renderer", func(t *testing.T) {
		g := NewWithT(t)
		renderer := new(mockRenderer)
		renderer.On("Process", mock.Anything, mock.Anything).Return([]unstructured.Unstructured{makePod("test-pod")}, nil)
		renderer.On("Name").Return("mock")

		e, err := engine.New(engine.WithRenderer(renderer))
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(e).ToNot(BeNil())
	})

	t.Run("should create engine with filter", func(t *testing.T) {
		g := NewWithT(t)
		filter := podFilter()
		e, err := engine.New(engine.WithFilter(filter))
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(e).ToNot(BeNil())
	})

	t.Run("should create engine with transformer", func(t *testing.T) {
		g := NewWithT(t)
		transformer := addLabels(map[string]string{"test": "value"})
		e, err := engine.New(engine.WithTransformer(transformer))
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(e).ToNot(BeNil())
	})
}

func TestEngineRender(t *testing.T) {

	t.Run("should render with single renderer", func(t *testing.T) {
		g := NewWithT(t)
		renderer := new(mockRenderer)
		renderer.On("Process", mock.Anything, mock.Anything).Return([]unstructured.Unstructured{
			makePod("pod1"),
			makePod("pod2"),
		}, nil)
		renderer.On("Name").Return("mock")

		e, err := engine.New(engine.WithRenderer(renderer))
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := e.Render(t.Context())
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(objects).To(HaveLen(2))
		g.Expect(objects[0].GetName()).To(Equal("pod1"))
		g.Expect(objects[1].GetName()).To(Equal("pod2"))
	})

	t.Run("should render with multiple renderers", func(t *testing.T) {
		g := NewWithT(t)
		renderer1 := new(mockRenderer)
		renderer1.On("Process", mock.Anything, mock.Anything).Return([]unstructured.Unstructured{makePod("pod1")}, nil)
		renderer1.On("Name").Return("mock")
		renderer2 := new(mockRenderer)
		renderer2.On("Process", mock.Anything, mock.Anything).Return([]unstructured.Unstructured{makePod("pod2")}, nil)
		renderer2.On("Name").Return("mock")

		e, err := engine.New(
			engine.WithRenderer(renderer1),
			engine.WithRenderer(renderer2),
		)
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := e.Render(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(2))
	})

	t.Run("should apply engine-level filter", func(t *testing.T) {
		g := NewWithT(t)
		renderer := new(mockRenderer)
		renderer.On("Process", mock.Anything, mock.Anything).Return([]unstructured.Unstructured{
			makePod("pod1"),
			makeService(),
		}, nil)
		renderer.On("Name").Return("mock")

		filter := podFilter()
		e, err := engine.New(
			engine.WithRenderer(renderer),
			engine.WithFilter(filter),
		)
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := e.Render(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(1))
		g.Expect(objects[0].GetKind()).To(Equal("Pod"))
	})

	t.Run("should apply engine-level transformer", func(t *testing.T) {
		g := NewWithT(t)
		renderer := new(mockRenderer)
		renderer.On("Process", mock.Anything, mock.Anything).Return([]unstructured.Unstructured{makePod("pod1")}, nil)
		renderer.On("Name").Return("mock")

		transformer := addLabels(map[string]string{
			"managed-by": "engine",
		})
		e, err := engine.New(
			engine.WithRenderer(renderer),
			engine.WithTransformer(transformer),
		)
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := e.Render(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(1))
		g.Expect(objects[0].GetLabels()).To(HaveKeyWithValue("managed-by", "engine"))
	})

	t.Run("should apply render-time filter", func(t *testing.T) {
		g := NewWithT(t)
		renderer := new(mockRenderer)
		renderer.On("Process", mock.Anything, mock.Anything).Return([]unstructured.Unstructured{
			makePod("pod1"),
			makeService(),
		}, nil)
		renderer.On("Name").Return("mock")

		e, err := engine.New(engine.WithRenderer(renderer))
		g.Expect(err).ToNot(HaveOccurred())

		filter := podFilter()
		objects, err := e.Render(t.Context(), render.WithFilter(filter))
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(1))
		g.Expect(objects[0].GetKind()).To(Equal("Pod"))
	})

	t.Run("should apply render-time transformer", func(t *testing.T) {
		g := NewWithT(t)
		renderer := new(mockRenderer)
		renderer.On("Process", mock.Anything, mock.Anything).Return([]unstructured.Unstructured{makePod("pod1")}, nil)
		renderer.On("Name").Return("mock")

		e, err := engine.New(engine.WithRenderer(renderer))
		g.Expect(err).ToNot(HaveOccurred())

		transformer := addLabels(map[string]string{
			"render-time": "true",
		})
		objects, err := e.Render(t.Context(), render.WithTransformer(transformer))
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(1))
		g.Expect(objects[0].GetLabels()).To(HaveKeyWithValue("render-time", "true"))
	})

	t.Run("should combine engine-level and render-time filters", func(t *testing.T) {
		g := NewWithT(t)
		renderer := new(mockRenderer)
		renderer.On("Process", mock.Anything, mock.Anything).Return([]unstructured.Unstructured{
			makePod("pod1"),
			makeService(),
			makePodWithNamespace("pod2", defaultNamespace),
			makePodWithNamespace("pod3", systemNamespace),
		}, nil)
		renderer.On("Name").Return("mock")

		engineFilter := podFilter()
		e, err := engine.New(
			engine.WithRenderer(renderer),
			engine.WithFilter(engineFilter),
		)
		g.Expect(err).ToNot(HaveOccurred())

		renderFilter := func(_ context.Context, obj unstructured.Unstructured) (bool, error) {
			return obj.GetNamespace() == defaultNamespace || obj.GetNamespace() == "", nil
		}

		objects, err := e.Render(t.Context(), render.WithFilter(renderFilter))
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(2))
	})

	t.Run("should combine engine-level and render-time transformers", func(t *testing.T) {
		g := NewWithT(t)
		renderer := new(mockRenderer)
		renderer.On("Process", mock.Anything, mock.Anything).Return([]unstructured.Unstructured{makePod("pod1")}, nil)
		renderer.On("Name").Return("mock")

		engineTransformer := addLabels(map[string]string{
			"engine": "level",
		})
		e, err := engine.New(
			engine.WithRenderer(renderer),
			engine.WithTransformer(engineTransformer),
		)
		g.Expect(err).ToNot(HaveOccurred())

		renderTransformer := addLabels(map[string]string{
			"render": "time",
		})

		objects, err := e.Render(t.Context(), render.WithTransformer(renderTransformer))
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(1))
		g.Expect(objects[0].GetLabels()).To(HaveKeyWithValue("engine", "level"))
		g.Expect(objects[0].GetLabels()).To(HaveKeyWithValue("render", "time"))
	})

	t.Run("should handle empty renderer", func(t *testing.T) {
		g := NewWithT(t)
		renderer := new(mockRenderer)
		renderer.On("Process", mock.Anything, mock.Anything).Return([]unstructured.Unstructured{}, nil)
		renderer.On("Name").Return("mock")

		e, err := engine.New(engine.WithRenderer(renderer))
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := e.Render(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(BeEmpty())
	})

	t.Run("should handle no renderers", func(t *testing.T) {
		g := NewWithT(t)
		e, err := engine.New()
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := e.Render(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(BeEmpty())
	})

	t.Run("should return error from failing renderer", func(t *testing.T) {
		g := NewWithT(t)
		r := new(mockRenderer)
		r.On("Process", mock.Anything, mock.Anything).Return([]unstructured.Unstructured{}, errors.New("renderer failed"))
		r.On("Name").Return("mock")

		e, err := engine.New(engine.WithRenderer(r))
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := e.Render(t.Context())
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("renderer failed"))
		g.Expect(objects).To(BeNil())
	})

	t.Run("should return error from failing filter", func(t *testing.T) {
		g := NewWithT(t)
		renderer := new(mockRenderer)
		renderer.On("Process", mock.Anything, mock.Anything).Return([]unstructured.Unstructured{makePod("pod1")}, nil)
		renderer.On("Name").Return("mock")

		failingFilter := func(_ context.Context, _ unstructured.Unstructured) (bool, error) {
			return false, errors.New("filter failed")
		}

		e, err := engine.New(
			engine.WithRenderer(renderer),
			engine.WithFilter(failingFilter),
		)
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := e.Render(t.Context())
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("filter failed"))
		g.Expect(objects).To(BeNil())
	})

	t.Run("should return error from failing transformer", func(t *testing.T) {
		g := NewWithT(t)
		renderer := new(mockRenderer)
		renderer.On("Process", mock.Anything, mock.Anything).Return([]unstructured.Unstructured{makePod("pod1")}, nil)
		renderer.On("Name").Return("mock")

		failingTransformer := func(_ context.Context, _ unstructured.Unstructured) (unstructured.Unstructured, error) {
			return unstructured.Unstructured{}, errors.New("transformer failed")
		}

		e, err := engine.New(
			engine.WithRenderer(renderer),
			engine.WithTransformer(failingTransformer),
		)
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := e.Render(t.Context())
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("transformer failed"))
		g.Expect(objects).To(BeNil())
	})

	t.Run("should apply multiple filters in sequence", func(t *testing.T) {
		g := NewWithT(t)
		renderer := new(mockRenderer)
		renderer.On("Process", mock.Anything, mock.Anything).Return([]unstructured.Unstructured{
			makePodWithNamespace("pod1", defaultNamespace),
			makePodWithNamespace("pod2", systemNamespace),
			makeService(),
		}, nil)
		renderer.On("Name").Return("mock")

		filter1 := podFilter()
		filter2 := func(_ context.Context, obj unstructured.Unstructured) (bool, error) {
			return obj.GetNamespace() == defaultNamespace, nil
		}

		e, err := engine.New(
			engine.WithRenderer(renderer),
			engine.WithFilter(filter1),
			engine.WithFilter(filter2),
		)
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := e.Render(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(1))
		g.Expect(objects[0].GetName()).To(Equal("pod1"))
	})

	t.Run("should apply multiple transformers in sequence", func(t *testing.T) {
		g := NewWithT(t)
		renderer := new(mockRenderer)
		renderer.On("Process", mock.Anything, mock.Anything).Return([]unstructured.Unstructured{makePod("pod1")}, nil)
		renderer.On("Name").Return("mock")

		transformer1 := addLabels(map[string]string{"label1": "value1"})
		transformer2 := addLabels(map[string]string{"label2": "value2"})

		e, err := engine.New(
			engine.WithRenderer(renderer),
			engine.WithTransformer(transformer1),
			engine.WithTransformer(transformer2),
		)
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := e.Render(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(1))
		g.Expect(objects[0].GetLabels()).To(HaveKeyWithValue("label1", "value1"))
		g.Expect(objects[0].GetLabels()).To(HaveKeyWithValue("label2", "value2"))
	})

	t.Run("should append struct-based render.Options filters to engine-level filters", func(t *testing.T) {
		g := NewWithT(t)
		renderer := new(mockRenderer)
		renderer.On("Process", mock.Anything, mock.Anything).Return([]unstructured.Unstructured{
			makePodWithNamespace("pod1", defaultNamespace),
			makePodWithNamespace("pod2", systemNamespace),
			makeService(),
		}, nil)
		renderer.On("Name").Return("mock")

		engineFilter := podFilter()
		e, err := engine.New(
			engine.WithRenderer(renderer),
			engine.WithFilter(engineFilter),
		)
		g.Expect(err).ToNot(HaveOccurred())

		renderFilter := func(_ context.Context, obj unstructured.Unstructured) (bool, error) {
			return obj.GetNamespace() == defaultNamespace, nil
		}

		objects, err := e.Render(t.Context(), render.Options{
			Filters: []types.Filter{renderFilter},
		})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(1))
		g.Expect(objects[0].GetName()).To(Equal("pod1"))
	})

	t.Run("should append struct-based render.Options transformers to engine-level transformers", func(t *testing.T) {
		g := NewWithT(t)
		renderer := new(mockRenderer)
		renderer.On("Process", mock.Anything, mock.Anything).Return([]unstructured.Unstructured{makePod("pod1")}, nil)
		renderer.On("Name").Return("mock")

		engineTransformer := addLabels(map[string]string{"engine": "level"})
		e, err := engine.New(
			engine.WithRenderer(renderer),
			engine.WithTransformer(engineTransformer),
		)
		g.Expect(err).ToNot(HaveOccurred())

		renderTransformer := addLabels(map[string]string{"render": "time"})

		objects, err := e.Render(t.Context(), render.Options{
			Transformers: []types.Transformer{renderTransformer},
		})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(1))
		g.Expect(objects[0].GetLabels()).To(HaveKeyWithValue("engine", "level"))
		g.Expect(objects[0].GetLabels()).To(HaveKeyWithValue("render", "time"))
	})

	t.Run("should apply engine-level post-renderer", func(t *testing.T) {
		g := NewWithT(t)
		renderer := new(mockRenderer)
		renderer.On("Process", mock.Anything, mock.Anything).Return([]unstructured.Unstructured{
			makePod("pod1"),
			makePod("pod2"),
		}, nil)
		renderer.On("Name").Return("mock")

		reversePR := func(_ context.Context, objects []unstructured.Unstructured) ([]unstructured.Unstructured, error) {
			for i, j := 0, len(objects)-1; i < j; i, j = i+1, j-1 {
				objects[i], objects[j] = objects[j], objects[i]
			}

			return objects, nil
		}

		e, err := engine.New(
			engine.WithRenderer(renderer),
			engine.WithPostRenderer(reversePR),
		)
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := e.Render(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(2))
		g.Expect(objects[0].GetName()).To(Equal("pod2"))
		g.Expect(objects[1].GetName()).To(Equal("pod1"))
	})

	t.Run("should apply render-time post-renderer", func(t *testing.T) {
		g := NewWithT(t)
		renderer := new(mockRenderer)
		renderer.On("Process", mock.Anything, mock.Anything).Return([]unstructured.Unstructured{
			makePod("pod1"),
			makePod("pod2"),
			makePod("pod3"),
		}, nil)
		renderer.On("Name").Return("mock")

		e, err := engine.New(engine.WithRenderer(renderer))
		g.Expect(err).ToNot(HaveOccurred())

		keepFirst := func(_ context.Context, objects []unstructured.Unstructured) ([]unstructured.Unstructured, error) {
			if len(objects) > 0 {
				return objects[:1], nil
			}

			return objects, nil
		}

		objects, err := e.Render(t.Context(), render.WithPostRenderer(keepFirst))
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(1))
		g.Expect(objects[0].GetName()).To(Equal("pod1"))
	})

	t.Run("should combine engine-level and render-time post-renderers", func(t *testing.T) {
		g := NewWithT(t)
		renderer := new(mockRenderer)
		renderer.On("Process", mock.Anything, mock.Anything).Return([]unstructured.Unstructured{
			makePod("pod1"),
			makePod("pod2"),
		}, nil)
		renderer.On("Name").Return("mock")

		enginePR := func(_ context.Context, objects []unstructured.Unstructured) ([]unstructured.Unstructured, error) {
			for i := range objects {
				k8s.SetLabel(&objects[i], "engine-pr", "applied")
			}

			return objects, nil
		}

		e, err := engine.New(
			engine.WithRenderer(renderer),
			engine.WithPostRenderer(enginePR),
		)
		g.Expect(err).ToNot(HaveOccurred())

		renderPR := func(_ context.Context, objects []unstructured.Unstructured) ([]unstructured.Unstructured, error) {
			for i := range objects {
				k8s.SetLabel(&objects[i], "render-pr", "applied")
			}

			return objects, nil
		}

		objects, err := e.Render(t.Context(), render.WithPostRenderer(renderPR))
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(2))

		for _, obj := range objects {
			g.Expect(obj.GetLabels()).To(HaveKeyWithValue("engine-pr", "applied"))
			g.Expect(obj.GetLabels()).To(HaveKeyWithValue("render-pr", "applied"))
		}
	})

	t.Run("should return error from failing post-renderer", func(t *testing.T) {
		g := NewWithT(t)
		renderer := new(mockRenderer)
		renderer.On("Process", mock.Anything, mock.Anything).Return([]unstructured.Unstructured{makePod("pod1")}, nil)
		renderer.On("Name").Return("mock")

		failingPR := func(_ context.Context, _ []unstructured.Unstructured) ([]unstructured.Unstructured, error) {
			return nil, errors.New("post-renderer failed")
		}

		e, err := engine.New(
			engine.WithRenderer(renderer),
			engine.WithPostRenderer(failingPR),
		)
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := e.Render(t.Context())
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("post-renderer failed"))
		g.Expect(objects).To(BeNil())
	})
}
