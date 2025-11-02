package labels

import (
	"context"
	"maps"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/k8s-manifest-kit/engine/pkg/types"
)

// Set returns a transformer that adds or updates labels on objects.
func Set(labelsToApply map[string]string) types.Transformer {
	return func(_ context.Context, obj unstructured.Unstructured) (unstructured.Unstructured, error) {
		values := obj.GetLabels()
		if values == nil {
			values = make(map[string]string)
		}

		maps.Copy(values, labelsToApply)

		obj.SetLabels(values)

		return obj, nil
	}
}

// Remove returns a transformer that removes specific labels from objects.
func Remove(keys ...string) types.Transformer {
	return func(_ context.Context, obj unstructured.Unstructured) (unstructured.Unstructured, error) {
		values := obj.GetLabels()
		if values == nil {
			return obj, nil
		}

		for _, key := range keys {
			delete(values, key)
		}

		obj.SetLabels(values)

		return obj, nil
	}
}

// RemoveIf returns a transformer that removes labels matching a predicate.
func RemoveIf(predicate func(key string, value string) bool) types.Transformer {
	return func(_ context.Context, obj unstructured.Unstructured) (unstructured.Unstructured, error) {
		values := obj.GetLabels()
		if values == nil {
			return obj, nil
		}

		for k, v := range values {
			if predicate(k, v) {
				delete(values, k)
			}
		}

		obj.SetLabels(values)

		return obj, nil
	}
}
