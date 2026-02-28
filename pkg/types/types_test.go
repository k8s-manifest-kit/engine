package types_test

import (
	"context"
	"errors"
	"testing"

	"github.com/k8s-manifest-kit/pkg/util/k8s"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/k8s-manifest-kit/engine/pkg/types"

	. "github.com/onsi/gomega"
)

const valueMutated = "mutated"

func TestValuesClone(t *testing.T) {
	t.Run("should create a shallow copy", func(t *testing.T) {
		g := NewWithT(t)

		original := types.Values{
			"key1": "value1",
			"key2": 42,
		}

		clone := original.Clone()

		clone["key1"] = "modified"
		clone["key3"] = "new"

		g.Expect(original).To(HaveKeyWithValue("key1", "value1"))
		g.Expect(original).ToNot(HaveKey("key3"))
	})

	t.Run("should return nil for nil Values", func(t *testing.T) {
		g := NewWithT(t)

		var v types.Values
		g.Expect(v.Clone()).To(BeNil())
	})

	t.Run("should share nested maps (shallow)", func(t *testing.T) {
		g := NewWithT(t)

		nested := map[string]any{"inner": "original"}
		original := types.Values{"nested": nested}
		clone := original.Clone()

		nested["inner"] = valueMutated

		clonedNested := clone["nested"].(map[string]any)
		g.Expect(clonedNested["inner"]).To(Equal(valueMutated))
	})
}

func TestValuesDeepClone(t *testing.T) {
	t.Run("should return nil for nil Values", func(t *testing.T) {
		g := NewWithT(t)

		var v types.Values
		g.Expect(v.DeepClone()).To(BeNil())
	})

	t.Run("should deeply copy nested maps", func(t *testing.T) {
		g := NewWithT(t)

		original := types.Values{
			"top": "value",
			"nested": map[string]any{
				"level2": map[string]any{
					"level3": "deep",
				},
			},
		}

		clone := original.DeepClone()

		nestedClone := clone["nested"].(map[string]any)
		level2Clone := nestedClone["level2"].(map[string]any)
		level2Clone["level3"] = valueMutated

		nestedOrig := original["nested"].(map[string]any)
		level2Orig := nestedOrig["level2"].(map[string]any)
		g.Expect(level2Orig["level3"]).To(Equal("deep"))
	})

	t.Run("should deeply copy []any slices", func(t *testing.T) {
		g := NewWithT(t)

		original := types.Values{
			"items": []any{
				map[string]any{"name": "a"},
				map[string]any{"name": "b"},
			},
		}

		clone := original.DeepClone()

		clonedItems := clone["items"].([]any)
		clonedItems[0].(map[string]any)["name"] = valueMutated

		originalItems := original["items"].([]any)
		g.Expect(originalItems[0].(map[string]any)["name"]).To(Equal("a"))
	})

	t.Run("should deeply copy typed slices", func(t *testing.T) {
		g := NewWithT(t)

		original := types.Values{
			"strings": []string{"a", "b", "c"},
			"ints":    []int{1, 2, 3},
			"floats":  []float64{1.1, 2.2},
			"bools":   []bool{true, false},
		}

		clone := original.DeepClone()

		clone["strings"].([]string)[0] = valueMutated
		clone["ints"].([]int)[0] = 999
		clone["floats"].([]float64)[0] = 9.9
		clone["bools"].([]bool)[0] = false

		g.Expect(original["strings"].([]string)[0]).To(Equal("a"))
		g.Expect(original["ints"].([]int)[0]).To(Equal(1))
		g.Expect(original["floats"].([]float64)[0]).To(Equal(1.1))
		g.Expect(original["bools"].([]bool)[0]).To(BeTrue())
	})

	t.Run("should handle empty Values", func(t *testing.T) {
		g := NewWithT(t)

		original := types.Values{}
		clone := original.DeepClone()
		g.Expect(clone).ToNot(BeNil())
		g.Expect(clone).To(BeEmpty())
	})
}

func TestFilterAsPostRenderer(t *testing.T) {
	ctx := t.Context()

	t.Run("should keep matching objects and discard non-matching", func(t *testing.T) {
		g := NewWithT(t)
		objects := []unstructured.Unstructured{
			makeObj("Pod", "pod1"),
			makeObj("Service", "svc1"),
			makeObj("Pod", "pod2"),
		}

		podFilter := func(_ context.Context, obj unstructured.Unstructured) (bool, error) {
			return obj.GetKind() == "Pod", nil
		}

		pr := types.FilterAsPostRenderer(podFilter)
		result, err := pr(ctx, objects)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(HaveLen(2))
		g.Expect(result[0].GetName()).To(Equal("pod1"))
		g.Expect(result[1].GetName()).To(Equal("pod2"))
	})

	t.Run("should propagate filter error", func(t *testing.T) {
		g := NewWithT(t)
		objects := []unstructured.Unstructured{makeObj("Pod", "pod1")}

		failing := func(_ context.Context, _ unstructured.Unstructured) (bool, error) {
			return false, errors.New("filter boom")
		}

		pr := types.FilterAsPostRenderer(failing)
		result, err := pr(ctx, objects)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("filter boom"))
		g.Expect(result).To(BeNil())
	})

	t.Run("should handle all objects rejected", func(t *testing.T) {
		g := NewWithT(t)
		objects := []unstructured.Unstructured{
			makeObj("Pod", "pod1"),
			makeObj("Service", "svc1"),
		}

		rejectAll := func(_ context.Context, _ unstructured.Unstructured) (bool, error) {
			return false, nil
		}

		pr := types.FilterAsPostRenderer(rejectAll)
		result, err := pr(ctx, objects)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeEmpty())
	})
}

