package pipeline_test

import (
	"context"
	"errors"
	"testing"

	"github.com/k8s-manifest-kit/pkg/util/k8s"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/k8s-manifest-kit/engine/pkg/filter"
	"github.com/k8s-manifest-kit/engine/pkg/pipeline"
	"github.com/k8s-manifest-kit/engine/pkg/transformer"
	"github.com/k8s-manifest-kit/engine/pkg/types"

	. "github.com/onsi/gomega"
)

const (
	kindPod        = "Pod"
	labelValueTrue = "true"
)

func TestApplyFilters(t *testing.T) {
	ctx := t.Context()

	t.Run("should return all objects when no filters", func(t *testing.T) {
		g := NewWithT(t)
		objects := []unstructured.Unstructured{
			makeObject("Pod", "pod1"),
			makeObject("Service", "svc1"),
		}

		result, err := pipeline.ApplyFilters(ctx, objects, nil)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(HaveLen(2))
		g.Expect(result).To(Equal(objects))
	})

	t.Run("should filter objects with single filter", func(t *testing.T) {
		g := NewWithT(t)
		objects := []unstructured.Unstructured{
			makeObject("Pod", "pod1"),
			makeObject("Service", "svc1"),
			makeObject("Pod", "pod2"),
		}

		podFilter := func(_ context.Context, obj unstructured.Unstructured) (bool, error) {
			return obj.GetKind() == kindPod, nil
		}

		result, err := pipeline.ApplyFilters(ctx, objects, []types.Filter{podFilter})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(HaveLen(2))
		g.Expect(result[0].GetKind()).To(Equal("Pod"))
		g.Expect(result[1].GetKind()).To(Equal("Pod"))
	})

	t.Run("should apply multiple filters with AND logic", func(t *testing.T) {
		g := NewWithT(t)
		objects := []unstructured.Unstructured{
			makeObjectWithNamespace("Pod", "pod1", "default"),
			makeObjectWithNamespace("Pod", "pod2", "kube-system"),
			makeObjectWithNamespace("Service", "svc1", "default"),
		}

		podFilter := func(_ context.Context, obj unstructured.Unstructured) (bool, error) {
			return obj.GetKind() == kindPod, nil
		}

		namespaceFilter := func(_ context.Context, obj unstructured.Unstructured) (bool, error) {
			return obj.GetNamespace() == "default", nil
		}

		result, err := pipeline.ApplyFilters(ctx, objects, []types.Filter{podFilter, namespaceFilter})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(HaveLen(1))
		g.Expect(result[0].GetKind()).To(Equal("Pod"))
		g.Expect(result[0].GetName()).To(Equal("pod1"))
		g.Expect(result[0].GetNamespace()).To(Equal("default"))
	})

	t.Run("should return error when filter fails", func(t *testing.T) {
		g := NewWithT(t)
		objects := []unstructured.Unstructured{
			makeObject("Pod", "pod1"),
		}

		errorFilter := func(_ context.Context, _ unstructured.Unstructured) (bool, error) {
			return false, errors.New("filter error")
		}

		result, err := pipeline.ApplyFilters(ctx, objects, []types.Filter{errorFilter})
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("filter error"))
		g.Expect(result).To(BeNil())
	})

	t.Run("should handle empty objects slice", func(t *testing.T) {
		g := NewWithT(t)
		objects := []unstructured.Unstructured{}

		filter := func(_ context.Context, obj unstructured.Unstructured) (bool, error) {
			return obj.GetKind() == kindPod, nil
		}

		result, err := pipeline.ApplyFilters(ctx, objects, []types.Filter{filter})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeEmpty())
	})

	t.Run("should reject all objects if any filter rejects", func(t *testing.T) {
		g := NewWithT(t)
		objects := []unstructured.Unstructured{
			makeObject("Pod", "pod1"),
		}

		acceptFilter := func(_ context.Context, _ unstructured.Unstructured) (bool, error) {
			return true, nil
		}

		rejectFilter := func(_ context.Context, _ unstructured.Unstructured) (bool, error) {
			return false, nil
		}

		result, err := pipeline.ApplyFilters(ctx, objects, []types.Filter{acceptFilter, rejectFilter})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeEmpty())
	})
}

