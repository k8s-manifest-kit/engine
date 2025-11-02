package labels_test

import (
	"testing"

	jqmatcher "github.com/lburgazzoli/gomega-matchers/pkg/matchers/jq"
	"github.com/onsi/gomega/types"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/k8s-manifest-kit/engine/pkg/transformer/meta/labels"

	. "github.com/onsi/gomega"
)

func toUnstructured(t *testing.T, obj runtime.Object) unstructured.Unstructured {
	t.Helper()

	unstr, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	NewWithT(t).Expect(err).ShouldNot(HaveOccurred())

	return unstructured.Unstructured{Object: unstr}
}

func TestTransform(t *testing.T) {

	tests := []struct {
		name          string
		labelsToApply map[string]string
		inputObject   runtime.Object
		expected      types.GomegaMatcher
	}{
		{
			name:          "should add new labels to empty labels",
			labelsToApply: map[string]string{"key1": "value1", "key2": "value2"},
			inputObject: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{},
			},
			expected: And(
				jqmatcher.Match(`.metadata.labels["key1"] == "value1"`),
				jqmatcher.Match(`.metadata.labels["key2"] == "value2"`),
			),
		},
		{
			name:          "should merge with existing labels",
			labelsToApply: map[string]string{"key2": "new-value2", "key3": "value3"},
			inputObject: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"key1": "value1", "key2": "old-value2"},
				},
			},
			expected: And(
				jqmatcher.Match(`.metadata.labels["key1"] == "value1"`),
				jqmatcher.Match(`.metadata.labels["key2"] == "new-value2"`),
				jqmatcher.Match(`.metadata.labels["key3"] == "value3"`),
			),
		},
		{
			name:          "should handle nil labels map",
			labelsToApply: nil,
			inputObject: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"key1": "value1"},
				},
			},
			expected: jqmatcher.Match(`.metadata.labels["key1"] == "value1"`),
		},
		{
			name:          "should handle empty labels map",
			labelsToApply: map[string]string{},
			inputObject: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"key1": "value1"},
				},
			},
			expected: jqmatcher.Match(`.metadata.labels["key1"] == "value1"`),
		},
		{
			name:          "should handle object with nil metadata",
			labelsToApply: map[string]string{"key1": "value1"},
			inputObject:   &corev1.ConfigMap{},
			expected:      jqmatcher.Match(`.metadata.labels["key1"] == "value1"`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			transformer := labels.Set(tt.labelsToApply)
			unstrObj := toUnstructured(t, tt.inputObject)
			transformed, err := transformer(t.Context(), unstrObj)

			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(transformed.Object).To(tt.expected)
		})
	}
}

func TestRemove(t *testing.T) {

	t.Run("should remove specific labels", func(t *testing.T) {
		g := NewWithT(t)
		transformer := labels.Remove("key1", "key3")

		obj := toUnstructured(t, &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"key1": "value1",
					"key2": "value2",
					"key3": "value3",
				},
			},
		})

		transformed, err := transformer(t.Context(), obj)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(transformed.GetLabels()).Should(Equal(map[string]string{"key2": "value2"}))
	})

	t.Run("should handle removing non-existent labels", func(t *testing.T) {
		g := NewWithT(t)
		transformer := labels.Remove("missing")

		obj := toUnstructured(t, &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{"key1": "value1"},
			},
		})

		transformed, err := transformer(t.Context(), obj)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(transformed.GetLabels()).Should(Equal(map[string]string{"key1": "value1"}))
	})

	t.Run("should handle objects with no labels", func(t *testing.T) {
		g := NewWithT(t)
		transformer := labels.Remove("any")

		obj := toUnstructured(t, &corev1.ConfigMap{})

		transformed, err := transformer(t.Context(), obj)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(transformed.GetLabels()).Should(BeNil())
	})
}

func TestRemoveIf(t *testing.T) {

	t.Run("should remove labels matching predicate", func(t *testing.T) {
		g := NewWithT(t)
		transformer := labels.RemoveIf(func(key string, value string) bool {
			return key == "remove-me" || value == "delete"
		})

		obj := toUnstructured(t, &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"remove-me": "anything",
					"keep":      "delete",
					"preserve":  "value",
				},
			},
		})

		transformed, err := transformer(t.Context(), obj)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(transformed.GetLabels()).Should(Equal(map[string]string{"preserve": "value"}))
	})

	t.Run("should handle no matches", func(t *testing.T) {
		g := NewWithT(t)
		transformer := labels.RemoveIf(func(_ string, _ string) bool {
			return false
		})

		obj := toUnstructured(t, &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{"key": "value"},
			},
		})

		transformed, err := transformer(t.Context(), obj)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(transformed.GetLabels()).Should(Equal(map[string]string{"key": "value"}))
	})
}
