package annotations

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/k8s-manifest-kit/engine/pkg/types"
)

// HasAnnotation returns a filter that keeps objects that have the specified annotation key.
func HasAnnotation(key string) types.Filter {
	return func(_ context.Context, obj unstructured.Unstructured) (bool, error) {
		objAnnotations := obj.GetAnnotations()
		_, ok := objAnnotations[key]

		return ok, nil
	}
}

// HasAnnotations returns a filter that keeps objects that have all specified annotation keys.
func HasAnnotations(keys ...string) types.Filter {
	return func(_ context.Context, obj unstructured.Unstructured) (bool, error) {
		objAnnotations := obj.GetAnnotations()
		for _, key := range keys {
			if _, ok := objAnnotations[key]; !ok {
				return false, nil
			}
		}

		return true, nil
	}
}

// MatchAnnotations returns a filter that keeps objects that have all matching annotation key-values.
func MatchAnnotations(matchAnnotations map[string]string) types.Filter {
	return func(_ context.Context, obj unstructured.Unstructured) (bool, error) {
		objAnnotations := obj.GetAnnotations()
		for key, value := range matchAnnotations {
			if objValue, ok := objAnnotations[key]; !ok || objValue != value {
				return false, nil
			}
		}

		return true, nil
	}
}
