// Package postrenderer provides PostRenderer implementations.
package postrenderer

import (
	"context"
	"fmt"
	"sort"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/k8s-manifest-kit/engine/pkg/types"
)

// SortFunc is a function that sorts Kubernetes resources.
type SortFunc func([]unstructured.Unstructured) []unstructured.Unstructured

//nolint:gochecknoglobals
var (
	// orderSpacing is the multiplier used to create gaps between built-in resources for custom insertion.
	orderSpacing = 10

	orderFirst = []string{
		"Namespace",
		"ResourceQuota",
		"StorageClass",
		"CustomResourceDefinition",
		"ServiceAccount",
		"PodSecurityPolicy",
		"Role",
		"ClusterRole",
		"RoleBinding",
		"ClusterRoleBinding",
		"ConfigMap",
		"Secret",
		"Endpoints",
		"Service",
		"LimitRange",
		"PriorityClass",
		"PersistentVolume",
		"PersistentVolumeClaim",
		"Deployment",
		"StatefulSet",
		"CronJob",
		"PodDisruptionBudget",
	}

	orderLast = []string{
		"MutatingWebhookConfiguration",
		"ValidatingWebhookConfiguration",
	}

	kindOrder map[string]int
)

//nolint:gochecknoinits
func init() {
	kindOrder = make(map[string]int, len(orderFirst)+len(orderLast))

	// Use orderSpacing to leave room for custom resource insertion
	for i, kind := range orderFirst {
		kindOrder[kind] = (i - len(orderFirst)) * orderSpacing
	}

	for i, kind := range orderLast {
		kindOrder[kind] = (i + 1) * orderSpacing
	}
}

// ApplyOrder returns a PostRenderer that sorts resources into dependency
// order for cluster application. If sortFunc is provided, uses custom sorting;
// otherwise uses built-in dependency ordering.
func ApplyOrder(sortFunc SortFunc) types.PostRenderer {
	return func(_ context.Context, objects []unstructured.Unstructured) ([]unstructured.Unstructured, error) {
		if sortFunc != nil {
			return sortFunc(objects), nil
		}

		sort.SliceStable(objects, func(i, j int) bool {
			return compareOrder(objects[i], objects[j])
		})

		return objects, nil
	}
}

func compareOrder(a unstructured.Unstructured, b unstructured.Unstructured) bool {
	orderA := kindOrder[a.GetKind()]
	orderB := kindOrder[b.GetKind()]

	if orderA != orderB {
		return orderA < orderB
	}

	gvkA := gvkString(a)
	gvkB := gvkString(b)

	if gvkA != gvkB {
		return gvkA < gvkB
	}

	nsA := a.GetNamespace()
	nsB := b.GetNamespace()

	if nsA != nsB {
		return nsA < nsB
	}

	return a.GetName() < b.GetName()
}

func gvkString(obj unstructured.Unstructured) string {
	gvk := obj.GroupVersionKind()

	return fmt.Sprintf("%s/%s/%s", gvk.Group, gvk.Version, gvk.Kind)
}
