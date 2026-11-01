package types_test

import (
	"fmt"
	"maps"
	"sync"
	"testing"

	"github.com/k8s-manifest-kit/pkg/util/k8s"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/k8s-manifest-kit/engine/pkg/types"

	. "github.com/onsi/gomega"
)

func newAnnotationObject(name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]any{
				"name": name,
			},
			"data": map[string]any{
				"key": "value",
			},
		},
	}
}

func newPreAnnotatedObject(name string) *unstructured.Unstructured {
	object := newAnnotationObject(name)
	object.SetAnnotations(map[string]string{
		"existing":                 "annotation",
		types.AnnotationSourcePath: "original/path",
	})

	return object
}

func sourceAnnotations() map[string]string {
	return map[string]string{
		types.AnnotationSourceType: "test-renderer",
		types.AnnotationSourcePath: "source/path",
	}
}

func expectedSourceHash(name string) string {
	expected := newAnnotationObject(name)
	expected.SetAnnotations(sourceAnnotations())

	return k8s.ContentHash(expected)
}

func TestAnnotateObjectsNoOp(t *testing.T) {
	t.Run("nil and empty batches are safe", func(t *testing.T) {
		g := NewWithT(t)

		g.Expect(func() {
			types.AnnotateObjects(nil)
			types.AnnotateObjects(nil, types.WithContentHash(true))
			types.AnnotateObjects([]unstructured.Unstructured{})
			types.AnnotateObjects([]unstructured.Unstructured{}, types.WithSourceAnnotations(sourceAnnotations()))
		}).NotTo(Panic())
	})

	t.Run("no options leave the object unchanged without allocating annotations", func(t *testing.T) {
		g := NewWithT(t)

		base := newAnnotationObject("config")
		objects := []unstructured.Unstructured{*base}

		types.AnnotateObjects(objects)

		g.Expect(objects[0]).To(Equal(*base))
		g.Expect(objects[0].GetAnnotations()).To(BeNil())
	})

	t.Run("default option values perform no writes", func(t *testing.T) {
		g := NewWithT(t)

		base := newAnnotationObject("config")
		objects := []unstructured.Unstructured{*base}

		types.AnnotateObjects(objects,
			types.WithSourceAnnotations(nil),
			types.WithSourceAnnotations(map[string]string{}),
			types.WithContentHash(false),
			types.WithRenderOrigin(""),
		)

		g.Expect(objects[0]).To(Equal(*base))
		g.Expect(objects[0].GetAnnotations()).To(BeNil())
	})
}

func TestAnnotateObjectsSourcePhase(t *testing.T) {
	t.Run("sets supplied keys on every object and preserves unrelated annotations", func(t *testing.T) {
		g := NewWithT(t)

		objects := []unstructured.Unstructured{
			*newAnnotationObject("first"),
			*newPreAnnotatedObject("second"),
		}

		types.AnnotateObjects(objects, types.WithSourceAnnotations(sourceAnnotations()))

		for i := range objects {
			annotations := objects[i].GetAnnotations()
			g.Expect(annotations).To(HaveKeyWithValue(types.AnnotationSourceType, "test-renderer"))
			g.Expect(annotations).To(HaveKeyWithValue(types.AnnotationSourcePath, "source/path"))
		}

		g.Expect(objects[0].GetAnnotations()).NotTo(HaveKey("existing"))
		g.Expect(objects[1].GetAnnotations()).To(HaveKeyWithValue("existing", "annotation"))
	})

	t.Run("explicit empty values overwrite and omitted keys are not created", func(t *testing.T) {
		g := NewWithT(t)

		objects := []unstructured.Unstructured{*newPreAnnotatedObject("config")}

		types.AnnotateObjects(objects, types.WithSourceAnnotations(map[string]string{
			types.AnnotationSourceType: "test-renderer",
			types.AnnotationSourcePath: "",
		}))

		annotations := objects[0].GetAnnotations()
		g.Expect(annotations).To(HaveKeyWithValue(types.AnnotationSourceType, "test-renderer"))
		g.Expect(annotations).To(HaveKeyWithValue(types.AnnotationSourcePath, ""))
		g.Expect(annotations).NotTo(HaveKey(types.AnnotationSourceFile))
		g.Expect(annotations).To(HaveKeyWithValue("existing", "annotation"))
	})

	t.Run("does not invent path or file keys", func(t *testing.T) {
		g := NewWithT(t)

		objects := []unstructured.Unstructured{*newAnnotationObject("config")}

		types.AnnotateObjects(objects, types.WithSourceAnnotations(map[string]string{
			types.AnnotationSourceType: "mem",
		}))

		annotations := objects[0].GetAnnotations()
		g.Expect(annotations).To(HaveLen(1))
		g.Expect(annotations).To(HaveKeyWithValue(types.AnnotationSourceType, "mem"))
		g.Expect(annotations).NotTo(HaveKey(types.AnnotationSourcePath))
		g.Expect(annotations).NotTo(HaveKey(types.AnnotationSourceFile))
	})
}

