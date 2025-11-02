package name

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/k8s-manifest-kit/engine/pkg/types"
)

// Exact returns a filter that keeps objects with exact name matches.
func Exact(names ...string) types.Filter {
	nameSet := make(map[string]bool, len(names))
	for _, name := range names {
		nameSet[name] = true
	}

	return func(_ context.Context, obj unstructured.Unstructured) (bool, error) {
		return nameSet[obj.GetName()], nil
	}
}

// Prefix returns a filter that keeps objects whose name starts with the given prefix.
func Prefix(prefix string) types.Filter {
	return func(_ context.Context, obj unstructured.Unstructured) (bool, error) {
		return strings.HasPrefix(obj.GetName(), prefix), nil
	}
}

// Suffix returns a filter that keeps objects whose name ends with the given suffix.
func Suffix(suffix string) types.Filter {
	return func(_ context.Context, obj unstructured.Unstructured) (bool, error) {
		return strings.HasSuffix(obj.GetName(), suffix), nil
	}
}

// Regex returns a filter that keeps objects whose name matches the given regex pattern.
func Regex(pattern string) (types.Filter, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	f := func(_ context.Context, obj unstructured.Unstructured) (bool, error) {
		return re.MatchString(obj.GetName()), nil
	}

	return f, nil
}
