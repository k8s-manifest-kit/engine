package annotations

import (
	"context"
	"maps"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/k8s-manifest-kit/engine/pkg/types"
)

// Set returns a transformer that adds or updates annotations on objects.
func Set(annotationsToApply map[string]string) types.Transformer {
	return func(_ context.Context, obj unstructured.Unstructured) (unstructured.Unstructured, error) {
		values := obj.GetAnnotations()
		if values == nil {
			values = make(map[string]string)
		}

		maps.Copy(values, annotationsToApply)

		obj.SetAnnotations(values)

		return obj, nil
	}
}

// Remove returns a transformer that removes specific annotations from objects.
func Remove(keys ...string) types.Transformer {
	return func(_ context.Context, obj unstructured.Unstructured) (unstructured.Unstructured, error) {
		values := obj.GetAnnotations()
		if values == nil {
			return obj, nil
		}

		for _, key := range keys {
			delete(values, key)
		}

		obj.SetAnnotations(values)

		return obj, nil
	}
}

// RemoveIf returns a transformer that removes annotations matching a predicate.
func RemoveIf(predicate func(key string, value string) bool) types.Transformer {
	return func(_ context.Context, obj unstructured.Unstructured) (unstructured.Unstructured, error) {
		values := obj.GetAnnotations()
		if values == nil {
			return obj, nil
		}

		for k, v := range values {
			if predicate(k, v) {
				delete(values, k)
			}
		}

		obj.SetAnnotations(values)

		return obj, nil
	}
}
