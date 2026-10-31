package pipeline_test

import (
	"context"
	"errors"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/k8s-manifest-kit/engine/pkg/pipeline"
	"github.com/k8s-manifest-kit/engine/pkg/transformer"
	"github.com/k8s-manifest-kit/engine/pkg/types"

	. "github.com/onsi/gomega"
)

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
