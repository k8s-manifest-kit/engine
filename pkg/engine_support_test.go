package engine_test

import (
	"context"
	"maps"
	"slices"

	"github.com/stretchr/testify/mock"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/k8s-manifest-kit/engine/pkg/types"
)

const (
	defaultNamespace = "default"
	systemNamespace  = "kube-system"
)

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
	//nolint:wrapcheck // mock returns the results configured by the caller, including its error, unchanged
	return args.Get(0).([]unstructured.Unstructured), args.Error(1)
}

func (m *mockRenderer) Name() string {
	args := m.Called()

	return args.String(0)
}

// hashRenderer is a deterministic types.Renderer that returns deep copies of
// its source objects annotated with real source and content-hash annotations
// through types.AnnotateObjects. Sources are owned by the renderer and never
// annotated, and previously annotated results are never retained as source,
// so every Process call hashes a fresh copy of the same input.
type hashRenderer struct {
	name    string
	sources []unstructured.Unstructured
}

func newHashRenderer(name string, sources ...unstructured.Unstructured) *hashRenderer {
	return &hashRenderer{
		name:    name,
		sources: slices.Clone(sources),
	}
}

func (r *hashRenderer) Name() string {
	return r.name
}

func (r *hashRenderer) Process(
	_ context.Context,
	_ types.Values,
) ([]unstructured.Unstructured, error) {
	objects := make([]unstructured.Unstructured, 0, len(r.sources))
	for i := range r.sources {
		objects = append(objects, *r.sources[i].DeepCopy())
	}

	types.AnnotateObjects(
		objects,
		types.WithSourceAnnotations(map[string]string{
			types.AnnotationSourceType: r.name,
		}),
		types.WithContentHash(true),
	)

	return objects, nil
}
