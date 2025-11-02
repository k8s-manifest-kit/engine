package annotations_test

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/k8s-manifest-kit/engine/pkg/filter/meta/annotations"

	. "github.com/onsi/gomega"
)

func TestHasAnnotation(t *testing.T) {
	g := NewWithT(t)

	t.Run("should keep objects with the annotation", func(t *testing.T) {
		filter := annotations.HasAnnotation("kubectl.kubernetes.io/last-applied-configuration")

		ok, err := filter(t.Context(), makePodWithAnnotations(map[string]string{
			"kubectl.kubernetes.io/last-applied-configuration": "{}",
		}))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeTrue())
	})

	t.Run("should exclude objects without the annotation", func(t *testing.T) {
		filter := annotations.HasAnnotation("missing")

		ok, err := filter(t.Context(), makePodWithAnnotations(map[string]string{
			"other": "value",
		}))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeFalse())
	})

	t.Run("should handle objects with no annotations", func(t *testing.T) {
		filter := annotations.HasAnnotation("any")

		ok, err := filter(t.Context(), makePodWithAnnotations(nil))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeFalse())
	})
}

func TestHasAnnotations(t *testing.T) {
	g := NewWithT(t)

	t.Run("should keep objects with all annotations", func(t *testing.T) {
		filter := annotations.HasAnnotations("ann1", "ann2")

		ok, err := filter(t.Context(), makePodWithAnnotations(map[string]string{
			"ann1": "value1",
			"ann2": "value2",
		}))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeTrue())
	})

	t.Run("should exclude objects missing any annotation", func(t *testing.T) {
		filter := annotations.HasAnnotations("ann1", "ann2")

		ok, err := filter(t.Context(), makePodWithAnnotations(map[string]string{
			"ann1": "value1",
		}))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeFalse())
	})
}

func TestMatchAnnotations(t *testing.T) {
	g := NewWithT(t)

	t.Run("should keep objects with matching annotations", func(t *testing.T) {
		filter := annotations.MatchAnnotations(map[string]string{
			"version": "1.0",
			"author":  "test",
		})

		ok, err := filter(t.Context(), makePodWithAnnotations(map[string]string{
			"version": "1.0",
			"author":  "test",
			"extra":   "ignored",
		}))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeTrue())
	})

	t.Run("should exclude objects with non-matching value", func(t *testing.T) {
		filter := annotations.MatchAnnotations(map[string]string{
			"version": "1.0",
		})

		ok, err := filter(t.Context(), makePodWithAnnotations(map[string]string{
			"version": "2.0",
		}))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeFalse())
	})
}

// Helper function

func makePodWithAnnotations(anns map[string]string) unstructured.Unstructured {
	obj := unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]any{
				"name": "test",
			},
		},
	}
	obj.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Pod"))
	if anns != nil {
		obj.SetAnnotations(anns)
	}

	return obj
}
