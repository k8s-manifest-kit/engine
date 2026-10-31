package engine_test

import (
	"testing"

	"github.com/k8s-manifest-kit/pkg/util/k8s"
	"github.com/stretchr/testify/mock"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	engine "github.com/k8s-manifest-kit/engine/pkg"
	"github.com/k8s-manifest-kit/engine/pkg/types"

	. "github.com/onsi/gomega"
)

func TestSourceAnnotations(t *testing.T) {

	t.Run("should render objects with source annotations when renderer has them enabled", func(t *testing.T) {
		g := NewWithT(t)
		pod := &corev1.Pod{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Pod",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-pod",
				Annotations: map[string]string{
					types.AnnotationSourceType: "test-renderer",
				},
			},
		}

		unstrPod, err := k8s.ToUnstructured(pod)
		g.Expect(err).ToNot(HaveOccurred())

		renderer := new(mockRenderer)
		renderer.On("Process", mock.Anything, mock.Anything).Return([]unstructured.Unstructured{*unstrPod}, nil)
		renderer.On("Name").Return("mock")

		e, err := engine.New(
			engine.WithRenderer(renderer),
		)
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := e.Render(t.Context())

		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(objects).Should(HaveLen(1))

		annotations := objects[0].GetAnnotations()
		g.Expect(annotations).Should(HaveKeyWithValue(types.AnnotationSourceType, "test-renderer"))
	})

	t.Run("should not have source annotations when renderer has them disabled", func(t *testing.T) {
		g := NewWithT(t)
		pod := &corev1.Pod{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Pod",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-pod",
			},
		}

		unstrPod, err := k8s.ToUnstructured(pod)
		g.Expect(err).ToNot(HaveOccurred())

		renderer := new(mockRenderer)
		renderer.On("Process", mock.Anything, mock.Anything).Return([]unstructured.Unstructured{*unstrPod}, nil)
		renderer.On("Name").Return("mock")

		e, err := engine.New(
			engine.WithRenderer(renderer),
		)
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := e.Render(t.Context())

		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(objects).Should(HaveLen(1))

		annotations := objects[0].GetAnnotations()
		g.Expect(annotations).ShouldNot(HaveKey(types.AnnotationSourceType))
	})

	t.Run("should work with multiple renderers with different annotation settings", func(t *testing.T) {
		g := NewWithT(t)
		pod1 := &corev1.Pod{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Pod",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "pod-with-annotations",
			},
		}

		pod2 := &corev1.Pod{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Pod",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "pod-without-annotations",
			},
		}

		pod1.Annotations = map[string]string{
			types.AnnotationSourceType: "test-renderer-1",
		}

		unstrPod1, err := k8s.ToUnstructured(pod1)
		g.Expect(err).ToNot(HaveOccurred())

		unstrPod2, err := k8s.ToUnstructured(pod2)
		g.Expect(err).ToNot(HaveOccurred())

		renderer1 := new(mockRenderer)
		renderer1.On("Process", mock.Anything, mock.Anything).Return([]unstructured.Unstructured{*unstrPod1}, nil)
		renderer1.On("Name").Return("mock")

		renderer2 := new(mockRenderer)
		renderer2.On("Process", mock.Anything, mock.Anything).Return([]unstructured.Unstructured{*unstrPod2}, nil)
		renderer2.On("Name").Return("mock")

		e, err := engine.New(
			engine.WithRenderer(renderer1),
			engine.WithRenderer(renderer2),
		)
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := e.Render(t.Context())

		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(objects).Should(HaveLen(2))

		for _, obj := range objects {
			annotations := obj.GetAnnotations()
			if obj.GetName() == "pod-with-annotations" {
				g.Expect(annotations).Should(HaveKeyWithValue(types.AnnotationSourceType, "test-renderer-1"))
			} else if obj.GetName() == "pod-without-annotations" {
				g.Expect(annotations).ShouldNot(HaveKey(types.AnnotationSourceType))
			}
		}
	})
}
