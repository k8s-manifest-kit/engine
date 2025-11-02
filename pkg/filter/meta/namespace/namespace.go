package namespace

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/k8s-manifest-kit/engine/pkg/types"
)

// Filter returns a filter that keeps objects in the specified namespaces.
// Empty namespace matches cluster-scoped resources.
func Filter(namespaces ...string) types.Filter {
	allowed := sets.New(namespaces...)

	return func(_ context.Context, obj unstructured.Unstructured) (bool, error) {
		return allowed.Has(obj.GetNamespace()), nil
	}
}

// Exclude returns a filter that excludes objects from the specified namespaces.
func Exclude(namespaces ...string) types.Filter {
	excluded := sets.New(namespaces...)

	return func(_ context.Context, obj unstructured.Unstructured) (bool, error) {
		return !excluded.Has(obj.GetNamespace()), nil
	}
}
