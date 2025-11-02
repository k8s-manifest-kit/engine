package name_test

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/k8s-manifest-kit/engine/pkg/transformer/meta/name"

	. "github.com/onsi/gomega"
)

func TestSetPrefix(t *testing.T) {
	g := NewWithT(t)

	t.Run("should add prefix to name", func(t *testing.T) {
		transformer := name.SetPrefix("prod-")

		obj, err := transformer(t.Context(), makePod("nginx"))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(obj.GetName()).Should(Equal("prod-nginx"))
	})

	t.Run("should handle empty prefix", func(t *testing.T) {
		transformer := name.SetPrefix("")

		obj, err := transformer(t.Context(), makePod("nginx"))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(obj.GetName()).Should(Equal("nginx"))
	})
}

func TestSetSuffix(t *testing.T) {
	g := NewWithT(t)

	t.Run("should add suffix to name", func(t *testing.T) {
		transformer := name.SetSuffix("-v2")

		obj, err := transformer(t.Context(), makePod("nginx"))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(obj.GetName()).Should(Equal("nginx-v2"))
	})

	t.Run("should handle empty suffix", func(t *testing.T) {
		transformer := name.SetSuffix("")

		obj, err := transformer(t.Context(), makePod("nginx"))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(obj.GetName()).Should(Equal("nginx"))
	})
}

func TestReplace(t *testing.T) {
	g := NewWithT(t)

	t.Run("should replace substring in name", func(t *testing.T) {
		transformer := name.Replace("nginx", "apache")

		obj, err := transformer(t.Context(), makePod("nginx-pod"))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(obj.GetName()).Should(Equal("apache-pod"))
	})

	t.Run("should replace all occurrences", func(t *testing.T) {
		transformer := name.Replace("test", "prod")

		obj, err := transformer(t.Context(), makePod("test-test-pod"))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(obj.GetName()).Should(Equal("prod-prod-pod"))
	})

	t.Run("should handle no match", func(t *testing.T) {
		transformer := name.Replace("missing", "replacement")

		obj, err := transformer(t.Context(), makePod("nginx"))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(obj.GetName()).Should(Equal("nginx"))
	})
}

// Helper function

func makePod(podName string) unstructured.Unstructured {
	obj := unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]any{
				"name": podName,
			},
		},
	}
	obj.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Pod"))

	return obj
}