func TestAnnotateObjectsHashPhase(t *testing.T) {
	t.Run("computes the exact hash over the complete current object", func(t *testing.T) {
		g := NewWithT(t)

		objects := []unstructured.Unstructured{
			*newAnnotationObject("first"),
			*newAnnotationObject("second"),
		}

		types.AnnotateObjects(objects, types.WithContentHash(true))

		for i := range objects {
			expected := newAnnotationObject(objects[i].GetName())
			g.Expect(objects[i].GetAnnotations()).To(HaveKeyWithValue(
				types.AnnotationContentHash,
				k8s.ContentHash(expected),
			))
		}
	})

	t.Run("hash input includes pre-existing hash and origin annotations", func(t *testing.T) {
		g := NewWithT(t)

		preAnnotated := newAnnotationObject("config")
		preAnnotated.SetAnnotations(map[string]string{
			types.AnnotationContentHash:  "sha256:previous",
			types.AnnotationRenderOrigin: types.RenderOriginLive,
		})
		objects := []unstructured.Unstructured{*preAnnotated}

		types.AnnotateObjects(objects, types.WithContentHash(true))

		expected := newAnnotationObject("config")
		expected.SetAnnotations(map[string]string{
			types.AnnotationContentHash:  "sha256:previous",
			types.AnnotationRenderOrigin: types.RenderOriginLive,
		})

		g.Expect(objects[0].GetAnnotations()).To(HaveKeyWithValue(
			types.AnnotationContentHash,
			k8s.ContentHash(expected),
		))
		g.Expect(objects[0].GetAnnotations()).NotTo(HaveKeyWithValue(
			types.AnnotationContentHash,
			"sha256:previous",
		))
	})
}

func TestAnnotateObjectsOriginPhase(t *testing.T) {
	t.Run("origin-only sets the origin and preserves source fields and hash", func(t *testing.T) {
		g := NewWithT(t)

		hash := expectedSourceHash("config")

		preAnnotated := newAnnotationObject("config")
		preAnnotated.SetAnnotations(map[string]string{
			types.AnnotationSourceType:  "test-renderer",
			types.AnnotationSourcePath:  "source/path",
			types.AnnotationContentHash: hash,
		})
		objects := []unstructured.Unstructured{*preAnnotated}

		types.AnnotateObjects(objects, types.WithRenderOrigin(types.RenderOriginCache))

		annotations := objects[0].GetAnnotations()
		g.Expect(annotations).To(HaveKeyWithValue(types.AnnotationRenderOrigin, types.RenderOriginCache))
		g.Expect(annotations).To(HaveKeyWithValue(types.AnnotationSourceType, "test-renderer"))
		g.Expect(annotations).To(HaveKeyWithValue(types.AnnotationSourcePath, "source/path"))
		g.Expect(annotations).To(HaveKeyWithValue(types.AnnotationContentHash, hash))
	})
}

