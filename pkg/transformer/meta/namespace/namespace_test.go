package namespace_test

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/k8s-manifest-kit/engine/pkg/transformer/meta/namespace"

	. "github.com/onsi/gomega"
)

func TestSet(t *testing.T) {
	g := NewWithT(t)

	t.Run("should set namespace on object", func(t *testing.T) {
		transformer := namespace.Set("production")

		obj, err := transformer(t.Context(), makePod("test", ""))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(obj.GetNamespace()).Should(Equal("production"))
	})

	t.Run("should overwrite existing namespace", func(t *testing.T) {
		transformer := namespace.Set("production")

		obj, err := transformer(t.Context(), makePod("test", "default"))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(obj.GetNamespace()).Should(Equal("production"))
	})
}

func TestEnsureDefault(t *testing.T) {
	g := NewWithT(t)

	t.Run("should set namespace when empty", func(t *testing.T) {
		transformer := namespace.EnsureDefault("default")

		obj, err := transformer(t.Context(), makePod("test", ""))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(obj.GetNamespace()).Should(Equal("default"))
	})

	t.Run("should not overwrite existing namespace", func(t *testing.T) {
		transformer := namespace.EnsureDefault("default")

		obj, err := transformer(t.Context(), makePod("test", "production"))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(obj.GetNamespace()).Should(Equal("production"))
	})
}

// Helper function

//nolint:unparam // Test helper needs consistent signature
func makePod(name string, ns string) unstructured.Unstructured {
	obj := unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]any{
				"name":      name,
				"namespace": ns,
			},
		},
	}
	obj.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Pod"))

	return obj
}
