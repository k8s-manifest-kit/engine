package transformer_test

import (
	"errors"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/k8s-manifest-kit/engine/pkg/transformer"

	. "github.com/onsi/gomega"
)

func TestTransformerWrap(t *testing.T) {
	t.Run("should return nil for nil error", func(t *testing.T) {
		g := NewWithT(t)

		err := transformer.Wrap(makePod("pod1"), nil)
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("should wrap error with object context", func(t *testing.T) {
		g := NewWithT(t)

		underlying := errors.New("custom transformer error")

		err := transformer.Wrap(makeNamespacedPod("pod1", "default"), underlying)
		g.Expect(err).To(HaveOccurred())

		transformerErr, ok := errors.AsType[*transformer.Error](err)
		g.Expect(ok).To(BeTrue())
		g.Expect(transformerErr.Object.GetKind()).To(Equal("Pod"))
		g.Expect(transformerErr.Object.GetName()).To(Equal("pod1"))
		g.Expect(transformerErr.Object.GetNamespace()).To(Equal("default"))
		g.Expect(transformerErr.Err.Error()).To(Equal("custom transformer error"))
	})

	t.Run("should preserve underlying error cause", func(t *testing.T) {
		g := NewWithT(t)

		underlying := errors.New("underlying error")

		err := transformer.Wrap(makePod("pod1"), underlying)
		g.Expect(err).To(HaveOccurred())
		g.Expect(errors.Is(err, underlying)).To(BeTrue())
	})

	t.Run("should not double-wrap Error", func(t *testing.T) {
		g := NewWithT(t)

		original := errors.New("original error")
		typed := &transformer.Error{
			Object: makeNamespacedPod("original-pod", "original-ns"),
			Err:    original,
		}

		err := transformer.Wrap(makePod("other-pod"), typed)
		g.Expect(err).To(HaveOccurred())
		// The existing typed error is returned as-is, preserving its object context.
		g.Expect(err).To(BeIdenticalTo(typed))

		transformerErr, ok := errors.AsType[*transformer.Error](err)
		g.Expect(ok).To(BeTrue())
		g.Expect(transformerErr.Object.GetName()).To(Equal("original-pod"))
		g.Expect(transformerErr.Object.GetNamespace()).To(Equal("original-ns"))
		g.Expect(transformerErr.Err).To(Equal(original))
		g.Expect(transformerErr.Err).ToNot(BeAssignableToTypeOf(&transformer.Error{}))
	})
}

func makeNamespacedPod(name string, namespace string) unstructured.Unstructured {
	obj := makePod(name)
	obj.SetNamespace(namespace)

	return obj
}
