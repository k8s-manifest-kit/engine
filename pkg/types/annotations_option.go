package types

import (
	"maps"

	"github.com/k8s-manifest-kit/pkg/util/option"
)

// AnnotationOptions holds the configuration applied by AnnotateObjects.
type AnnotationOptions struct {
	// SourceAnnotations are merged into each object's annotations during the
	// source phase. A nil or empty map skips the phase.
	SourceAnnotations map[string]string

	// ContentHash enables the content-hash annotation on each object during
	// the hash phase.
	ContentHash bool

	// RenderOrigin is the render-origin value applied to each object during
	// the origin phase. An empty value skips the phase.
	RenderOrigin string
}

// AnnotationOption is a generic option for AnnotationOptions.
type AnnotationOption = option.Option[AnnotationOptions]

// WithSourceAnnotations merges the given annotations into the source phase of
// AnnotateObjects. Repeated options merge, with later values winning on
// duplicate keys; an empty map merges nothing and cannot remove keys. The map
// is snapshotted when the option is created, so later mutation of the input
// map is not observed.
func WithSourceAnnotations(annotations map[string]string) AnnotationOption {
	snapshot := maps.Clone(annotations)

	return option.FunctionalOption[AnnotationOptions](func(o *AnnotationOptions) {
		if len(snapshot) == 0 {
			return
		}

		if o.SourceAnnotations == nil {
			o.SourceAnnotations = make(map[string]string, len(snapshot))
		}

		maps.Copy(o.SourceAnnotations, snapshot)
	})
}

// WithContentHash enables or disables the hash phase of AnnotateObjects.
func WithContentHash(enabled bool) AnnotationOption {
	return option.FunctionalOption[AnnotationOptions](func(o *AnnotationOptions) {
		o.ContentHash = enabled
	})
}

// WithRenderOrigin sets the render-origin value applied by AnnotateObjects.
// An empty origin disables the origin phase.
func WithRenderOrigin(origin string) AnnotationOption {
	return option.FunctionalOption[AnnotationOptions](func(o *AnnotationOptions) {
		o.RenderOrigin = origin
	})
}
