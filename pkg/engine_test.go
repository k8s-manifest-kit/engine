package engine_test

import (
	"context"
	"errors"
	"maps"
	"testing"

	"github.com/k8s-manifest-kit/pkg/util/k8s"
	"github.com/stretchr/testify/mock"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	engine "github.com/k8s-manifest-kit/engine/pkg"
	"github.com/k8s-manifest-kit/engine/pkg/render"
	"github.com/k8s-manifest-kit/engine/pkg/types"

	. "github.com/onsi/gomega"
)

const (
	defaultNamespace = "default"
	systemNamespace  = "kube-system"
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

// Helper functions

func makePod(name string) unstructured.Unstructured {
	obj := unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]any{
				"name": name,
			},
		},
	}
	obj.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Pod"))

	return obj
}

func makePodWithNamespace(name string, namespace string) unstructured.Unstructured {
	obj := unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
			},
		},
	}
	obj.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Pod"))

	return obj
}

func makeService() unstructured.Unstructured {
	obj := unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata": map[string]any{
				"name": "svc1",
			},
		},
	}
	obj.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Service"))

	return obj
}

// podFilter returns a filter that only accepts Pod kind objects.
func podFilter() func(context.Context, unstructured.Unstructured) (bool, error) {
	return func(_ context.Context, obj unstructured.Unstructured) (bool, error) {
		return obj.GetKind() == "Pod", nil
	}
}

// addLabels returns a transformer that adds the given labels to objects.
func addLabels(
	labels map[string]string,
) func(context.Context, unstructured.Unstructured) (unstructured.Unstructured, error) {
	return func(_ context.Context, obj unstructured.Unstructured) (unstructured.Unstructured, error) {
		existingLabels := obj.GetLabels()
		if existingLabels == nil {
			existingLabels = make(map[string]string)
		}
		maps.Copy(existingLabels, labels)
		obj.SetLabels(existingLabels)

		return obj, nil
	}
}

// mockRenderer is a mock implementation of types.Renderer for testing using testify/mock.
type mockRenderer struct {
	mock.Mock
}

func (m *mockRenderer) Process(ctx context.Context, values types.Values) ([]unstructured.Unstructured, error) {
	args := m.Called(ctx, values)
	//nolint:wrapcheck
	return args.Get(0).([]unstructured.Unstructured), args.Error(1)
}

func (m *mockRenderer) Name() string {
	args := m.Called()

	return args.String(0)
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

func TestSourceAnnotations(t *testing.T) {

	t.Run("should render objects with source annotations when renderer has them enabled", func(t *testing.T) {
		g := NewWithT(t)
		pod := &corev1.Pod{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Pod",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-pod",
				Annotations: map[string]string{
					types.AnnotationSourceType: "test-renderer",
				},
			},
		}

		unstrPod, err := k8s.ToUnstructured(pod)
		g.Expect(err).ToNot(HaveOccurred())

		renderer := new(mockRenderer)
		renderer.On("Process", mock.Anything, mock.Anything).Return([]unstructured.Unstructured{*unstrPod}, nil)
		renderer.On("Name").Return("mock")

		e, err := engine.New(
			engine.WithRenderer(renderer),
		)
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := e.Render(t.Context())

		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(objects).Should(HaveLen(1))

		annotations := objects[0].GetAnnotations()
		g.Expect(annotations).Should(HaveKeyWithValue(types.AnnotationSourceType, "test-renderer"))
	})

	t.Run("should not have source annotations when renderer has them disabled", func(t *testing.T) {
		g := NewWithT(t)
		pod := &corev1.Pod{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Pod",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-pod",
			},
		}

		unstrPod, err := k8s.ToUnstructured(pod)
		g.Expect(err).ToNot(HaveOccurred())

		renderer := new(mockRenderer)
		renderer.On("Process", mock.Anything, mock.Anything).Return([]unstructured.Unstructured{*unstrPod}, nil)
		renderer.On("Name").Return("mock")

		e, err := engine.New(
			engine.WithRenderer(renderer),
		)
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := e.Render(t.Context())

		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(objects).Should(HaveLen(1))

		annotations := objects[0].GetAnnotations()
		g.Expect(annotations).ShouldNot(HaveKey(types.AnnotationSourceType))
	})

	t.Run("should work with multiple renderers with different annotation settings", func(t *testing.T) {
		g := NewWithT(t)
		pod1 := &corev1.Pod{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Pod",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "pod-with-annotations",
			},
		}

		pod2 := &corev1.Pod{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Pod",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "pod-without-annotations",
			},
		}

		pod1.Annotations = map[string]string{
			types.AnnotationSourceType: "test-renderer-1",
		}

		unstrPod1, err := k8s.ToUnstructured(pod1)
		g.Expect(err).ToNot(HaveOccurred())

		unstrPod2, err := k8s.ToUnstructured(pod2)
		g.Expect(err).ToNot(HaveOccurred())

		renderer1 := new(mockRenderer)
		renderer1.On("Process", mock.Anything, mock.Anything).Return([]unstructured.Unstructured{*unstrPod1}, nil)
		renderer1.On("Name").Return("mock")

		renderer2 := new(mockRenderer)
		renderer2.On("Process", mock.Anything, mock.Anything).Return([]unstructured.Unstructured{*unstrPod2}, nil)
		renderer2.On("Name").Return("mock")

		e, err := engine.New(
			engine.WithRenderer(renderer1),
			engine.WithRenderer(renderer2),
		)
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := e.Render(t.Context())

		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(objects).Should(HaveLen(2))

		for _, obj := range objects {
			annotations := obj.GetAnnotations()
			if obj.GetName() == "pod-with-annotations" {
				g.Expect(annotations).Should(HaveKeyWithValue(types.AnnotationSourceType, "test-renderer-1"))
			} else if obj.GetName() == "pod-without-annotations" {
				g.Expect(annotations).ShouldNot(HaveKey(types.AnnotationSourceType))
			}
		}
	})
}

