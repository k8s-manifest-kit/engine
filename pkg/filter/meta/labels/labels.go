package labels

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/k8s-manifest-kit/engine/pkg/types"
)

// HasLabel returns a filter that keeps objects that have the specified label key.
func HasLabel(key string) types.Filter {
	return func(_ context.Context, obj unstructured.Unstructured) (bool, error) {
		objLabels := obj.GetLabels()
		_, ok := objLabels[key]

		return ok, nil
	}
}

// HasLabels returns a filter that keeps objects that have all specified label keys.
func HasLabels(keys ...string) types.Filter {
	return func(_ context.Context, obj unstructured.Unstructured) (bool, error) {
		objLabels := obj.GetLabels()
		for _, key := range keys {
			if _, ok := objLabels[key]; !ok {
				return false, nil
			}
		}

		return true, nil
	}
}

// MatchLabels returns a filter that keeps objects that have all matching label key-values.
func MatchLabels(matchLabels map[string]string) types.Filter {
	return func(_ context.Context, obj unstructured.Unstructured) (bool, error) {
		objLabels := obj.GetLabels()
		for key, value := range matchLabels {
			if objValue, ok := objLabels[key]; !ok || objValue != value {
				return false, nil
			}
		}

		return true, nil
	}
}

// Selector returns a filter that uses Kubernetes label selector syntax.
// The selector string uses the standard Kubernetes selector format (e.g., "app=nginx,env!=prod").
func Selector(selector string) (types.Filter, error) {
	sel, err := labels.Parse(selector)
	if err != nil {
		return nil, fmt.Errorf("invalid selector: %w", err)
	}

	f := func(_ context.Context, obj unstructured.Unstructured) (bool, error) {
		return sel.Matches(labels.Set(obj.GetLabels())), nil
	}

	return f, nil
}
