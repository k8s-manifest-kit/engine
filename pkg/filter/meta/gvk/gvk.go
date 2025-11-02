package gvk

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/k8s-manifest-kit/engine/pkg/types"
)

// Filter creates a new filter function that filters objects based on their GroupVersionKind.
// An object is kept if its GVK matches any of the provided GVKs.
func Filter(gvks ...schema.GroupVersionKind) types.Filter {
	s := sets.New(gvks...)

	return func(_ context.Context, object unstructured.Unstructured) (bool, error) {
		return s.Has(object.GetObjectKind().GroupVersionKind()), nil
	}
}
