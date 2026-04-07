package postrenderer

import (
	"context"
	"fmt"
	"sort"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/k8s-manifest-kit/engine/pkg/types"
)

// KindRegistration represents a registration request for custom kinds.
type KindRegistration interface {
	register() error
}

// KindGroup defines where a group of kinds should be positioned together in the apply order.
// This is an internal type used by the registration system. Users should use RegisterKinds()
// with After() or Before() positioning instead of constructing KindGroup directly.
//
// Internal structure:
//   - Kinds: List of custom kind names to insert as a group
//   - Before/After: Reference kind for positioning (exactly one must be specified)
//
// The registration system ensures kinds in the group maintain their relative order
// while being positioned correctly relative to built-in Kubernetes kinds.
type KindGroup struct {
	Kinds  []string // Custom kinds to insert as a group (single or multiple)
	Before string   // Insert before this built-in kind (optional, mutually exclusive with After)
	After  string   // Insert after this built-in kind (optional, mutually exclusive with Before)
}

//nolint:gochecknoglobals
var (
	// positionScale is the multiplier used to space built-in kinds (20x = 19 gaps between kinds)
	positionScale = 20

	// positionOffset is the default offset from reference kind for custom kind placement
	// With 20x scale, offset of 10 places custom kinds in middle of 19-position gap
	positionOffset = 10

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

// register implements KindRegistration for kind group placement.
func (kg KindGroup) register() error {
	if len(kg.Kinds) == 0 {
		return fmt.Errorf("kind group cannot be empty")
	}

	if kg.Before == "" && kg.After == "" {
		return fmt.Errorf("must specify either Before or After for kind group")
	}

	if kg.Before != "" && kg.After != "" {
		return fmt.Errorf("cannot specify both Before and After for kind group")
	}

	var basePosition int
	var referenceKindFound bool

	if kg.Before != "" {
		if order, exists := kindOrder[kg.Before]; exists {
			// Start before the reference (collision detection handled per-kind)
			basePosition = order - positionOffset
			referenceKindFound = true
		}
	} else if kg.After != "" {
		if order, exists := kindOrder[kg.After]; exists {
			// Start after the reference (collision detection handled per-kind)
			basePosition = order + positionOffset
			referenceKindFound = true
		}
	}

	if !referenceKindFound {
		return fmt.Errorf("reference kind not found for kind group")
	}

	// Insert kinds in sequence, with collision detection for each
	for i, kind := range kg.Kinds {
		if kind == "" {
			return fmt.Errorf("kind at index %d cannot be empty", i)
		}

		// Find available position for this kind
		targetPosition := findAvailablePosition(basePosition + i)
		kindOrder[kind] = targetPosition
	}

	return nil
}

// findAvailablePosition finds an available position near the preferred value.
// With the 20x scale, we have 19 positions between built-in kinds, so just use simple linear search.
func findAvailablePosition(preferred int) int {
	// With 20x scale, conflicts are rare. Simple approach: try preferred, then increment.
	position := preferred
	for positionTaken(position) {
		if preferred < 0 {
			position-- // For negative positions, go more negative to maintain order
		} else {
			position++ // For positive positions, go more positive
		}
	}
	return position
}

// positionTaken checks if a specific position is already occupied.
func positionTaken(position int) bool {
	// Position 0 is special - it's the default value for unknown kinds
	// We should avoid placing custom kinds at 0 to prevent conflicts
	if position == 0 {
		return true
	}

	for _, existingPos := range kindOrder {
		if existingPos == position {
			return true
		}
	}

	return false
}

// registerCustomKinds allows internal registration of custom kinds with specific ordering.
func registerCustomKinds(registrations ...KindRegistration) error {
	for _, reg := range registrations {
		if err := reg.register(); err != nil {
			return err
		}
	}

	return nil
}

// Position represents where custom kinds should be placed relative to a reference kind.
// Position instances are created using the After() and Before() helper functions.
// This type is not meant to be constructed directly by users.
//
// Example:
//
//	pos := After("Service")     // Place after Service
//	pos := Before("Deployment") // Place before Deployment
type Position struct {
	reference string // The built-in Kubernetes kind to use as reference
	placement string // "before" or "after" - indicates placement relative to reference
}

// After creates a position that places custom kinds after the reference kind.
// Example: RegisterKinds([]string{"Certificate"}, After("Service"))
func After(referenceKind string) Position {
	return Position{reference: referenceKind, placement: "after"}
}

// Before creates a position that places custom kinds before the reference kind.
// Example: RegisterKinds([]string{"Gateway"}, Before("Deployment"))
func Before(referenceKind string) Position {
	return Position{reference: referenceKind, placement: "before"}
}

// RegisterKinds registers custom kinds at the specified position in the apply order.
// Kinds are inserted as a group in sequence with automatic collision resolution.
//
// Examples:
//
//	RegisterKinds([]string{"Certificate", "Issuer"}, After("Service"))
//	RegisterKinds([]string{"Gateway"}, Before("Deployment"))
func RegisterKinds(customKinds []string, position Position) error {
	if len(customKinds) == 0 {
		return fmt.Errorf("kinds cannot be empty")
	}

	for i, kind := range customKinds {
		if kind == "" {
			return fmt.Errorf("kind at index %d cannot be empty", i)
		}
	}

	if position.reference == "" {
		return fmt.Errorf("reference kind cannot be empty")
	}

	switch position.placement {
	case "after":
		return registerCustomKinds(KindGroup{
			Kinds: customKinds,
			After: position.reference,
		})
	case "before":
		return registerCustomKinds(KindGroup{
			Kinds:  customKinds,
			Before: position.reference,
		})
	default:
		return fmt.Errorf("invalid position placement: %s", position.placement)
	}
}

//nolint:gochecknoinits
func init() {
	kindOrder = make(map[string]int, len(orderFirst)+len(orderLast))

	// Use positionScale to leave room for insertions between built-in kinds
	// orderFirst: -1160, -1140, -1120, ..., -20 (22 kinds × positionScale)
	// orderLast: 20, 40 (2 kinds × positionScale)
	// This leaves (positionScale-1) positions between each built-in kind for custom insertions

	for i, kind := range orderFirst {
		kindOrder[kind] = (i - len(orderFirst)) * positionScale
	}

	for i, kind := range orderLast {
		kindOrder[kind] = (i + 1) * positionScale
	}
}

// ApplyOrder returns a PostRenderer that sorts resources into dependency
// order for cluster application. Cluster-wide foundational resources
// (Namespace, CRD, ServiceAccount, etc.) come first; resources with many
// dependencies (webhooks) come last. Resources not in either list are
// placed in the middle, sorted by GVK string for stability.
func ApplyOrder() types.PostRenderer {
	return func(_ context.Context, objects []unstructured.Unstructured) ([]unstructured.Unstructured, error) {
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