func TestApplyTransformers(t *testing.T) {
	ctx := t.Context()

	t.Run("should return objects unchanged when no transformers", func(t *testing.T) {
		g := NewWithT(t)
		objects := []unstructured.Unstructured{
			makeObject("Pod", "pod1"),
			makeObject("Service", "svc1"),
		}

		result, err := pipeline.ApplyTransformers(ctx, objects, nil)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(HaveLen(2))
		g.Expect(result).To(Equal(objects))
	})

	t.Run("should apply single transformer", func(t *testing.T) {
		g := NewWithT(t)
		objects := []unstructured.Unstructured{
			makeObject("Pod", "pod1"),
		}

		addLabelTransformer := func(_ context.Context, obj unstructured.Unstructured) (unstructured.Unstructured, error) {
			labels := obj.GetLabels()
			if labels == nil {
				labels = make(map[string]string)
			}
			labels["env"] = "test"
			obj.SetLabels(labels)

			return obj, nil
		}

		result, err := pipeline.ApplyTransformers(ctx, objects, []types.Transformer{addLabelTransformer})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(HaveLen(1))
		g.Expect(result[0].GetLabels()).To(HaveKeyWithValue("env", "test"))
	})

	t.Run("should chain multiple transformers", func(t *testing.T) {
		g := NewWithT(t)
		objects := []unstructured.Unstructured{
			makeObject("Pod", "pod1"),
		}

		addLabel1 := func(_ context.Context, obj unstructured.Unstructured) (unstructured.Unstructured, error) {
			labels := obj.GetLabels()
			if labels == nil {
				labels = make(map[string]string)
			}
			labels["label1"] = "value1"
			obj.SetLabels(labels)

			return obj, nil
		}

		addLabel2 := func(_ context.Context, obj unstructured.Unstructured) (unstructured.Unstructured, error) {
			labels := obj.GetLabels()
			if labels == nil {
				labels = make(map[string]string)
			}
			labels["label2"] = "value2"
			obj.SetLabels(labels)

			return obj, nil
		}

		result, err := pipeline.ApplyTransformers(ctx, objects, []types.Transformer{addLabel1, addLabel2})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(HaveLen(1))
		g.Expect(result[0].GetLabels()).To(HaveKeyWithValue("label1", "value1"))
		g.Expect(result[0].GetLabels()).To(HaveKeyWithValue("label2", "value2"))
	})

	t.Run("should return error when transformer fails", func(t *testing.T) {
		g := NewWithT(t)
		objects := []unstructured.Unstructured{
			makeObject("Pod", "pod1"),
		}

		errorTransformer := func(_ context.Context, _ unstructured.Unstructured) (unstructured.Unstructured, error) {
			return unstructured.Unstructured{}, errors.New("transformer error")
		}

		result, err := pipeline.ApplyTransformers(ctx, objects, []types.Transformer{errorTransformer})
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("transformer error"))
		g.Expect(result).To(BeNil())
	})

	t.Run("should handle empty objects slice", func(t *testing.T) {
		g := NewWithT(t)
		objects := []unstructured.Unstructured{}

		transformer := func(_ context.Context, obj unstructured.Unstructured) (unstructured.Unstructured, error) {
			labels := obj.GetLabels()
			if labels == nil {
				labels = make(map[string]string)
			}
			labels["test"] = "value"
			obj.SetLabels(labels)

			return obj, nil
		}

		result, err := pipeline.ApplyTransformers(ctx, objects, []types.Transformer{transformer})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeEmpty())
	})

	t.Run("should stop on first transformer error", func(t *testing.T) {
		g := NewWithT(t)
		objects := []unstructured.Unstructured{
			makeObject("Pod", "pod1"),
		}

		successTransformer := func(_ context.Context, obj unstructured.Unstructured) (unstructured.Unstructured, error) {
			labels := obj.GetLabels()
			if labels == nil {
				labels = make(map[string]string)
			}
			labels["success"] = labelValueTrue
			obj.SetLabels(labels)

			return obj, nil
		}

		errorTransformer := func(_ context.Context, _ unstructured.Unstructured) (unstructured.Unstructured, error) {
			return unstructured.Unstructured{}, errors.New("second transformer failed")
		}

		result, err := pipeline.ApplyTransformers(ctx, objects, []types.Transformer{successTransformer, errorTransformer})
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("second transformer failed"))
		g.Expect(result).To(BeNil())
	})

	t.Run("should preserve transformer order", func(t *testing.T) {
		g := NewWithT(t)
		objects := []unstructured.Unstructured{
			makeObject("Pod", "pod1"),
		}

		setAnnotation := func(key string, value string) types.Transformer {
			return func(_ context.Context, obj unstructured.Unstructured) (unstructured.Unstructured, error) {
				annotations := obj.GetAnnotations()
				if annotations == nil {
					annotations = make(map[string]string)
				}
				annotations[key] = value
				obj.SetAnnotations(annotations)

				return obj, nil
			}
		}

		overwriteAnnotation := func(_ context.Context, obj unstructured.Unstructured) (unstructured.Unstructured, error) {
			annotations := obj.GetAnnotations()
			if annotations == nil {
				annotations = make(map[string]string)
			}
			annotations["key"] = "overwritten"
			obj.SetAnnotations(annotations)

			return obj, nil
		}

		result, err := pipeline.ApplyTransformers(ctx, objects, []types.Transformer{
			setAnnotation("key", "original"),
			overwriteAnnotation,
		})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(HaveLen(1))
		g.Expect(result[0].GetAnnotations()).To(HaveKeyWithValue("key", "overwritten"))
	})
}