func TestValidateRenderer(t *testing.T) {

	t.Run("should accept valid renderer", func(t *testing.T) {
		g := NewWithT(t)
		renderer := new(mockRenderer)
		renderer.On("Process", mock.Anything, mock.Anything).Return([]unstructured.Unstructured{makePod("pod1")}, nil)
		renderer.On("Name").Return("mock")

		err := types.ValidateRenderer(renderer)
		g.Expect(err).ShouldNot(HaveOccurred())
	})

	t.Run("should reject nil renderer", func(t *testing.T) {
		g := NewWithT(t)
		err := types.ValidateRenderer(nil)
		g.Expect(err).Should(HaveOccurred())
		g.Expect(err.Error()).Should(ContainSubstring("renderer cannot be nil"))
	})

	t.Run("should reject renderer with empty name", func(t *testing.T) {
		g := NewWithT(t)
		renderer := new(mockRenderer)
		renderer.On("Process", mock.Anything, mock.Anything).Return([]unstructured.Unstructured{makePod("pod1")}, nil)
		renderer.On("Name").Return("")

		err := types.ValidateRenderer(renderer)
		g.Expect(err).Should(HaveOccurred())
		g.Expect(err.Error()).Should(ContainSubstring("must return a non-empty name"))
	})

	t.Run("should reject renderer with whitespace-only name", func(t *testing.T) {
		g := NewWithT(t)
		renderer := new(mockRenderer)
		renderer.On("Process", mock.Anything, mock.Anything).Return([]unstructured.Unstructured{makePod("pod1")}, nil)
		renderer.On("Name").Return("   \t\n  ")

		err := types.ValidateRenderer(renderer)
		g.Expect(err).Should(HaveOccurred())
		g.Expect(err.Error()).Should(ContainSubstring("must return a non-empty name"))
	})
}

func TestNewValidatesRenderers(t *testing.T) {

	t.Run("should reject engine creation with nil renderer", func(t *testing.T) {
		g := NewWithT(t)
		e, err := engine.New(engine.WithRenderer(nil))
		g.Expect(err).Should(HaveOccurred())
		g.Expect(err.Error()).Should(ContainSubstring("invalid renderer"))
		g.Expect(err.Error()).Should(ContainSubstring("renderer cannot be nil"))
		g.Expect(e).Should(BeNil())
	})

	t.Run("should reject engine creation with renderer with empty name", func(t *testing.T) {
		g := NewWithT(t)
		renderer := new(mockRenderer)
		renderer.On("Process", mock.Anything, mock.Anything).Return([]unstructured.Unstructured{makePod("pod1")}, nil)
		renderer.On("Name").Return("")

		e, err := engine.New(engine.WithRenderer(renderer))
		g.Expect(err).Should(HaveOccurred())
		g.Expect(err.Error()).Should(ContainSubstring("invalid renderer"))
		g.Expect(err.Error()).Should(ContainSubstring("must return a non-empty name"))
		g.Expect(e).Should(BeNil())
	})

	t.Run("should accept engine creation with valid renderer", func(t *testing.T) {
		g := NewWithT(t)
		renderer := new(mockRenderer)
		renderer.On("Process", mock.Anything, mock.Anything).Return([]unstructured.Unstructured{makePod("pod1")}, nil)
		renderer.On("Name").Return("mock")

		e, err := engine.New(engine.WithRenderer(renderer))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(e).ShouldNot(BeNil())
	})
}
