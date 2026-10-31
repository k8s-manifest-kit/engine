package pipeline_test

import (
	"context"
	"errors"
	"testing"

	"github.com/k8s-manifest-kit/pkg/util/k8s"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/k8s-manifest-kit/engine/pkg/pipeline"
	"github.com/k8s-manifest-kit/engine/pkg/types"

	. "github.com/onsi/gomega"
)

func TestApplyPostRenderers(t *testing.T) {
	ctx := t.Context()

	t.Run("should return objects unchanged when no post-renderers", func(t *testing.T) {
		g := NewWithT(t)
		objects := []unstructured.Unstructured{
			makeObject("Pod", "pod1"),
			makeObject("Service", "svc1"),
		}

		result, err := pipeline.ApplyPostRenderers(ctx, objects, nil)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(HaveLen(2))
		g.Expect(result).To(Equal(objects))
	})

	t.Run("should apply single post-renderer", func(t *testing.T) {
		g := NewWithT(t)
		objects := []unstructured.Unstructured{
			makeObject("Pod", "pod1"),
			makeObject("Service", "svc1"),
		}

		addLabel := func(_ context.Context, objs []unstructured.Unstructured) ([]unstructured.Unstructured, error) {
			for i := range objs {
				k8s.SetLabel(&objs[i], "post-rendered", labelValueTrue)
			}

			return objs, nil
		}

		result, err := pipeline.ApplyPostRenderers(ctx, objects, []types.PostRenderer{addLabel})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(HaveLen(2))

		for _, obj := range result {
			g.Expect(obj.GetLabels()).To(HaveKeyWithValue("post-rendered", labelValueTrue))
		}
	})

	t.Run("should chain multiple post-renderers sequentially", func(t *testing.T) {
		g := NewWithT(t)
		objects := []unstructured.Unstructured{
			makeObject("Pod", "pod1"),
			makeObject("Service", "svc1"),
			makeObject("Deployment", "deploy1"),
		}

		keepPods := func(_ context.Context, objs []unstructured.Unstructured) ([]unstructured.Unstructured, error) {
			var result []unstructured.Unstructured
			for _, obj := range objs {
				if obj.GetKind() == kindPod {
					result = append(result, obj)
				}
			}

			return result, nil
		}

		addLabel := func(_ context.Context, objs []unstructured.Unstructured) ([]unstructured.Unstructured, error) {
			for i := range objs {
				k8s.SetLabel(&objs[i], "stage", "second")
			}

			return objs, nil
		}

		result, err := pipeline.ApplyPostRenderers(ctx, objects, []types.PostRenderer{keepPods, addLabel})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(HaveLen(1))
		g.Expect(result[0].GetKind()).To(Equal("Pod"))
		g.Expect(result[0].GetLabels()).To(HaveKeyWithValue("stage", "second"))
	})

	t.Run("should return error from failing post-renderer", func(t *testing.T) {
		g := NewWithT(t)
		objects := []unstructured.Unstructured{
			makeObject("Pod", "pod1"),
		}

		failing := func(_ context.Context, _ []unstructured.Unstructured) ([]unstructured.Unstructured, error) {
			return nil, errors.New("post-renderer failed")
		}

		result, err := pipeline.ApplyPostRenderers(ctx, objects, []types.PostRenderer{failing})
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("post-renderer failed"))
		g.Expect(result).To(BeNil())
	})

	t.Run("should stop on first error in chain", func(t *testing.T) {
		g := NewWithT(t)
		objects := []unstructured.Unstructured{
			makeObject("Pod", "pod1"),
		}

		secondCalled := false

		failing := func(_ context.Context, _ []unstructured.Unstructured) ([]unstructured.Unstructured, error) {
			return nil, errors.New("first failed")
		}

		second := func(_ context.Context, objs []unstructured.Unstructured) ([]unstructured.Unstructured, error) {
			secondCalled = true

			return objs, nil
		}

		result, err := pipeline.ApplyPostRenderers(ctx, objects, []types.PostRenderer{failing, second})
		g.Expect(err).To(HaveOccurred())
		g.Expect(result).To(BeNil())
		g.Expect(secondCalled).To(BeFalse())
	})

	t.Run("should handle empty objects slice", func(t *testing.T) {
		g := NewWithT(t)

		pr := func(_ context.Context, objs []unstructured.Unstructured) ([]unstructured.Unstructured, error) {
			return objs, nil
		}

		result, err := pipeline.ApplyPostRenderers(ctx, []unstructured.Unstructured{}, []types.PostRenderer{pr})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeEmpty())
	})
}