func TestFilterError(t *testing.T) {
	ctx := t.Context()

	t.Run("should return Error with object and error context", func(t *testing.T) {
		g := NewWithT(t)
		objects := []unstructured.Unstructured{
			makeObjectWithNamespace("Pod", "pod1", "default"),
			makeObjectWithNamespace("Service", "svc1", "kube-system"),
		}

		failingFilter := func(_ context.Context, _ unstructured.Unstructured) (bool, error) {
			return false, errors.New("custom filter error")
		}

		result, err := pipeline.ApplyFilters(ctx, objects, []types.Filter{failingFilter})

		g.Expect(result).To(BeNil())
		g.Expect(err).To(HaveOccurred())

		var filterErr *filter.Error
		g.Expect(errors.As(err, &filterErr)).To(BeTrue())
		g.Expect(filterErr.Object.GetName()).To(Equal("pod1"))
		g.Expect(filterErr.Object.GetNamespace()).To(Equal("default"))
		g.Expect(filterErr.Err.Error()).To(Equal("custom filter error"))
	})

	t.Run("should wrap underlying error", func(t *testing.T) {
		g := NewWithT(t)
		objects := []unstructured.Unstructured{
			makeObject("Pod", "pod1"),
		}

		underlyingErr := errors.New("underlying error")
		failingFilter := func(_ context.Context, _ unstructured.Unstructured) (bool, error) {
			return false, underlyingErr
		}

		_, err := pipeline.ApplyFilters(ctx, objects, []types.Filter{failingFilter})

		g.Expect(err).To(HaveOccurred())

		var filterErr *filter.Error
		g.Expect(errors.As(err, &filterErr)).To(BeTrue())
		g.Expect(errors.Is(err, underlyingErr)).To(BeTrue())
	})

	t.Run("should not double-wrap Error", func(t *testing.T) {
		g := NewWithT(t)
		objects := []unstructured.Unstructured{
			makeObjectWithNamespace("Pod", "original-pod", "original-ns"),
		}

		// Filter that returns a Error with specific object context
		originalErr := errors.New("original error")
		failingFilter := func(_ context.Context, obj unstructured.Unstructured) (bool, error) {
			return false, &filter.Error{
				Object: obj,
				Err:    originalErr,
			}
		}

		_, err := pipeline.ApplyFilters(ctx, objects, []types.Filter{failingFilter})

		g.Expect(err).To(HaveOccurred())

		var filterErr *filter.Error
		g.Expect(errors.As(err, &filterErr)).To(BeTrue())
		// Should preserve the original object context, not double-wrap
		g.Expect(filterErr.Object.GetName()).To(Equal("original-pod"))
		g.Expect(filterErr.Object.GetNamespace()).To(Equal("original-ns"))
		g.Expect(filterErr.Err).To(Equal(originalErr))
		// The wrapped error should be the original error, not another Error
		g.Expect(filterErr.Err).ToNot(BeAssignableToTypeOf(&filter.Error{}))
	})
}

