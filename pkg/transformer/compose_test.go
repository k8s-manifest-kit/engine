package transformer_test

import (
	"context"
	"errors"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/k8s-manifest-kit/engine/pkg/transformer"
	"github.com/k8s-manifest-kit/engine/pkg/types"

	. "github.com/onsi/gomega"
)

func TestChain(t *testing.T) {

	t.Run("should apply transformers in sequence", func(t *testing.T) {
		g := NewWithT(t)
		tr := transformer.Chain(
			setLabel("label1", "value1"),
			setLabel("label2", "value2"),
			setLabel("label3", "value3"),
		)

		obj, err := tr(t.Context(), makePod("test"))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(obj.GetLabels()).Should(HaveKeyWithValue("label1", "value1"))
		g.Expect(obj.GetLabels()).Should(HaveKeyWithValue("label2", "value2"))
		g.Expect(obj.GetLabels()).Should(HaveKeyWithValue("label3", "value3"))
	})

	t.Run("should return unchanged with no transformers", func(t *testing.T) {
		g := NewWithT(t)
		tr := transformer.Chain()

		original := makePod("test")
		obj, err := tr(t.Context(), original)
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(obj.GetName()).Should(Equal(original.GetName()))
	})

	t.Run("should propagate error from transformer", func(t *testing.T) {
		g := NewWithT(t)
		tr := transformer.Chain(
			setLabel("label1", "value1"),
			errorTransformer(),
		)

		obj, err := tr(t.Context(), makePod("test"))
		g.Expect(err).Should(HaveOccurred())
		g.Expect(obj.Object).Should(BeEmpty())
	})

	t.Run("should pass output of one transformer to next", func(t *testing.T) {
		g := NewWithT(t)
		tr := transformer.Chain(
			setLabel("count", "1"),
			func(_ context.Context, obj unstructured.Unstructured) (unstructured.Unstructured, error) {
				labels := obj.GetLabels()
				if labels["count"] == "1" {
					labels["count"] = "2"
					obj.SetLabels(labels)
				}

				return obj, nil
			},
		)

		obj, err := tr(t.Context(), makePod("test"))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(obj.GetLabels()).Should(HaveKeyWithValue("count", "2"))
	})
}

func TestIf(t *testing.T) {

	t.Run("should apply transformer when condition passes", func(t *testing.T) {
		g := NewWithT(t)
		tr := transformer.If(
			alwaysTrue(),
			setLabel("applied", "true"),
		)

		obj, err := tr(t.Context(), makePod("test"))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(obj.GetLabels()).Should(HaveKeyWithValue("applied", "true"))
	})

	t.Run("should not apply transformer when condition fails", func(t *testing.T) {
		g := NewWithT(t)
		tr := transformer.If(
			alwaysFalse(),
			setLabel("applied", "true"),
		)

		obj, err := tr(t.Context(), makePod("test"))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(obj.GetLabels()).ShouldNot(HaveKey("applied"))
	})

	t.Run("should propagate error from condition", func(t *testing.T) {
		g := NewWithT(t)
		tr := transformer.If(
			errorFilter(),
			setLabel("applied", "true"),
		)

		obj, err := tr(t.Context(), makePod("test"))
		g.Expect(err).Should(HaveOccurred())
		g.Expect(obj.Object).Should(BeEmpty())
	})

	t.Run("should propagate error from transformer", func(t *testing.T) {
		g := NewWithT(t)
		tr := transformer.If(
			alwaysTrue(),
			errorTransformer(),
		)

		obj, err := tr(t.Context(), makePod("test"))
		g.Expect(err).Should(HaveOccurred())
		g.Expect(obj.Object).Should(BeEmpty())
	})

	t.Run("should use filter to check object properties", func(t *testing.T) {
		g := NewWithT(t)
		isPod := func(_ context.Context, obj unstructured.Unstructured) (bool, error) {
			return obj.GetKind() == "Pod", nil //nolint:goconst // Test code
		}

		tr := transformer.If(
			isPod,
			setLabel("is-pod", "true"),
		)

		obj, err := tr(t.Context(), makePod("test"))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(obj.GetLabels()).Should(HaveKeyWithValue("is-pod", "true"))
	})
}

