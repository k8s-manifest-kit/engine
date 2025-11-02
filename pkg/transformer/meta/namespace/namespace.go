package namespace

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/k8s-manifest-kit/engine/pkg/types"
)

// Set returns a transformer that sets the namespace on all objects.
func Set(namespace string) types.Transformer {
	return func(_ context.Context, obj unstructured.Unstructured) (unstructured.Unstructured, error) {
		obj.SetNamespace(namespace)

		return obj, nil
	}
}

// EnsureDefault returns a transformer that sets the namespace only if it's empty.
// This is useful for ensuring objects have a namespace without overwriting existing ones.
func EnsureDefault(namespace string) types.Transformer {
	return func(_ context.Context, obj unstructured.Unstructured) (unstructured.Unstructured, error) {
		if obj.GetNamespace() == "" {
			obj.SetNamespace(namespace)
		}

		return obj, nil
	}
}