func TestTransformerError(t *testing.T) {
	ctx := t.Context()

	t.Run("should return Error with object and error context", func(t *testing.T) {
		g := NewWithT(t)
		objects := []unstructured.Unstructured{
			makeObjectWithNamespace("Pod", "pod1", "default"),
			makeObjectWithNamespace("Service", "svc1", "kube-system"),
		}

		failingTransformer := func(_ context.Context, _ unstructured.Unstructured) (unstructured.Unstructured, error) {
			return unstructured.Unstructured{}, errors.New("custom transformer error")
		}

		result, err := pipeline.ApplyTransformers(ctx, objects, []types.Transformer{failingTransformer})

		g.Expect(result).To(BeNil())
		g.Expect(err).To(HaveOccurred())

		var transformerErr *transformer.Error
		g.Expect(errors.As(err, &transformerErr)).To(BeTrue())
		g.Expect(transformerErr.Object.GetName()).To(Equal("pod1"))
		g.Expect(transformerErr.Object.GetNamespace()).To(Equal("default"))
		g.Expect(transformerErr.Err.Error()).To(Equal("custom transformer error"))
	})

	t.Run("should preserve object identity in error", func(t *testing.T) {
		g := NewWithT(t)
		objects := []unstructured.Unstructured{
			makeObject("Pod", "pod1"),
			makeObject("Service", "svc1"),
			makeObject("Deployment", "deploy1"),
		}

		failOnDeployment := func(_ context.Context, obj unstructured.Unstructured) (unstructured.Unstructured, error) {
			if obj.GetKind() == "Deployment" {
				return unstructured.Unstructured{}, errors.New("deployment transformation failed")
			}

			return obj, nil
		}

		result, err := pipeline.ApplyTransformers(ctx, objects, []types.Transformer{failOnDeployment})

		g.Expect(result).To(BeNil())
		g.Expect(err).To(HaveOccurred())

		var transformerErr *transformer.Error
		g.Expect(errors.As(err, &transformerErr)).To(BeTrue())
		g.Expect(transformerErr.Object.GetKind()).To(Equal("Deployment"))
		g.Expect(transformerErr.Object.GetName()).To(Equal("deploy1"))
	})

	t.Run("should wrap underlying error", func(t *testing.T) {
		g := NewWithT(t)
		objects := []unstructured.Unstructured{
			makeObject("Pod", "pod1"),
		}

		underlyingErr := errors.New("underlying error")
		failingTransformer := func(_ context.Context, _ unstructured.Unstructured) (unstructured.Unstructured, error) {
			return unstructured.Unstructured{}, underlyingErr
		}

		_, err := pipeline.ApplyTransformers(ctx, objects, []types.Transformer{failingTransformer})

		g.Expect(err).To(HaveOccurred())

		var transformerErr *transformer.Error
		g.Expect(errors.As(err, &transformerErr)).To(BeTrue())
		g.Expect(errors.Is(err, underlyingErr)).To(BeTrue())
	})

	t.Run("should not double-wrap Error", func(t *testing.T) {
		g := NewWithT(t)
		objects := []unstructured.Unstructured{
			makeObjectWithNamespace("Pod", "original-pod", "original-ns"),
		}

		// Transformer that returns a Error with specific object context
		originalErr := errors.New("original error")
		failingTransformer := func(_ context.Context, obj unstructured.Unstructured) (unstructured.Unstructured, error) {
			return unstructured.Unstructured{}, &transformer.Error{
				Object: obj,
				Err:    originalErr,
			}
		}

		_, err := pipeline.ApplyTransformers(ctx, objects, []types.Transformer{failingTransformer})

		g.Expect(err).To(HaveOccurred())

		var transformerErr *transformer.Error
		g.Expect(errors.As(err, &transformerErr)).To(BeTrue())
		// Should preserve the original object context, not double-wrap
		g.Expect(transformerErr.Object.GetName()).To(Equal("original-pod"))
		g.Expect(transformerErr.Object.GetNamespace()).To(Equal("original-ns"))
		g.Expect(transformerErr.Err).To(Equal(originalErr))
		// The wrapped error should be the original error, not another Error
		g.Expect(transformerErr.Err).ToNot(BeAssignableToTypeOf(&transformer.Error{}))
	})
}

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