func TestAnnotateObjectsPhaseOrder(t *testing.T) {
	t.Run("combined phases run source then hash then origin regardless of argument order", func(t *testing.T) {
		g := NewWithT(t)

		objects := []unstructured.Unstructured{*newAnnotationObject("config")}

		types.AnnotateObjects(objects,
			types.WithRenderOrigin(types.RenderOriginLive),
			types.WithContentHash(true),
			types.WithSourceAnnotations(sourceAnnotations()),
		)

		annotations := objects[0].GetAnnotations()
		g.Expect(annotations).To(HaveKeyWithValue(types.AnnotationSourceType, "test-renderer"))
		g.Expect(annotations).To(HaveKeyWithValue(types.AnnotationSourcePath, "source/path"))
		g.Expect(annotations).To(HaveKeyWithValue(types.AnnotationContentHash, expectedSourceHash("config")))
		g.Expect(annotations).To(HaveKeyWithValue(types.AnnotationRenderOrigin, types.RenderOriginLive))
	})

	t.Run("each phase completes across the batch before the next phase starts", func(t *testing.T) {
		g := NewWithT(t)

		// Both slice elements share one underlying object map, so an
		// implementation that completed every phase per object would compute
		// the second element's hash after the origin annotation is present.
		shared := newAnnotationObject("shared")
		objects := []unstructured.Unstructured{*shared, *shared}

		types.AnnotateObjects(objects,
			types.WithSourceAnnotations(sourceAnnotations()),
			types.WithContentHash(true),
			types.WithRenderOrigin(types.RenderOriginLive),
		)

		expectedAfterSource := newAnnotationObject("shared")
		expectedAfterSource.SetAnnotations(sourceAnnotations())
		firstHash := k8s.ContentHash(expectedAfterSource)

		withFirstHash := maps.Clone(sourceAnnotations())
		withFirstHash[types.AnnotationContentHash] = firstHash

		expectedAfterFirstHash := newAnnotationObject("shared")
		expectedAfterFirstHash.SetAnnotations(withFirstHash)
		secondHash := k8s.ContentHash(expectedAfterFirstHash)

		withOrigin := maps.Clone(withFirstHash)
		withOrigin[types.AnnotationRenderOrigin] = types.RenderOriginLive

		expectedWithOrigin := newAnnotationObject("shared")
		expectedWithOrigin.SetAnnotations(withOrigin)

		for i := range objects {
			annotations := objects[i].GetAnnotations()
			g.Expect(annotations).To(HaveKeyWithValue(types.AnnotationSourceType, "test-renderer"))
			g.Expect(annotations).To(HaveKeyWithValue(types.AnnotationContentHash, secondHash))
			g.Expect(annotations).To(HaveKeyWithValue(types.AnnotationRenderOrigin, types.RenderOriginLive))
		}

		g.Expect(secondHash).NotTo(Equal(k8s.ContentHash(expectedWithOrigin)))
	})
}

func TestAnnotationOptionMerging(t *testing.T) {
	t.Run("repeated source maps merge with later values winning on duplicate keys", func(t *testing.T) {
		g := NewWithT(t)

		objects := []unstructured.Unstructured{*newPreAnnotatedObject("config")}

		types.AnnotateObjects(objects,
			types.WithSourceAnnotations(map[string]string{
				types.AnnotationSourceType: "first-renderer",
				types.AnnotationSourcePath: "first/path",
			}),
			types.WithSourceAnnotations(map[string]string{
				types.AnnotationSourcePath: "second/path",
				types.AnnotationSourceFile: "file.yaml",
			}),
		)

		annotations := objects[0].GetAnnotations()
		g.Expect(annotations).To(HaveKeyWithValue(types.AnnotationSourceType, "first-renderer"))
		g.Expect(annotations).To(HaveKeyWithValue(types.AnnotationSourcePath, "second/path"))
		g.Expect(annotations).To(HaveKeyWithValue(types.AnnotationSourceFile, "file.yaml"))
		g.Expect(annotations).To(HaveKeyWithValue("existing", "annotation"))
	})

	t.Run("later scalar options win including disabling values", func(t *testing.T) {
		g := NewWithT(t)

		base := newAnnotationObject("config")
		objects := []unstructured.Unstructured{*base}

		types.AnnotateObjects(objects,
			types.WithContentHash(true),
			types.WithContentHash(false),
			types.WithRenderOrigin(types.RenderOriginLive),
			types.WithRenderOrigin(""),
		)

		g.Expect(objects[0]).To(Equal(*base))
		g.Expect(objects[0].GetAnnotations()).To(BeNil())
	})
}

