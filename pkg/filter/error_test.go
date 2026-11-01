package filter_test

import (
	"errors"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/k8s-manifest-kit/engine/pkg/filter"

	. "github.com/onsi/gomega"
)

func TestFilterWrap(t *testing.T) {
	t.Run("should return nil for nil error", func(t *testing.T) {
		g := NewWithT(t)

		err := filter.Wrap(makePod("pod1"), nil)
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("should wrap error with object context", func(t *testing.T) {
		g := NewWithT(t)

		underlying := errors.New("custom filter error")

		err := filter.Wrap(makeNamespacedPod("pod1", "default"), underlying)
		g.Expect(err).To(HaveOccurred())

		filterErr, ok := errors.AsType[*filter.Error](err)
		g.Expect(ok).To(BeTrue())
		g.Expect(filterErr.Object.GetKind()).To(Equal("Pod"))
		g.Expect(filterErr.Object.GetName()).To(Equal("pod1"))
		g.Expect(filterErr.Object.GetNamespace()).To(Equal("default"))
		g.Expect(filterErr.Err.Error()).To(Equal("custom filter error"))
	})

	t.Run("should preserve underlying error cause", func(t *testing.T) {
		g := NewWithT(t)

		underlying := errors.New("underlying error")

		err := filter.Wrap(makePod("pod1"), underlying)
		g.Expect(err).To(HaveOccurred())
		g.Expect(errors.Is(err, underlying)).To(BeTrue())
	})

	t.Run("should not double-wrap Error", func(t *testing.T) {
		g := NewWithT(t)

		original := errors.New("original error")
		typed := &filter.Error{
			Object: makeNamespacedPod("original-pod", "original-ns"),
			Err:    original,
		}

		err := filter.Wrap(makePod("other-pod"), typed)
		g.Expect(err).To(HaveOccurred())
		// The existing typed error is returned as-is, preserving its object context.
		g.Expect(err).To(BeIdenticalTo(typed))

		filterErr, ok := errors.AsType[*filter.Error](err)
		g.Expect(ok).To(BeTrue())
		g.Expect(filterErr.Object.GetName()).To(Equal("original-pod"))
		g.Expect(filterErr.Object.GetNamespace()).To(Equal("original-ns"))
		g.Expect(filterErr.Err).To(Equal(original))
		g.Expect(filterErr.Err).ToNot(BeAssignableToTypeOf(&filter.Error{}))
	})
}

func makeNamespacedPod(name string, namespace string) unstructured.Unstructured {
	obj := makePod(name)
	obj.SetNamespace(namespace)

	return obj
}
