package name_test

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/k8s-manifest-kit/engine/pkg/filter/meta/name"

	. "github.com/onsi/gomega"
)

func TestExact(t *testing.T) {

	t.Run("should keep objects with exact name match", func(t *testing.T) {
		g := NewWithT(t)
		filter := name.Exact("nginx-pod", "apache-pod")

		ok, err := filter(t.Context(), makePod("nginx-pod"))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeTrue())

		ok, err = filter(t.Context(), makePod("apache-pod"))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeTrue())
	})

	t.Run("should exclude objects without exact match", func(t *testing.T) {
		g := NewWithT(t)
		filter := name.Exact("nginx-pod")

		ok, err := filter(t.Context(), makePod("nginx-deployment"))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeFalse())
	})
}

func TestPrefix(t *testing.T) {

	t.Run("should keep objects with matching prefix", func(t *testing.T) {
		g := NewWithT(t)
		filter := name.Prefix("nginx-")

		ok, err := filter(t.Context(), makePod("nginx-pod"))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeTrue())

		ok, err = filter(t.Context(), makePod("nginx-deployment"))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeTrue())
	})

	t.Run("should exclude objects without prefix", func(t *testing.T) {
		g := NewWithT(t)
		filter := name.Prefix("nginx-")

		ok, err := filter(t.Context(), makePod("apache-pod"))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeFalse())
	})
}

func TestSuffix(t *testing.T) {

	t.Run("should keep objects with matching suffix", func(t *testing.T) {
		g := NewWithT(t)
		filter := name.Suffix("-pod")

		ok, err := filter(t.Context(), makePod("nginx-pod"))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeTrue())

		ok, err = filter(t.Context(), makePod("apache-pod"))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeTrue())
	})

	t.Run("should exclude objects without suffix", func(t *testing.T) {
		g := NewWithT(t)
		filter := name.Suffix("-pod")

		ok, err := filter(t.Context(), makePod("nginx-deployment"))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeFalse())
	})
}

func TestRegex(t *testing.T) {

	t.Run("should keep objects matching regex", func(t *testing.T) {
		g := NewWithT(t)
		filter, err := name.Regex("^nginx-.*-[0-9]+$")
		g.Expect(err).ShouldNot(HaveOccurred())

		ok, err := filter(t.Context(), makePod("nginx-pod-123"))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeTrue())

		ok, err = filter(t.Context(), makePod("nginx-deployment-456"))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeTrue())
	})

	t.Run("should exclude objects not matching regex", func(t *testing.T) {
		g := NewWithT(t)
		filter, err := name.Regex("^nginx-")
		g.Expect(err).ShouldNot(HaveOccurred())

		ok, err := filter(t.Context(), makePod("apache-pod"))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeFalse())
	})

	t.Run("should return error for invalid regex", func(t *testing.T) {
		g := NewWithT(t)
		_, err := name.Regex("[invalid")
		g.Expect(err).Should(HaveOccurred())
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
