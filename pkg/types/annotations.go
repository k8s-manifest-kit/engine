package types

import (
	"github.com/k8s-manifest-kit/pkg/util/k8s"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	// AnnotationSourceType is the annotation key for the renderer type.
	AnnotationSourceType = "manifests.k8s-manifests-kit/source.type"

	// AnnotationSourcePath is the annotation key for the source path/chart identifier.
	AnnotationSourcePath = "manifests.k8s-manifests-kit/source.path"

	// AnnotationSourceFile is the annotation key for the specific template file.
	AnnotationSourceFile = "manifests.k8s-manifests-kit/source.file"

	// AnnotationContentHash is the annotation key for the SHA-256 content hash of the rendered resource.
	AnnotationContentHash = "manifests.k8s-manifests-kit/content.hash"

	// AnnotationRenderOrigin is the annotation key for the origin of a rendered resource.
	AnnotationRenderOrigin = "manifests.k8s-manifests-kit/render.origin"

	// RenderOriginLive indicates that the source renderer executed for the current render.
	RenderOriginLive = "live"

	// RenderOriginCache indicates that the source result was returned from the render cache.
	RenderOriginCache = "cache"
)

// SetContentHash computes a content hash for the given object and stores it
// as the AnnotationContentHash annotation.
//
// The hash is deterministic for an unchanged object, but it covers the
// complete object as it stands at invocation — including any prior hash,
// origin, or source annotations already present. Setting a hash therefore
// mutates the object and is not idempotent: calling it again includes the
// previous hash annotation in the hashed content and produces a different
// value. Origin-only annotation calls never rehash. The full batch-phase
// ordering contract is documented on AnnotateObjects.
func SetContentHash(obj k8s.Object) {
	k8s.SetAnnotation(obj, AnnotationContentHash, k8s.ContentHash(obj))
}

// SetRenderOrigin stores the origin of a rendered resource as an annotation.
func SetRenderOrigin(obj k8s.Object, origin string) {
	k8s.SetAnnotation(obj, AnnotationRenderOrigin, origin)
}

// AnnotateObjects applies source annotations, content hashes, and render
// origins to a batch of objects in place. Objects are caller-owned: the
// function performs no cloning and returns no result. Nil and empty batches
// are safe.
//
// Options configure one invocation: they are applied once to fresh per-call
// state, then the phases run. The source, hash, and origin phases are
// independent and always execute in that order, regardless of the order of
// the option arguments. Each phase runs across the whole batch before the
// next phase starts, and processes objects in slice order within a phase.
//
// Defaults perform no writes: a nil or empty source-annotation map, a false
// content-hash flag, and an empty render origin each skip their phase,
// including allocation of an object's annotations map. Source-annotation
// values are written when their keys are explicitly present, so an empty
// value overwrites an existing one and omitted keys are left untouched.
//
// Hashing covers the complete object as it stands at that point in the phase
// order, including previously applied source annotations and any
// pre-existing annotations, and is not idempotent: re-running the hash phase
// over an already annotated object produces a different value. Origin-only
// invocations never rehash or overwrite source fields.
//
// Options created by this package snapshot their input and are safe to reuse
// across calls and goroutines, provided the objects of each call are not
// mutated concurrently. Custom AnnotationOption implementations own their
// aliasing and concurrency safety. The batch is not deduplicated: objects
// sharing an underlying map are processed repeatedly in the documented order.
func AnnotateObjects(objects []unstructured.Unstructured, opts ...AnnotationOption) {
	cfg := AnnotationOptions{}
	for _, opt := range opts {
		opt.ApplyTo(&cfg)
	}

	if len(cfg.SourceAnnotations) > 0 {
		for i := range objects {
			k8s.SetAnnotations(&objects[i], cfg.SourceAnnotations)
		}
	}

	if cfg.ContentHash {
		for i := range objects {
			SetContentHash(&objects[i])
		}
	}

	if cfg.RenderOrigin != "" {
		for i := range objects {
			SetRenderOrigin(&objects[i], cfg.RenderOrigin)
		}
	}
}
