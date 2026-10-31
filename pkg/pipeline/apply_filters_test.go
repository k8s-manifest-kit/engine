package pipeline_test

import (
	"context"
	"errors"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/k8s-manifest-kit/engine/pkg/filter"
	"github.com/k8s-manifest-kit/engine/pkg/pipeline"
	"github.com/k8s-manifest-kit/engine/pkg/types"

	. "github.com/onsi/gomega"
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