func TestTransformerAsPostRenderer(t *testing.T) {
	ctx := t.Context()

	t.Run("should transform all objects in batch", func(t *testing.T) {
		g := NewWithT(t)
		objects := []unstructured.Unstructured{
			makeObj("Pod", "pod1"),
			makeObj("Service", "svc1"),
		}

		addLabel := func(_ context.Context, obj unstructured.Unstructured) (unstructured.Unstructured, error) {
			k8s.SetLabel(&obj, "transformed", "yes")

			return obj, nil
		}

		pr := types.TransformerAsPostRenderer(addLabel)
		result, err := pr(ctx, objects)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(HaveLen(2))

		for _, obj := range result {
			g.Expect(obj.GetLabels()).To(HaveKeyWithValue("transformed", "yes"))
		}
	})

	t.Run("should propagate transformer error", func(t *testing.T) {
		g := NewWithT(t)
		objects := []unstructured.Unstructured{makeObj("Pod", "pod1")}

		failing := func(_ context.Context, _ unstructured.Unstructured) (unstructured.Unstructured, error) {
			return unstructured.Unstructured{}, errors.New("transformer boom")
		}

		pr := types.TransformerAsPostRenderer(failing)
		result, err := pr(ctx, objects)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("transformer boom"))
		g.Expect(result).To(BeNil())
	})
}

func TestBuildPostRendererChain(t *testing.T) {
	t.Run("should return nil when all inputs are empty", func(t *testing.T) {
		g := NewWithT(t)

		chain := types.BuildPostRendererChain(nil, nil, nil)
		g.Expect(chain).To(BeNil())
	})

	t.Run("should include only filters when no transformers or post-renderers", func(t *testing.T) {
		g := NewWithT(t)

		f := func(_ context.Context, _ unstructured.Unstructured) (bool, error) { return true, nil }
		chain := types.BuildPostRendererChain([]types.Filter{f, f}, nil, nil)
		g.Expect(chain).To(HaveLen(2))
	})

	t.Run("should include only post-renderers when no filters or transformers", func(t *testing.T) {
		g := NewWithT(t)

		pr := func(_ context.Context, objs []unstructured.Unstructured) ([]unstructured.Unstructured, error) {
			return objs, nil
		}
		chain := types.BuildPostRendererChain(nil, nil, []types.PostRenderer{pr})
		g.Expect(chain).To(HaveLen(1))
	})

	t.Run("should execute in order: filters then transformers then post-renderers", func(t *testing.T) {
		g := NewWithT(t)
		ctx := context.Background()

		// Each stage writes a marker annotation so we can verify order
		filterFunc := func(_ context.Context, obj unstructured.Unstructured) (bool, error) {
			k8s.SetAnnotation(&obj, "stage", "filter")

			return true, nil
		}

		transformerFunc := func(_ context.Context, obj unstructured.Unstructured) (unstructured.Unstructured, error) {
			g.Expect(obj.GetAnnotations()).To(HaveKeyWithValue("stage", "filter"))
			k8s.SetAnnotation(&obj, "stage", "transformer")

			return obj, nil
		}

		postRendererFunc := func(_ context.Context, objs []unstructured.Unstructured) ([]unstructured.Unstructured, error) {
			for i := range objs {
				g.Expect(objs[i].GetAnnotations()).To(HaveKeyWithValue("stage", "transformer"))
				k8s.SetAnnotation(&objs[i], "stage", "post-renderer")
			}

			return objs, nil
		}

		chain := types.BuildPostRendererChain(
			[]types.Filter{filterFunc},
			[]types.Transformer{transformerFunc},
			[]types.PostRenderer{postRendererFunc},
		)

		g.Expect(chain).To(HaveLen(3))

		objects := []unstructured.Unstructured{makeObj("Pod", "pod1")}
		for _, pr := range chain {
			var err error
			objects, err = pr(ctx, objects)
			g.Expect(err).ToNot(HaveOccurred())
		}

		g.Expect(objects[0].GetAnnotations()).To(HaveKeyWithValue("stage", "post-renderer"))
	})
}

func makeObj(kind string, name string) unstructured.Unstructured {
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
