package engine_test

import (
	"slices"
	"testing"

	"github.com/k8s-manifest-kit/pkg/util/k8s"
	"github.com/stretchr/testify/mock"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	engine "github.com/k8s-manifest-kit/engine/pkg"
	"github.com/k8s-manifest-kit/engine/pkg/render"
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

// deepCopyBatch returns deep copies of every object in the batch.
func deepCopyBatch(objects []unstructured.Unstructured) []unstructured.Unstructured {
	copies := make([]unstructured.Unstructured, 0, len(objects))
	for i := range objects {
		copies = append(copies, *objects[i].DeepCopy())
	}

	return copies
}

// expectAnnotatedBatch asserts that the batch is nonempty and that every
// object in it carries a nonempty content-hash annotation. The subject names
// the renderer and phase the assertion belongs to, for attributable failures.
func expectAnnotatedBatch(g *WithT, objects []unstructured.Unstructured, subject string) {
	g.Expect(objects).ToNot(BeEmpty(), "%s should be nonempty", subject)

	for i := range objects {
		hash, ok := objects[i].GetAnnotations()[types.AnnotationContentHash]
		g.Expect(ok).To(BeTrue(), "%s object %d should carry a content hash annotation", subject, i)
		g.Expect(hash).ToNot(BeEmpty(), "%s object %d should carry a nonempty content hash", subject, i)
	}
}

func TestRenderPreservesRendererContentHashes(t *testing.T) {

	t.Run("should preserve renderer hashes through aggregation and repeated renders", func(t *testing.T) {
		g := NewWithT(t)

		values := types.Values{"environment": "test"}

		firstSource := makePod("hash-pod-alpha")
		firstSourceSnapshot := []unstructured.Unstructured{*firstSource.DeepCopy()}
		first := newHashRenderer("hash-first", firstSource)

		secondSource := makeService()
		secondSourceSnapshot := []unstructured.Unstructured{*secondSource.DeepCopy()}
		second := newHashRenderer("hash-second", secondSource)

		firstBaseline, err := first.Process(t.Context(), values.DeepClone())
		g.Expect(err).ToNot(HaveOccurred(), "direct Process of the first renderer should succeed")

		secondBaseline, err := second.Process(t.Context(), values.DeepClone())
		g.Expect(err).ToNot(HaveOccurred(), "direct Process of the second renderer should succeed")

		expectAnnotatedBatch(g, firstBaseline, "first renderer baseline")
		expectAnnotatedBatch(g, secondBaseline, "second renderer baseline")

		firstBaselineSnapshot := deepCopyBatch(firstBaseline)
		secondBaselineSnapshot := deepCopyBatch(secondBaseline)

		expected := slices.Concat(firstBaseline, secondBaseline)

		e, err := engine.New(
			engine.WithRenderer(first),
			engine.WithRenderer(second),
		)
		g.Expect(err).ToNot(HaveOccurred())

		firstRender, err := e.Render(t.Context(), render.WithValues(values))
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(firstRender).To(
			Equal(expected),
			"first engine render should equal the ordered concatenation of both direct baselines",
		)

		secondRender, err := e.Render(t.Context(), render.WithValues(values))
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(secondRender).To(
			Equal(expected),
			"second engine render of unchanged inputs should equal the ordered concatenation of both direct baselines",
		)

		g.Expect(firstBaseline).To(
			Equal(firstBaselineSnapshot),
			"the first baseline must remain unchanged after engine renders",
		)
		g.Expect(secondBaseline).To(
			Equal(secondBaselineSnapshot),
			"the second baseline must remain unchanged after engine renders",
		)
		g.Expect(first.sources).To(
			Equal(firstSourceSnapshot),
			"the first renderer source must remain unchanged after engine renders",
		)
		g.Expect(second.sources).To(
			Equal(secondSourceSnapshot),
			"the second renderer source must remain unchanged after engine renders",
		)
	})

	t.Run("should keep the renderer hash when a transformer mutates object content", func(t *testing.T) {
		g := NewWithT(t)

		values := types.Values{"environment": "test"}

		source := makePod("hash-pod-transformed")
		sourceSnapshot := []unstructured.Unstructured{*source.DeepCopy()}
		renderer := newHashRenderer("hash-transform", source)

		baseline, err := renderer.Process(t.Context(), values.DeepClone())
		g.Expect(err).ToNot(HaveOccurred(), "direct Process of the renderer should succeed")
		expectAnnotatedBatch(g, baseline, "transform scenario baseline")

		preTransformHash, ok := baseline[0].GetAnnotations()[types.AnnotationContentHash]
		g.Expect(ok).To(BeTrue(), "the baseline object should carry a content hash annotation")
		g.Expect(preTransformHash).ToNot(BeEmpty(), "the baseline content hash should be nonempty")

		expected := *baseline[0].DeepCopy()
		expected.SetLabels(map[string]string{"mutated": "true"})

		e, err := engine.New(
			engine.WithRenderer(renderer),
			engine.WithTransformer(addLabels(map[string]string{"mutated": "true"})),
		)
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := e.Render(t.Context(), render.WithValues(values))
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(
			Equal([]unstructured.Unstructured{expected}),
			"transformed output should differ from the renderer baseline by the label mutation only",
		)

		transformedHash, ok := objects[0].GetAnnotations()[types.AnnotationContentHash]
		g.Expect(ok).To(BeTrue(), "the transformed object should keep its content hash annotation")
		g.Expect(transformedHash).To(
			Equal(preTransformHash),
			"the transformer must not change the renderer-generated content hash",
		)

		g.Expect(objects[0].GetLabels()).To(
			Equal(map[string]string{"mutated": "true"}),
			"the label mutation should be visible in the rendered output",
		)

		g.Expect(renderer.sources).To(
			Equal(sourceSnapshot),
			"the renderer source must remain unchanged after the transformed render",
		)
	})
}
