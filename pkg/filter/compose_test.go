package filter_test

import (
	"context"
	"errors"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/k8s-manifest-kit/engine/pkg/filter"
	"github.com/k8s-manifest-kit/engine/pkg/types"

	. "github.com/onsi/gomega"
)

func TestOr(t *testing.T) {
	g := NewWithT(t)

	t.Run("should pass if any filter passes", func(t *testing.T) {
		f := filter.Or(
			alwaysFalse(),
			alwaysTrue(),
			alwaysFalse(),
		)

		ok, err := f(t.Context(), makePod("test"))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeTrue())
	})

	t.Run("should fail if all filters fail", func(t *testing.T) {
		f := filter.Or(
			alwaysFalse(),
			alwaysFalse(),
		)

		ok, err := f(t.Context(), makePod("test"))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeFalse())
	})

	t.Run("should pass with no filters", func(t *testing.T) {
		f := filter.Or()

		ok, err := f(t.Context(), makePod("test"))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeTrue())
	})

	t.Run("should return error from filter", func(t *testing.T) {
		f := filter.Or(
			alwaysFalse(),
			alwaysError(),
		)

		ok, err := f(t.Context(), makePod("test"))
		g.Expect(err).Should(HaveOccurred())
		g.Expect(ok).Should(BeFalse())
	})

	t.Run("should short-circuit on first true", func(t *testing.T) {
		callCount := 0
		counting := func(_ context.Context, _ unstructured.Unstructured) (bool, error) {
			callCount++

			return false, nil
		}

		f := filter.Or(
			counting,
			alwaysTrue(),
			counting, // Should not be called
		)

		ok, err := f(t.Context(), makePod("test"))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeTrue())
		g.Expect(callCount).Should(Equal(1))
	})
}

func TestAnd(t *testing.T) {
	g := NewWithT(t)

	t.Run("should pass if all filters pass", func(t *testing.T) {
		f := filter.And(
			alwaysTrue(),
			alwaysTrue(),
			alwaysTrue(),
		)

		ok, err := f(t.Context(), makePod("test"))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeTrue())
	})

	t.Run("should fail if any filter fails", func(t *testing.T) {
		f := filter.And(
			alwaysTrue(),
			alwaysFalse(),
			alwaysTrue(),
		)

		ok, err := f(t.Context(), makePod("test"))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeFalse())
	})

	t.Run("should pass with no filters", func(t *testing.T) {
		f := filter.And()

		ok, err := f(t.Context(), makePod("test"))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeTrue())
	})

	t.Run("should return error from filter", func(t *testing.T) {
		f := filter.And(
			alwaysTrue(),
			alwaysError(),
		)

		ok, err := f(t.Context(), makePod("test"))
		g.Expect(err).Should(HaveOccurred())
		g.Expect(ok).Should(BeFalse())
	})

	t.Run("should short-circuit on first false", func(t *testing.T) {
		callCount := 0
		counting := func(_ context.Context, _ unstructured.Unstructured) (bool, error) {
			callCount++

			return true, nil
		}

		f := filter.And(
			counting,
			alwaysFalse(),
			counting, // Should not be called
		)

		ok, err := f(t.Context(), makePod("test"))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeFalse())
		g.Expect(callCount).Should(Equal(1))
	})
}

func TestNot(t *testing.T) {
	g := NewWithT(t)

	t.Run("should invert true to false", func(t *testing.T) {
		f := filter.Not(alwaysTrue())

		ok, err := f(t.Context(), makePod("test"))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeFalse())
	})

	t.Run("should invert false to true", func(t *testing.T) {
		f := filter.Not(alwaysFalse())

		ok, err := f(t.Context(), makePod("test"))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeTrue())
	})

	t.Run("should propagate error", func(t *testing.T) {
		f := filter.Not(alwaysError())

		ok, err := f(t.Context(), makePod("test"))
		g.Expect(err).Should(HaveOccurred())
		g.Expect(ok).Should(BeFalse())
	})
}

func TestIf(t *testing.T) {
	g := NewWithT(t)

	t.Run("should apply then filter when condition passes", func(t *testing.T) {
		f := filter.If(
			alwaysTrue(),  // condition
			alwaysFalse(), // then
		)

		ok, err := f(t.Context(), makePod("test"))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeFalse())
	})

	t.Run("should pass when condition fails", func(t *testing.T) {
		f := filter.If(
			alwaysFalse(), // condition
			alwaysFalse(), // then (not executed)
		)

		ok, err := f(t.Context(), makePod("test"))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeTrue())
	})

	t.Run("should propagate error from condition", func(t *testing.T) {
		f := filter.If(
			alwaysError(), // condition
			alwaysTrue(),  // then (not executed)
		)

		ok, err := f(t.Context(), makePod("test"))
		g.Expect(err).Should(HaveOccurred())
		g.Expect(ok).Should(BeFalse())
	})

	t.Run("should propagate error from then filter", func(t *testing.T) {
		f := filter.If(
			alwaysTrue(),  // condition
			alwaysError(), // then
		)

		ok, err := f(t.Context(), makePod("test"))
		g.Expect(err).Should(HaveOccurred())
		g.Expect(ok).Should(BeFalse())
	})
}

func TestNestedComposition(t *testing.T) {
	g := NewWithT(t)

	t.Run("should support Or(And(...), ...)", func(t *testing.T) {
		f := filter.Or(
			filter.And(
				alwaysTrue(),
				alwaysFalse(), // Makes And false
			),
			alwaysTrue(), // Or passes
		)

		ok, err := f(t.Context(), makePod("test"))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeTrue())
	})

	t.Run("should support And(Or(...), Not(...))", func(t *testing.T) {
		f := filter.And(
			filter.Or(
				alwaysFalse(),
				alwaysTrue(), // Or passes
			),
			filter.Not(alwaysFalse()), // Not(false) = true
		)

		ok, err := f(t.Context(), makePod("test"))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeTrue())
	})

	t.Run("should support complex nested composition", func(t *testing.T) {
		// (A AND B) OR (C AND NOT D)
		f := filter.Or(
			filter.And(
				alwaysTrue(), // A
				alwaysTrue(), // B
			),
			filter.And(
				alwaysFalse(),             // C
				filter.Not(alwaysFalse()), // NOT D
			),
		)

		ok, err := f(t.Context(), makePod("test"))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ok).Should(BeTrue())
	})
}

// Helper functions

//nolint:unparam // Test helper needs consistent signature
func makePod(name string) unstructured.Unstructured {
	obj := unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]any{
				"name": name,
			},
		},
	}
	obj.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Pod"))

	return obj
}

func alwaysTrue() types.Filter {
	return func(_ context.Context, _ unstructured.Unstructured) (bool, error) {
		return true, nil
	}
}

func alwaysFalse() types.Filter {
	return func(_ context.Context, _ unstructured.Unstructured) (bool, error) {
		return false, nil
	}
}

func alwaysError() types.Filter {
	return func(_ context.Context, _ unstructured.Unstructured) (bool, error) {
		return false, errors.New("filter error")
	}
}
