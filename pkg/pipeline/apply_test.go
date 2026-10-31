package pipeline_test

import (
	"context"
	"errors"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/k8s-manifest-kit/engine/pkg/pipeline"
	"github.com/k8s-manifest-kit/engine/pkg/types"

	. "github.com/onsi/gomega"
)

func TestApply(t *testing.T) {
	ctx := t.Context()

	t.Run("should apply filters then transformers in sequence", func(t *testing.T) {
		g := NewWithT(t)
		objects := []unstructured.Unstructured{
			makeObjectWithNamespace("Pod", "pod1", "default"),
			makeObjectWithNamespace("Service", "svc1", "default"),
			makeObjectWithNamespace("Pod", "pod2", "kube-system"),
		}

		podFilter := func(_ context.Context, obj unstructured.Unstructured) (bool, error) {
			return obj.GetKind() == kindPod, nil
		}

		addLabelTransformer := func(_ context.Context, obj unstructured.Unstructured) (unstructured.Unstructured, error) {
			labels := obj.GetLabels()
			if labels == nil {
				labels = make(map[string]string)
			}
			labels["filtered"] = labelValueTrue
			obj.SetLabels(labels)

			return obj, nil
		}

		result, err := pipeline.Apply(ctx, objects, []types.Filter{podFilter}, []types.Transformer{addLabelTransformer})

		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(HaveLen(2))
		for _, obj := range result {
			g.Expect(obj.GetKind()).To(Equal("Pod"))
			g.Expect(obj.GetLabels()).To(HaveKeyWithValue("filtered", labelValueTrue))
		}
	})

	t.Run("should stop on filter error without calling transformers", func(t *testing.T) {
		g := NewWithT(t)
		objects := []unstructured.Unstructured{
			makeObject("Pod", "pod1"),
		}

		errorFilter := func(_ context.Context, _ unstructured.Unstructured) (bool, error) {
			return false, errors.New("filter failed")
		}

		transformerCalled := false
		transformer := func(_ context.Context, obj unstructured.Unstructured) (unstructured.Unstructured, error) {
			transformerCalled = true

			return obj, nil
		}

		result, err := pipeline.Apply(ctx, objects, []types.Filter{errorFilter}, []types.Transformer{transformer})

		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("filter failed"))
		g.Expect(result).To(BeNil())
		g.Expect(transformerCalled).To(BeFalse())
	})

	t.Run("should handle empty result from filters", func(t *testing.T) {
		g := NewWithT(t)
		objects := []unstructured.Unstructured{
			makeObject("Pod", "pod1"),
			makeObject("Service", "svc1"),
		}

		rejectAllFilter := func(_ context.Context, _ unstructured.Unstructured) (bool, error) {
			return false, nil
		}

		transformer := func(_ context.Context, obj unstructured.Unstructured) (unstructured.Unstructured, error) {
			labels := obj.GetLabels()
			if labels == nil {
				labels = make(map[string]string)
			}
			labels["transformed"] = labelValueTrue
			obj.SetLabels(labels)

			return obj, nil
		}

		result, err := pipeline.Apply(ctx, objects, []types.Filter{rejectAllFilter}, []types.Transformer{transformer})

		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeEmpty())
	})
}