func TestSwitch(t *testing.T) {

	t.Run("should apply first matching case", func(t *testing.T) {
		g := NewWithT(t)
		tr := transformer.Switch(
			[]transformer.Case{
				{When: alwaysFalse(), Then: setLabel("case", "1")},
				{When: alwaysTrue(), Then: setLabel("case", "2")},
				{When: alwaysTrue(), Then: setLabel("case", "3")},
			},
			nil,
		)

		obj, err := tr(t.Context(), makePod("test"))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(obj.GetLabels()).Should(HaveKeyWithValue("case", "2"))
	})

	t.Run("should apply default when no cases match", func(t *testing.T) {
		g := NewWithT(t)
		tr := transformer.Switch(
			[]transformer.Case{
				{When: alwaysFalse(), Then: setLabel("case", "1")},
				{When: alwaysFalse(), Then: setLabel("case", "2")},
			},
			setLabel("case", "default"),
		)

		obj, err := tr(t.Context(), makePod("test"))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(obj.GetLabels()).Should(HaveKeyWithValue("case", "default"))
	})

	t.Run("should return unchanged when no cases match and no default", func(t *testing.T) {
		g := NewWithT(t)
		tr := transformer.Switch(
			[]transformer.Case{
				{When: alwaysFalse(), Then: setLabel("case", "1")},
			},
			nil,
		)

		obj, err := tr(t.Context(), makePod("test"))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(obj.GetLabels()).ShouldNot(HaveKey("case"))
	})

	t.Run("should propagate error from filter", func(t *testing.T) {
		g := NewWithT(t)
		tr := transformer.Switch(
			[]transformer.Case{
				{When: errorFilter(), Then: setLabel("case", "1")},
			},
			nil,
		)

		obj, err := tr(t.Context(), makePod("test"))
		g.Expect(err).Should(HaveOccurred())
		g.Expect(obj.Object).Should(BeEmpty())
	})

	t.Run("should propagate error from transformer", func(t *testing.T) {
		g := NewWithT(t)
		tr := transformer.Switch(
			[]transformer.Case{
				{When: alwaysTrue(), Then: errorTransformer()},
			},
			nil,
		)

		obj, err := tr(t.Context(), makePod("test"))
		g.Expect(err).Should(HaveOccurred())
		g.Expect(obj.Object).Should(BeEmpty())
	})

	t.Run("should handle real-world branching by kind", func(t *testing.T) {
		g := NewWithT(t)
		isPod := func(_ context.Context, obj unstructured.Unstructured) (bool, error) {
			return obj.GetKind() == "Pod", nil
		}
		isService := func(_ context.Context, obj unstructured.Unstructured) (bool, error) {
			return obj.GetKind() == "Service", nil
		}

		tr := transformer.Switch(
			[]transformer.Case{
				{When: isPod, Then: setLabel("resource-type", "pod")},
				{When: isService, Then: setLabel("resource-type", "service")},
			},
			setLabel("resource-type", "other"),
		)

		obj, err := tr(t.Context(), makePod("test"))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(obj.GetLabels()).Should(HaveKeyWithValue("resource-type", "pod"))
	})
}

func TestNestedComposition(t *testing.T) {

	t.Run("should support Chain(If(...), If(...))", func(t *testing.T) {
		g := NewWithT(t)
		tr := transformer.Chain(
			transformer.If(alwaysTrue(), setLabel("label1", "value1")),
			transformer.If(alwaysTrue(), setLabel("label2", "value2")),
		)

		obj, err := tr(t.Context(), makePod("test"))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(obj.GetLabels()).Should(HaveKeyWithValue("label1", "value1"))
		g.Expect(obj.GetLabels()).Should(HaveKeyWithValue("label2", "value2"))
	})

	t.Run("should support If with Switch", func(t *testing.T) {
		g := NewWithT(t)
		isPod := func(_ context.Context, obj unstructured.Unstructured) (bool, error) {
			return obj.GetKind() == "Pod", nil
		}

		tr := transformer.If(
			isPod,
			transformer.Switch(
				[]transformer.Case{
					{When: alwaysTrue(), Then: setLabel("pod-label", "true")},
				},
				nil,
			),
		)

		obj, err := tr(t.Context(), makePod("test"))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(obj.GetLabels()).Should(HaveKeyWithValue("pod-label", "true"))
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

func setLabel(key string, value string) types.Transformer {
	return func(_ context.Context, obj unstructured.Unstructured) (unstructured.Unstructured, error) {
		labels := obj.GetLabels()
		if labels == nil {
			labels = make(map[string]string)
		}
		labels[key] = value
		obj.SetLabels(labels)

		return obj, nil
	}
}

func errorTransformer() types.Transformer {
	return func(_ context.Context, _ unstructured.Unstructured) (unstructured.Unstructured, error) {
		return unstructured.Unstructured{}, errors.New("transformer error")
	}
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

func errorFilter() types.Filter {
	return func(_ context.Context, _ unstructured.Unstructured) (bool, error) {
		return false, errors.New("filter error")
	}
}
