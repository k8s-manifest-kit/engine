package labels_test

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/k8s-manifest-kit/engine/pkg/filter/meta/labels"

	. "github.com/onsi/gomega"
)

func TestHasLabel(t *testing.T) {

	t.Run("should keep objects with the label", func(t *testing.T) {
		g := NewWithT(t)
		filter := labels.HasLabel("app")

		ok, err := filter(t.Context(), makePodWithLabels(map[string]string{"app": "nginx"}))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeTrue())
	})

	t.Run("should exclude objects without the label", func(t *testing.T) {
		g := NewWithT(t)
		filter := labels.HasLabel("app")

		ok, err := filter(t.Context(), makePodWithLabels(map[string]string{"version": "1.0"}))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeFalse())
	})

	t.Run("should handle objects with no labels", func(t *testing.T) {
		g := NewWithT(t)
		filter := labels.HasLabel("app")

		ok, err := filter(t.Context(), makePodWithLabels(nil))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeFalse())
	})
}

func TestHasLabels(t *testing.T) {

	t.Run("should keep objects with all labels", func(t *testing.T) {
		g := NewWithT(t)
		filter := labels.HasLabels("app", "version")

		ok, err := filter(t.Context(), makePodWithLabels(map[string]string{
			"app":     "nginx",
			"version": "1.0",
		}))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeTrue())
	})

	t.Run("should exclude objects missing any label", func(t *testing.T) {
		g := NewWithT(t)
		filter := labels.HasLabels("app", "version")

		ok, err := filter(t.Context(), makePodWithLabels(map[string]string{
			"app": "nginx",
		}))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeFalse())
	})

	t.Run("should pass with empty label list", func(t *testing.T) {
		g := NewWithT(t)
		filter := labels.HasLabels()

		ok, err := filter(t.Context(), makePodWithLabels(nil))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeTrue())
	})
}

func TestMatchLabels(t *testing.T) {

	t.Run("should keep objects with matching labels", func(t *testing.T) {
		g := NewWithT(t)
		filter := labels.MatchLabels(map[string]string{
			"app": "nginx",
			"env": "prod",
		})

		ok, err := filter(t.Context(), makePodWithLabels(map[string]string{
			"app":     "nginx",
			"env":     "prod",
			"version": "1.0",
		}))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeTrue())
	})

	t.Run("should exclude objects with non-matching value", func(t *testing.T) {
		g := NewWithT(t)
		filter := labels.MatchLabels(map[string]string{
			"app": "nginx",
		})

		ok, err := filter(t.Context(), makePodWithLabels(map[string]string{
			"app": "apache",
		}))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeFalse())
	})

	t.Run("should exclude objects missing label", func(t *testing.T) {
		g := NewWithT(t)
		filter := labels.MatchLabels(map[string]string{
			"app": "nginx",
		})

		ok, err := filter(t.Context(), makePodWithLabels(map[string]string{
			"version": "1.0",
		}))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeFalse())
	})

	t.Run("should pass with empty match labels", func(t *testing.T) {
		g := NewWithT(t)
		filter := labels.MatchLabels(map[string]string{})

		ok, err := filter(t.Context(), makePodWithLabels(nil))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeTrue())
	})
}

func TestSelector(t *testing.T) {

	t.Run("should support equality selector", func(t *testing.T) {
		g := NewWithT(t)
		filter, err := labels.Selector("app=nginx")
		g.Expect(err).ShouldNot(HaveOccurred())

		ok, err := filter(t.Context(), makePodWithLabels(map[string]string{"app": "nginx"}))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeTrue())

		ok, err = filter(t.Context(), makePodWithLabels(map[string]string{"app": "apache"}))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeFalse())
	})

	t.Run("should support inequality selector", func(t *testing.T) {
		g := NewWithT(t)
		filter, err := labels.Selector("env!=prod")
		g.Expect(err).ShouldNot(HaveOccurred())

		ok, err := filter(t.Context(), makePodWithLabels(map[string]string{"env": "dev"}))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeTrue())

		ok, err = filter(t.Context(), makePodWithLabels(map[string]string{"env": "prod"}))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeFalse())
	})

	t.Run("should support set-based selectors", func(t *testing.T) {
		g := NewWithT(t)
		filter, err := labels.Selector("env in (dev,staging)")
		g.Expect(err).ShouldNot(HaveOccurred())

		ok, err := filter(t.Context(), makePodWithLabels(map[string]string{"env": "dev"}))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeTrue())

		ok, err = filter(t.Context(), makePodWithLabels(map[string]string{"env": "prod"}))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeFalse())
	})

	t.Run("should support combined selectors", func(t *testing.T) {
		g := NewWithT(t)
		filter, err := labels.Selector("app=nginx,env!=prod")
		g.Expect(err).ShouldNot(HaveOccurred())

		ok, err := filter(t.Context(), makePodWithLabels(map[string]string{
			"app": "nginx",
			"env": "dev",
		}))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeTrue())

		ok, err = filter(t.Context(), makePodWithLabels(map[string]string{
			"app": "nginx",
			"env": "prod",
		}))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeFalse())
	})

	t.Run("should return error for invalid selector", func(t *testing.T) {
		g := NewWithT(t)
		_, err := labels.Selector("invalid=selector=syntax")
		g.Expect(err).Should(HaveOccurred())
	})
}

// Helper functions

func makePodWithLabels(lbls map[string]string) unstructured.Unstructured {
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
	if lbls != nil {
		obj.SetLabels(lbls)
	}

	return obj
}