func TestAnnotationOptionSnapshotAndReuse(t *testing.T) {
	t.Run("source maps are snapshotted at option creation", func(t *testing.T) {
		g := NewWithT(t)

		input := map[string]string{types.AnnotationSourceType: "test-renderer"}
		opt := types.WithSourceAnnotations(input)

		input[types.AnnotationSourcePath] = "mutated-after-creation"

		objects := []unstructured.Unstructured{*newAnnotationObject("config")}
		types.AnnotateObjects(objects, opt)

		annotations := objects[0].GetAnnotations()
		g.Expect(annotations).To(HaveKeyWithValue(types.AnnotationSourceType, "test-renderer"))
		g.Expect(annotations).NotTo(HaveKey(types.AnnotationSourcePath))
	})

	t.Run("options are reusable across independent calls without aliasing", func(t *testing.T) {
		g := NewWithT(t)

		opt := types.WithSourceAnnotations(sourceAnnotations())

		first := []unstructured.Unstructured{*newAnnotationObject("first")}
		second := []unstructured.Unstructured{*newAnnotationObject("second")}

		types.AnnotateObjects(first, opt)
		types.AnnotateObjects(second, opt)

		first[0].GetAnnotations()["mutated"] = "result-state"

		third := []unstructured.Unstructured{*newAnnotationObject("third")}
		types.AnnotateObjects(third, opt)

		annotations := third[0].GetAnnotations()
		g.Expect(annotations).To(HaveLen(2))
		g.Expect(annotations).To(HaveKeyWithValue(types.AnnotationSourceType, "test-renderer"))
		g.Expect(annotations).To(HaveKeyWithValue(types.AnnotationSourcePath, "source/path"))

		g.Expect(second[0].GetAnnotations()).NotTo(HaveKey("mutated"))
	})
}

func TestAnnotateObjectsConcurrentReuse(t *testing.T) {
	g := NewWithT(t)

	sourceOpt := types.WithSourceAnnotations(sourceAnnotations())
	hashOpt := types.WithContentHash(true)
	originOpt := types.WithRenderOrigin(types.RenderOriginLive)

	const workers = 16

	batch := make([]unstructured.Unstructured, workers)
	expectedHashes := make([]string, workers)

	for i := range workers {
		name := fmt.Sprintf("concurrent-%d", i)
		batch[i] = *newAnnotationObject(name)

		expected := newAnnotationObject(name)
		expected.SetAnnotations(sourceAnnotations())
		expectedHashes[i] = k8s.ContentHash(expected)
	}

	var wg sync.WaitGroup
	start := make(chan struct{})

	for i := range workers {
		wg.Go(func() {
			<-start
			types.AnnotateObjects(batch[i:i+1], sourceOpt, hashOpt, originOpt)
		})
	}

	close(start)
	wg.Wait()

	for i := range workers {
		annotations := batch[i].GetAnnotations()
		g.Expect(annotations).To(HaveKeyWithValue(types.AnnotationSourceType, "test-renderer"))
		g.Expect(annotations).To(HaveKeyWithValue(types.AnnotationSourcePath, "source/path"))
		g.Expect(annotations).To(HaveKeyWithValue(types.AnnotationContentHash, expectedHashes[i]))
		g.Expect(annotations).To(HaveKeyWithValue(types.AnnotationRenderOrigin, types.RenderOriginLive))
	}
}
