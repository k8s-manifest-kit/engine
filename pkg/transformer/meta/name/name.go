package name

import (
	"context"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/k8s-manifest-kit/engine/pkg/types"
)

// SetPrefix returns a transformer that adds a prefix to resource names.
func SetPrefix(prefix string) types.Transformer {
	return func(_ context.Context, obj unstructured.Unstructured) (unstructured.Unstructured, error) {
		obj.SetName(prefix + obj.GetName())

		return obj, nil
	}
}

// SetSuffix returns a transformer that adds a suffix to resource names.
func SetSuffix(suffix string) types.Transformer {
	return func(_ context.Context, obj unstructured.Unstructured) (unstructured.Unstructured, error) {
		obj.SetName(obj.GetName() + suffix)

		return obj, nil
	}
}

// Replace returns a transformer that replaces all occurrences of a substring in resource names.
func Replace(from string, to string) types.Transformer {
	return func(_ context.Context, obj unstructured.Unstructured) (unstructured.Unstructured, error) {
		obj.SetName(strings.ReplaceAll(obj.GetName(), from, to))

		return obj, nil
	}
}
