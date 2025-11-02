package namespace_test

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/k8s-manifest-kit/engine/pkg/filter/meta/namespace"

	. "github.com/onsi/gomega"
)

const (
	defaultNS = "default"
	systemNS  = "kube-system"
	prodNS    = "production"
)

func TestFilter(t *testing.T) {
	g := NewWithT(t)

	t.Run("should keep objects in specified namespaces", func(t *testing.T) {
		filter := namespace.Filter(defaultNS, systemNS)

		ok, err := filter(t.Context(), makePodInNamespace("test", defaultNS))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeTrue())

		ok, err = filter(t.Context(), makePodInNamespace("test", systemNS))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeTrue())
	})

	t.Run("should exclude objects not in specified namespaces", func(t *testing.T) {
		filter := namespace.Filter(defaultNS)

		ok, err := filter(t.Context(), makePodInNamespace("test", prodNS))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeFalse())
	})

	t.Run("should handle empty namespace for cluster-scoped resources", func(t *testing.T) {
		filter := namespace.Filter("")

		ok, err := filter(t.Context(), makePodInNamespace("test", ""))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeTrue())
	})

	t.Run("should handle multiple namespaces", func(t *testing.T) {
		filter := namespace.Filter(defaultNS, systemNS, prodNS)

		ok, err := filter(t.Context(), makePodInNamespace("test", defaultNS))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeTrue())

		ok, err = filter(t.Context(), makePodInNamespace("test", prodNS))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeTrue())

		ok, err = filter(t.Context(), makePodInNamespace("test", "other"))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeFalse())
	})
}

func TestExclude(t *testing.T) {
	g := NewWithT(t)

	t.Run("should exclude objects in specified namespaces", func(t *testing.T) {
		filter := namespace.Exclude(systemNS)

		ok, err := filter(t.Context(), makePodInNamespace("test", systemNS))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeFalse())
	})

	t.Run("should keep objects not in excluded namespaces", func(t *testing.T) {
		filter := namespace.Exclude(systemNS)

		ok, err := filter(t.Context(), makePodInNamespace("test", defaultNS))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeTrue())

		ok, err = filter(t.Context(), makePodInNamespace("test", prodNS))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeTrue())
	})

	t.Run("should handle multiple excluded namespaces", func(t *testing.T) {
		filter := namespace.Exclude(systemNS, defaultNS)

		ok, err := filter(t.Context(), makePodInNamespace("test", systemNS))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeFalse())

		ok, err = filter(t.Context(), makePodInNamespace("test", defaultNS))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeFalse())

		ok, err = filter(t.Context(), makePodInNamespace("test", prodNS))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeTrue())
	})
}

// Helper functions

//nolint:unparam // Test helper needs consistent signature
func makePodInNamespace(name string, ns string) unstructured.Unstructured {
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
