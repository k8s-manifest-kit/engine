package types_test

import (
	"testing"

	"github.com/k8s-manifest-kit/pkg/util/k8s"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/k8s-manifest-kit/engine/pkg/types"

	. "github.com/onsi/gomega"
)

func TestAnnotationKeys(t *testing.T) {
	g := NewWithT(t)

	g.Expect(types.AnnotationSourceType).Should(Equal("manifests.k8s-manifests-kit/source.type"))
	g.Expect(types.AnnotationSourcePath).Should(Equal("manifests.k8s-manifests-kit/source.path"))
	g.Expect(types.AnnotationSourceFile).Should(Equal("manifests.k8s-manifests-kit/source.file"))
	g.Expect(types.AnnotationContentHash).Should(Equal("manifests.k8s-manifests-kit/content.hash"))
}

func TestSetContentHash(t *testing.T) {
	t.Run("computes and sets the annotation on the object", func(t *testing.T) {
		g := NewWithT(t)

		obj := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]any{
					"name": "test",
				},
				"data": map[string]any{
					"key": "value",
				},
			},
		}

		expectedHash := k8s.ContentHash(obj)

		types.SetContentHash(obj)

		annotations := obj.GetAnnotations()
		g.Expect(annotations).Should(HaveKeyWithValue(types.AnnotationContentHash, expectedHash))
	})

	t.Run("preserves existing annotations", func(t *testing.T) {
		g := NewWithT(t)

		obj := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]any{
					"name": "test",
					"annotations": map[string]any{
						"existing": "annotation",
					},
				},
			},
		}

		types.SetContentHash(obj)

		annotations := obj.GetAnnotations()
		g.Expect(annotations).Should(HaveKeyWithValue("existing", "annotation"))
		g.Expect(annotations).Should(HaveKey(types.AnnotationContentHash))
	})

	t.Run("creates annotations map when none exists", func(t *testing.T) {
		g := NewWithT(t)

		obj := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]any{
					"name": "test",
				},
			},
		}

		types.SetContentHash(obj)

		annotations := obj.GetAnnotations()
		g.Expect(annotations).Should(HaveKey(types.AnnotationContentHash))
	})
}