func TestApplySourceSelectors(t *testing.T) {
	ctx := t.Context()

	t.Run("should return true when no selectors", func(t *testing.T) {
		g := NewWithT(t)

		ok, err := pipeline.ApplySourceSelectors(ctx, "any-source", nil)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(ok).To(BeTrue())
	})

	t.Run("should return true when selector accepts", func(t *testing.T) {
		g := NewWithT(t)

		accept := func(_ context.Context, _ types.Source) (bool, error) {
			return true, nil
		}

		ok, err := pipeline.ApplySourceSelectors(ctx, "source", []types.SourceSelector{accept})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(ok).To(BeTrue())
	})

	t.Run("should return false when selector rejects", func(t *testing.T) {
		g := NewWithT(t)

		reject := func(_ context.Context, _ types.Source) (bool, error) {
			return false, nil
		}

		ok, err := pipeline.ApplySourceSelectors(ctx, "source", []types.SourceSelector{reject})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(ok).To(BeFalse())
	})

	t.Run("should require all selectors to accept (AND logic)", func(t *testing.T) {
		g := NewWithT(t)

		accept := func(_ context.Context, _ types.Source) (bool, error) {
			return true, nil
		}

		reject := func(_ context.Context, _ types.Source) (bool, error) {
			return false, nil
		}

		ok, err := pipeline.ApplySourceSelectors(ctx, "source", []types.SourceSelector{accept, reject})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(ok).To(BeFalse())
	})

	t.Run("should short-circuit on first rejection", func(t *testing.T) {
		g := NewWithT(t)

		secondCalled := false

		reject := func(_ context.Context, _ types.Source) (bool, error) {
			return false, nil
		}

		second := func(_ context.Context, _ types.Source) (bool, error) {
			secondCalled = true

			return true, nil
		}

		ok, err := pipeline.ApplySourceSelectors(ctx, "source", []types.SourceSelector{reject, second})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(ok).To(BeFalse())
		g.Expect(secondCalled).To(BeFalse())
	})

	t.Run("should return error from failing selector", func(t *testing.T) {
		g := NewWithT(t)

		failing := func(_ context.Context, _ types.Source) (bool, error) {
			return false, errors.New("selector error")
		}

		ok, err := pipeline.ApplySourceSelectors(ctx, "source", []types.SourceSelector{failing})
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("selector error"))
		g.Expect(ok).To(BeFalse())
	})

	t.Run("should pass source to selector", func(t *testing.T) {
		g := NewWithT(t)

		type testSource struct{ Name string }

		src := testSource{Name: "my-source"}

		selector := func(_ context.Context, s types.Source) (bool, error) {
			ts, ok := s.(testSource)
			g.Expect(ok).To(BeTrue())
			g.Expect(ts.Name).To(Equal("my-source"))

			return true, nil
		}

		ok, err := pipeline.ApplySourceSelectors(ctx, src, []types.SourceSelector{selector})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(ok).To(BeTrue())
	})
}

// Helper functions

func makeObject(kind string, name string) unstructured.Unstructured {
	return unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       kind,
			"metadata": map[string]any{
				"name": name,
			},
		},
	}
}

func makeObjectWithNamespace(kind string, name string, namespace string) unstructured.Unstructured {
	return unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       kind,
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
			},
		},
	}
}
