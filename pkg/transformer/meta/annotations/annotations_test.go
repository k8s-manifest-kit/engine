package annotations_test

import (
	"testing"

	jqmatcher "github.com/lburgazzoli/gomega-matchers/pkg/matchers/jq"
	"github.com/onsi/gomega/types"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/k8s-manifest-kit/engine/pkg/transformer/meta/annotations"

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
		name               string
		annotationsToApply map[string]string
		inputObject        runtime.Object
		expected           types.GomegaMatcher
	}{
		{
			name:               "should add new annotations to empty annotations",
			annotationsToApply: map[string]string{"key1": "value1", "key2": "value2"},
			inputObject: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{},
			},
			expected: And(
				jqmatcher.Match(`.metadata.annotations["key1"] == "value1"`),
				jqmatcher.Match(`.metadata.annotations["key2"] == "value2"`),
			),
		},
		{
			name:               "should merge with existing annotations",
			annotationsToApply: map[string]string{"key2": "new-value2", "key3": "value3"},
			inputObject: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{"key1": "value1", "key2": "old-value2"},
				},
			},
			expected: And(
				jqmatcher.Match(`.metadata.annotations["key1"] == "value1"`),
				jqmatcher.Match(`.metadata.annotations["key2"] == "new-value2"`),
				jqmatcher.Match(`.metadata.annotations["key3"] == "value3"`),
			),
		},
		{
			name:               "should handle nil annotations map",
			annotationsToApply: nil,
			inputObject: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{"key1": "value1"},
				},
			},
			expected: jqmatcher.Match(`.metadata.annotations["key1"] == "value1"`),
		},
		{
			name:               "should handle empty annotations map",
			annotationsToApply: map[string]string{},
			inputObject: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{"key1": "value1"},
				},
			},
			expected: jqmatcher.Match(`.metadata.annotations["key1"] == "value1"`),
		},
		{
			name:               "should handle object with nil metadata",
			annotationsToApply: map[string]string{"key1": "value1"},
			inputObject:        &corev1.ConfigMap{},
			expected:           jqmatcher.Match(`.metadata.annotations["key1"] == "value1"`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			transformer := annotations.Set(tt.annotationsToApply)
			unstrObj := toUnstructured(t, tt.inputObject)
			transformed, err := transformer(t.Context(), unstrObj)

			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(transformed.Object).To(tt.expected)
		})
	}
}

func TestRemove(t *testing.T) {

	t.Run("should remove specific annotations", func(t *testing.T) {
		g := NewWithT(t)
		transformer := annotations.Remove("ann1", "ann3")

		obj := toUnstructured(t, &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"ann1": "value1",
					"ann2": "value2",
					"ann3": "value3",
				},
			},
		})

		transformed, err := transformer(t.Context(), obj)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(transformed.GetAnnotations()).Should(Equal(map[string]string{"ann2": "value2"}))
	})

	t.Run("should handle removing non-existent annotations", func(t *testing.T) {
		g := NewWithT(t)
		transformer := annotations.Remove("missing")

		obj := toUnstructured(t, &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{"ann1": "value1"},
			},
		})

		transformed, err := transformer(t.Context(), obj)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(transformed.GetAnnotations()).Should(Equal(map[string]string{"ann1": "value1"}))
	})

	t.Run("should handle objects with no annotations", func(t *testing.T) {
		g := NewWithT(t)
		transformer := annotations.Remove("any")

		obj := toUnstructured(t, &corev1.ConfigMap{})

		transformed, err := transformer(t.Context(), obj)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(transformed.GetAnnotations()).Should(BeNil())
	})
}

func TestRemoveIf(t *testing.T) {

	t.Run("should remove annotations matching predicate", func(t *testing.T) {
		g := NewWithT(t)
		transformer := annotations.RemoveIf(func(key string, value string) bool {
			return key == "remove-me" || value == "delete"
		})

		obj := toUnstructured(t, &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"remove-me": "anything",
					"keep":      "delete",
					"preserve":  "value",
				},
			},
		})

		transformed, err := transformer(t.Context(), obj)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(transformed.GetAnnotations()).Should(Equal(map[string]string{"preserve": "value"}))
	})

	t.Run("should handle no matches", func(t *testing.T) {
		g := NewWithT(t)
		transformer := annotations.RemoveIf(func(_ string, _ string) bool {
			return false
		})

		obj := toUnstructured(t, &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{"key": "value"},
			},
		})

		transformed, err := transformer(t.Context(), obj)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(transformed.GetAnnotations()).Should(Equal(map[string]string{"key": "value"}))
	})
}
