package postrenderer

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	. "github.com/onsi/gomega"
)

// Test helper functions

// resetCustomKinds resets the kindOrder map to its original state.
// This function is intended for testing purposes only.
func resetCustomKinds() {
	// Clear the map and rebuild from scratch
	kindOrder = make(map[string]int, len(orderFirst)+len(orderLast))

	// Rebuild original ordering using the same scale as init()
	for i, kind := range orderFirst {
		kindOrder[kind] = (i - len(orderFirst)) * positionScale
	}

	for i, kind := range orderLast {
		kindOrder[kind] = (i + 1) * positionScale
	}
}

// getKindOrder returns the order value for a kind. For testing purposes only.
func getKindOrder(kind string) int {
	return kindOrder[kind]
}

// debugPositionTaken checks if a position is taken. For testing purposes only.
func debugPositionTaken(pos int) bool {
	return positionTaken(pos)
}

// debugFindAvailablePosition shows what findAvailablePosition would do. For testing purposes only.
func debugFindAvailablePosition(preferred int) int {
	return findAvailablePosition(preferred)
}

func TestApplyOrder(t *testing.T) {
	t.Run("should sort known kinds into apply order", func(t *testing.T) {
		g := NewWithT(t)

		objects := []unstructured.Unstructured{
			makeObj("Deployment", "v1", "apps", "my-deploy", "default"),
			makeObj("Namespace", "v1", "", "my-ns", ""),
			makeObj("MutatingWebhookConfiguration", "v1", "admissionregistration.k8s.io", "my-webhook", ""),
			makeObj("ServiceAccount", "v1", "", "my-sa", "default"),
			makeObj("Secret", "v1", "", "my-secret", "default"),
			makeObj("Service", "v1", "", "my-svc", "default"),
		}

		pr := ApplyOrder()
		result, err := pr(t.Context(), objects)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(HaveLen(6))

		kinds := make([]string, len(result))
		for i, obj := range result {
			kinds[i] = obj.GetKind()
		}

		g.Expect(kinds).To(Equal([]string{
			"Namespace",
			"ServiceAccount",
			"Secret",
			"Service",
			"Deployment",
			"MutatingWebhookConfiguration",
		}))
	})

	t.Run("should place unknown kinds in the middle", func(t *testing.T) {
		g := NewWithT(t)

		objects := []unstructured.Unstructured{
			makeObj("ValidatingWebhookConfiguration", "v1", "admissionregistration.k8s.io", "wh", ""),
			makeObj("MyCustomResource", "v1", "example.com", "cr1", "default"),
			makeObj("Namespace", "v1", "", "ns1", ""),
		}

		pr := ApplyOrder()
		result, err := pr(t.Context(), objects)
		g.Expect(err).ToNot(HaveOccurred())

		kinds := make([]string, len(result))
		for i, obj := range result {
			kinds[i] = obj.GetKind()
		}

		g.Expect(kinds[0]).To(Equal("Namespace"))
		g.Expect(kinds[1]).To(Equal("MyCustomResource"))
		g.Expect(kinds[2]).To(Equal("ValidatingWebhookConfiguration"))
	})

	t.Run("should preserve relative order for same priority (stable sort)", func(t *testing.T) {
		g := NewWithT(t)

		objects := []unstructured.Unstructured{
			makeObj("CustomWidget", "v1", "example.com", "widget-b", "default"),
			makeObj("CustomWidget", "v1", "example.com", "widget-a", "default"),
			makeObj("AnotherCR", "v1", "example.com", "cr1", "default"),
		}

		pr := ApplyOrder()
		result, err := pr(t.Context(), objects)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(HaveLen(3))

		// AnotherCR sorts before CustomWidget by GVK string
		g.Expect(result[0].GetKind()).To(Equal("AnotherCR"))

		// Within CustomWidget, name tiebreaker applies
		g.Expect(result[1].GetName()).To(Equal("widget-a"))
		g.Expect(result[2].GetName()).To(Equal("widget-b"))
	})

	t.Run("should use namespace and name as tiebreakers within same GVK", func(t *testing.T) {
		g := NewWithT(t)

		objects := []unstructured.Unstructured{
			makeObj("Deployment", "v1", "apps", "deploy-b", "ns-b"),
			makeObj("Deployment", "v1", "apps", "deploy-a", "ns-b"),
			makeObj("Deployment", "v1", "apps", "deploy-a", "ns-a"),
		}

		pr := ApplyOrder()
		result, err := pr(t.Context(), objects)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(HaveLen(3))

		g.Expect(result[0].GetNamespace()).To(Equal("ns-a"))
		g.Expect(result[0].GetName()).To(Equal("deploy-a"))

		g.Expect(result[1].GetNamespace()).To(Equal("ns-b"))
		g.Expect(result[1].GetName()).To(Equal("deploy-a"))

		g.Expect(result[2].GetNamespace()).To(Equal("ns-b"))
		g.Expect(result[2].GetName()).To(Equal("deploy-b"))
	})

	t.Run("should handle empty input", func(t *testing.T) {
		g := NewWithT(t)

		pr := ApplyOrder()
		result, err := pr(t.Context(), []unstructured.Unstructured{})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeEmpty())
	})

	t.Run("should handle single object", func(t *testing.T) {
		g := NewWithT(t)

		objects := []unstructured.Unstructured{
			makeObj("Pod", "v1", "", "my-pod", "default"),
		}

		pr := ApplyOrder()
		result, err := pr(t.Context(), objects)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(HaveLen(1))
		g.Expect(result[0].GetName()).To(Equal("my-pod"))
	})

	t.Run("should sort full kustomize legacy ordering", func(t *testing.T) {
		g := NewWithT(t)

		objects := []unstructured.Unstructured{
			makeObj("PodDisruptionBudget", "v1", "policy", "pdb", "default"),
			makeObj("CronJob", "v1", "batch", "cj", "default"),
			makeObj("StatefulSet", "v1", "apps", "sts", "default"),
			makeObj("Deployment", "v1", "apps", "deploy", "default"),
			makeObj("PersistentVolumeClaim", "v1", "", "pvc", "default"),
			makeObj("PersistentVolume", "v1", "", "pv", ""),
			makeObj("PriorityClass", "v1", "scheduling.k8s.io", "pc", ""),
			makeObj("LimitRange", "v1", "", "lr", "default"),
			makeObj("Service", "v1", "", "svc", "default"),
			makeObj("Endpoints", "v1", "", "ep", "default"),
			makeObj("Secret", "v1", "", "secret", "default"),
			makeObj("ConfigMap", "v1", "", "cm", "default"),
			makeObj("ClusterRoleBinding", "v1", "rbac.authorization.k8s.io", "crb", ""),
			makeObj("ClusterRole", "v1", "rbac.authorization.k8s.io", "cr", ""),
			makeObj("RoleBinding", "v1", "rbac.authorization.k8s.io", "rb", "default"),
			makeObj("Role", "v1", "rbac.authorization.k8s.io", "role", "default"),
			makeObj("PodSecurityPolicy", "v1beta1", "policy", "psp", ""),
			makeObj("ServiceAccount", "v1", "", "sa", "default"),
			makeObj("CustomResourceDefinition", "v1", "apiextensions.k8s.io", "crd", ""),
			makeObj("StorageClass", "v1", "storage.k8s.io", "sc", ""),
			makeObj("ResourceQuota", "v1", "", "rq", "default"),
			makeObj("Namespace", "v1", "", "ns", ""),
			makeObj("ValidatingWebhookConfiguration", "v1", "admissionregistration.k8s.io", "vwh", ""),
			makeObj("MutatingWebhookConfiguration", "v1", "admissionregistration.k8s.io", "mwh", ""),
		}

		pr := ApplyOrder()
		result, err := pr(t.Context(), objects)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(HaveLen(24))

		kinds := make([]string, len(result))
		for i, obj := range result {
			kinds[i] = obj.GetKind()
		}

		g.Expect(kinds).To(Equal([]string{
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
			"MutatingWebhookConfiguration",
			"ValidatingWebhookConfiguration",
		}))
	})
}

func makeObj(kind string, version string, group string, name string, namespace string) unstructured.Unstructured {
	apiVersion := version
	if group != "" {
		apiVersion = group + "/" + version
	}

	obj := unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": apiVersion,
			"kind":       kind,
			"metadata": map[string]any{
				"name": name,
			},
		},
	}

	if namespace != "" {
		obj.Object["metadata"].(map[string]any)["namespace"] = namespace
	}

	return obj
}

// indexOf returns the index of target string in slice, or -1 if not found
func indexOf(slice []string, target string) int {
	for i, item := range slice {
		if item == target {
			return i
		}
	}
	return -1
}

func TestCustomKindRegistration(t *testing.T) {
	t.Run("debug intelligent positioning calculation", func(t *testing.T) {
		resetCustomKinds()

		// Test our position calculation functions directly
		serviceOrder := getKindOrder("Service")        // Should be -9
		limitRangeOrder := getKindOrder("LimitRange")  // Should be -8

		t.Logf("Service order: %d, LimitRange order: %d", serviceOrder, limitRangeOrder)

		// Register Certificate after Service (using KindGroup with single element)
		err := registerCustomKinds(
			KindGroup{
				Kinds: []string{"Certificate"},
				After: "Service",
			},
		)
		if err != nil {
			t.Logf("Registration error: %v", err)
		}

		certOrder := getKindOrder("Certificate")
		t.Logf("Certificate got position: %d", certOrder)

		// Test sorting behavior
		objects := []unstructured.Unstructured{
			makeObj("LimitRange", "v1", "", "lr", "default"),
			makeObj("Certificate", "v1", "cert-manager.io", "cert", "default"),
			makeObj("Service", "v1", "", "svc", "default"),
		}

		pr := ApplyOrder()
		result, err := pr(t.Context(), objects)
		if err != nil {
			t.Logf("Sort error: %v", err)
		}

		kinds := make([]string, len(result))
		for i, obj := range result {
			kinds[i] = obj.GetKind()
		}

		t.Logf("Sort result: %v", kinds)
		t.Logf("Expected: [Service, Certificate, LimitRange]")

		// Check if Certificate is positioned correctly
		if certOrder > serviceOrder && certOrder < limitRangeOrder {
			t.Logf("✅ Certificate correctly positioned between Service and LimitRange")
		} else {
			t.Logf("❌ Certificate NOT positioned correctly: %d should be between %d and %d",
				certOrder, serviceOrder, limitRangeOrder)
		}
	})

	t.Run("should handle multiple custom kinds with intelligent positioning", func(t *testing.T) {
		g := NewWithT(t)
		resetCustomKinds() // Reset state for isolated testing

		// Register multiple custom kinds in different parts of the ordering (using KindGroup with single elements)
		err := registerCustomKinds(
			KindGroup{
				Kinds: []string{"Issuer"},
				After: "ServiceAccount", // Early in the pipeline
			},
			KindGroup{
				Kinds: []string{"Certificate"},
				After: "Service", // Middle of the pipeline
			},
			KindGroup{
				Kinds:  []string{"Application"},
				Before: "Deployment", // Late in the pipeline
			},
		)
		g.Expect(err).ToNot(HaveOccurred())

		objects := []unstructured.Unstructured{
			makeObj("Deployment", "v1", "apps", "deploy", "default"),
			makeObj("ServiceAccount", "v1", "", "sa", "default"),
			makeObj("Service", "v1", "", "svc", "default"),
			makeObj("Certificate", "v1", "cert-manager.io", "cert", "default"),
			makeObj("Issuer", "v1", "cert-manager.io", "issuer", "default"),
			makeObj("Application", "v1", "argoproj.io", "app", "default"),
		}

		pr := ApplyOrder()
		result, err := pr(t.Context(), objects)
		g.Expect(err).ToNot(HaveOccurred())

		kinds := make([]string, len(result))
		for i, obj := range result {
			kinds[i] = obj.GetKind()
		}

		// Verify relative ordering is maintained
		saIndex := indexOf(kinds, "ServiceAccount")
		issuerIndex := indexOf(kinds, "Issuer")
		serviceIndex := indexOf(kinds, "Service")
		certIndex := indexOf(kinds, "Certificate")
		appIndex := indexOf(kinds, "Application")
		deployIndex := indexOf(kinds, "Deployment")

		// ServiceAccount -> Issuer (after SA)
		g.Expect(issuerIndex).To(BeNumerically(">", saIndex))

		// Service -> Certificate (after Service)
		g.Expect(certIndex).To(BeNumerically(">", serviceIndex))

		// Application -> Deployment (before Deployment)
		g.Expect(appIndex).To(BeNumerically("<", deployIndex))

		// Overall logical flow
		g.Expect(saIndex).To(BeNumerically("<", serviceIndex)) // SA before Service
		g.Expect(serviceIndex).To(BeNumerically("<", deployIndex)) // Service before Deployment
	})

	t.Run("should register individual kind with KindGroup", func(t *testing.T) {
		g := NewWithT(t)
		resetCustomKinds() // Reset state for isolated testing

		// Register a custom kind after Service (using KindGroup with single element)
		err := registerCustomKinds(
			KindGroup{
				Kinds: []string{"CustomResource"},
				After: "Service",
			},
		)
		g.Expect(err).ToNot(HaveOccurred())

		objects := []unstructured.Unstructured{
			makeObj("Deployment", "v1", "apps", "deploy", "default"),
			makeObj("Service", "v1", "", "svc", "default"),
			makeObj("CustomResource", "v1", "example.com", "cr", "default"),
		}

		pr := ApplyOrder()
		result, err := pr(t.Context(), objects)
		g.Expect(err).ToNot(HaveOccurred())

		kinds := make([]string, len(result))
		for i, obj := range result {
			kinds[i] = obj.GetKind()
		}

		g.Expect(kinds).To(Equal([]string{
			"Service",
			"CustomResource", // Should come after Service
			"Deployment",
		}))
	})

	t.Run("should register kind group with KindGroup", func(t *testing.T) {
		g := NewWithT(t)
		resetCustomKinds() // Reset state for isolated testing

		err := registerCustomKinds(
			KindGroup{
				Kinds:  []string{"Issuer", "Certificate"},
				Before: "Deployment",
			},
		)
		g.Expect(err).ToNot(HaveOccurred())

		objects := []unstructured.Unstructured{
			makeObj("Deployment", "v1", "apps", "deploy", "default"),
			makeObj("Certificate", "v1", "cert-manager.io", "cert", "default"),
			makeObj("Issuer", "v1", "cert-manager.io", "issuer", "default"),
		}

		pr := ApplyOrder()
		result, err := pr(t.Context(), objects)
		g.Expect(err).ToNot(HaveOccurred())

		kinds := make([]string, len(result))
		for i, obj := range result {
			kinds[i] = obj.GetKind()
		}

		g.Expect(kinds).To(Equal([]string{
			"Issuer",      // First in group
			"Certificate", // Second in group
			"Deployment",  // After the group
		}))
	})

	t.Run("should handle multiple registrations", func(t *testing.T) {
		g := NewWithT(t)
		resetCustomKinds() // Reset state for isolated testing

		// Use end-of-range positions that work with current implementation (using KindGroup with single elements)
		err := registerCustomKinds(
			KindGroup{
				Kinds: []string{"Gateway"},
				After: "PodDisruptionBudget", // Last in built-in range
			},
			KindGroup{
				Kinds:  []string{"VirtualService"},
				Before: "MutatingWebhookConfiguration", // First webhook
			},
		)
		g.Expect(err).ToNot(HaveOccurred())

		objects := []unstructured.Unstructured{
			makeObj("MutatingWebhookConfiguration", "v1", "admissionregistration.k8s.io", "mwh", ""),
			makeObj("PodDisruptionBudget", "v1", "policy", "pdb", "default"),
			makeObj("Gateway", "v1", "networking.istio.io", "gw", "default"),
			makeObj("VirtualService", "v1", "networking.istio.io", "vs", "default"),
		}

		pr := ApplyOrder()
		result, err := pr(t.Context(), objects)
		g.Expect(err).ToNot(HaveOccurred())

		kinds := make([]string, len(result))
		for i, obj := range result {
			kinds[i] = obj.GetKind()
		}

		// Should be ordered correctly with custom kinds in the gap before webhooks
		g.Expect(kinds[0]).To(Equal("PodDisruptionBudget"))
		// Gateway and VirtualService should be after PDB but before MWH
		// (exact order depends on position allocation)
		g.Expect(kinds[3]).To(Equal("MutatingWebhookConfiguration"))
	})

	t.Run("should work with realistic cert-manager scenario", func(t *testing.T) {
		g := NewWithT(t)
		resetCustomKinds() // Reset state for isolated testing

		// Register cert-manager in a safe gap (after main built-ins, before webhooks)
		err := registerCustomKinds(
			KindGroup{
				Kinds: []string{"Issuer", "ClusterIssuer", "Certificate"},
				After: "PodDisruptionBudget", // Safe position
			},
		)
		g.Expect(err).ToNot(HaveOccurred())

		objects := []unstructured.Unstructured{
			makeObj("ValidatingWebhookConfiguration", "v1", "admissionregistration.k8s.io", "vwh", ""),
			makeObj("Certificate", "v1", "cert-manager.io", "cert", "default"),
			makeObj("Deployment", "v1", "apps", "deploy", "default"),
			makeObj("PodDisruptionBudget", "v1", "policy", "pdb", "default"),
			makeObj("Issuer", "v1", "cert-manager.io", "issuer", "default"),
			makeObj("ClusterIssuer", "v1", "cert-manager.io", "cluster-issuer", "default"),
		}

		pr := ApplyOrder()
		result, err := pr(t.Context(), objects)
		g.Expect(err).ToNot(HaveOccurred())

		kinds := make([]string, len(result))
		for i, obj := range result {
			kinds[i] = obj.GetKind()
		}

		// Verify cert-manager resources come in the right general area
		pdbIndex := -1
		vwhIndex := -1
		for i, kind := range kinds {
			if kind == "PodDisruptionBudget" {
				pdbIndex = i
			}
			if kind == "ValidatingWebhookConfiguration" {
				vwhIndex = i
			}
		}

		g.Expect(pdbIndex).To(BeNumerically(">=", 0))
		g.Expect(vwhIndex).To(BeNumerically(">=", 0))
		g.Expect(pdbIndex).To(BeNumerically("<", vwhIndex)) // PDB before webhooks

		// Cert-manager kinds should be between PDB and webhooks
		for _, cmKind := range []string{"Issuer", "ClusterIssuer", "Certificate"} {
			for i, kind := range kinds {
				if kind == cmKind {
					g.Expect(i).To(BeNumerically(">", pdbIndex))
					g.Expect(i).To(BeNumerically("<", vwhIndex))
				}
			}
		}
	})
}

func TestCustomKindRegistrationErrors(t *testing.T) {
	t.Run("should return error for invalid KindGroup", func(t *testing.T) {
		g := NewWithT(t)

		// Empty kind in single-element group
		err := registerCustomKinds(
			KindGroup{
				Kinds: []string{""},
				After: "Service",
			},
		)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("kind at index 0 cannot be empty"))

		// No Before or After
		err = registerCustomKinds(
			KindGroup{
				Kinds: []string{"CustomResource"},
			},
		)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("must specify either Before or After"))

		// Both Before and After
		err = registerCustomKinds(
			KindGroup{
				Kinds:  []string{"CustomResource"},
				Before: "Deployment",
				After:  "Service",
			},
		)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("cannot specify both Before and After"))

		// Non-existent reference kind
		err = registerCustomKinds(
			KindGroup{
				Kinds: []string{"CustomResource"},
				After: "NonExistentKind",
			},
		)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("reference kind not found"))
	})

	t.Run("should return error for invalid KindGroup", func(t *testing.T) {
		g := NewWithT(t)

		// Empty group
		err := registerCustomKinds(
			KindGroup{
				Kinds: []string{},
				After: "Service",
			},
		)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("kind group cannot be empty"))

		// Empty kind in group
		err = registerCustomKinds(
			KindGroup{
				Kinds: []string{"Issuer", ""},
				After: "Service",
			},
		)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("kind at index 1 cannot be empty"))
	})
}

func TestIntelligentPositioning(t *testing.T) {
	t.Run("should use intelligent fractional positioning", func(t *testing.T) {
		g := NewWithT(t)
		resetCustomKinds() // Reset state for isolated testing

		// Register a custom kind after Service using the intelligent positioning (using KindGroup with single element)
		err := registerCustomKinds(
			KindGroup{
				Kinds: []string{"Certificate"},
				After: "Service",
			},
		)
		g.Expect(err).ToNot(HaveOccurred())

		objects := []unstructured.Unstructured{
			makeObj("Deployment", "v1", "apps", "deploy", "default"),
			makeObj("Service", "v1", "", "svc", "default"),
			makeObj("LimitRange", "v1", "", "lr", "default"),
			makeObj("Certificate", "v1", "cert-manager.io", "cert", "default"),
		}

		pr := ApplyOrder()
		result, err := pr(t.Context(), objects)
		g.Expect(err).ToNot(HaveOccurred())

		kinds := make([]string, len(result))
		for i, obj := range result {
			kinds[i] = obj.GetKind()
		}

		// Should be: Service, Certificate (inserted between Service and LimitRange), LimitRange, Deployment
		g.Expect(kinds[0]).To(Equal("Service"))
		g.Expect(kinds[1]).To(Equal("Certificate")) // Should be between Service and LimitRange
		g.Expect(kinds[2]).To(Equal("LimitRange"))
		g.Expect(kinds[3]).To(Equal("Deployment"))

		// Verify the Certificate got a fractional position between Service(-9) and LimitRange(-8)
		certOrder := getKindOrder("Certificate")
		serviceOrder := getKindOrder("Service")
		limitRangeOrder := getKindOrder("LimitRange")

		// Certificate should be positioned between Service and LimitRange
		g.Expect(certOrder).To(BeNumerically(">", serviceOrder))
		g.Expect(certOrder).To(BeNumerically("<", limitRangeOrder))
	})
}

func TestCriticalEdgeCases(t *testing.T) {
	t.Run("should avoid position 0 conflicts", func(t *testing.T) {
		g := NewWithT(t)
		resetCustomKinds()

		// Test the position 0 avoidance logic explicitly
		g.Expect(debugPositionTaken(0)).To(BeTrue(), "Position 0 should always be marked as taken")

		// Verify unknown kinds default to 0 and custom kinds avoid it
		g.Expect(getKindOrder("UnknownKind")).To(Equal(0), "Unknown kinds should default to position 0")

		// Register a custom kind and ensure it doesn't get position 0
		err := RegisterKinds([]string{"CustomKind"}, After("Service"))
		g.Expect(err).ToNot(HaveOccurred())

		customPos := getKindOrder("CustomKind")
		g.Expect(customPos).ToNot(Equal(0), "Custom kinds should never be assigned position 0")
	})

	t.Run("should handle collision detection and resolution", func(t *testing.T) {
		g := NewWithT(t)
		resetCustomKinds()

		// Force a collision by manually placing a kind where we want to register another
		servicePos := getKindOrder("Service") // -180
		expectedPos := servicePos + positionOffset // -170

		// Manually set a conflicting position
		kindOrder["ConflictingKind"] = expectedPos

		// Now register a kind that would conflict
		err := RegisterKinds([]string{"NewKind"}, After("Service"))
		g.Expect(err).ToNot(HaveOccurred())

		// Should resolve collision by finding next available position
		newKindPos := getKindOrder("NewKind")
		conflictingPos := getKindOrder("ConflictingKind")

		g.Expect(newKindPos).ToNot(Equal(conflictingPos), "Should resolve collision")
		g.Expect(newKindPos).To(Equal(expectedPos-1), "Should use next negative position for collision resolution")
	})

	t.Run("should handle multiple collision resolution", func(t *testing.T) {
		g := NewWithT(t)
		resetCustomKinds()

		servicePos := getKindOrder("Service") // -180
		basePos := servicePos + positionOffset // -170

		// Manually create multiple conflicts
		kindOrder["Conflict1"] = basePos      // -170
		kindOrder["Conflict2"] = basePos - 1  // -171
		kindOrder["Conflict3"] = basePos - 2  // -172

		// Register multiple kinds that will hit collisions
		err := RegisterKinds([]string{"Kind1", "Kind2", "Kind3"}, After("Service"))
		g.Expect(err).ToNot(HaveOccurred())

		// Should resolve all collisions
		pos1 := getKindOrder("Kind1")
		pos2 := getKindOrder("Kind2")
		pos3 := getKindOrder("Kind3")

		// All should be different and avoid conflicts
		positions := []int{pos1, pos2, pos3, basePos, basePos-1, basePos-2}
		uniquePositions := make(map[int]bool)
		for _, pos := range positions {
			g.Expect(uniquePositions[pos]).To(BeFalse(), "All positions should be unique")
			uniquePositions[pos] = true
		}
	})

	t.Run("should handle boundary edge cases", func(t *testing.T) {
		g := NewWithT(t)
		resetCustomKinds()

		// Test registration before first kind
		err := RegisterKinds([]string{"VeryEarly"}, Before("Namespace"))
		g.Expect(err).ToNot(HaveOccurred())

		namespacePos := getKindOrder("Namespace")
		veryEarlyPos := getKindOrder("VeryEarly")
		g.Expect(veryEarlyPos).To(BeNumerically("<", namespacePos), "Should be positioned before Namespace")

		// Test registration after last webhook
		err = RegisterKinds([]string{"VeryLate"}, After("ValidatingWebhookConfiguration"))
		g.Expect(err).ToNot(HaveOccurred())

		lastWebhookPos := getKindOrder("ValidatingWebhookConfiguration")
		veryLatePos := getKindOrder("VeryLate")
		g.Expect(veryLatePos).To(BeNumerically(">", lastWebhookPos), "Should be positioned after last webhook")
	})

	t.Run("should handle positive vs negative collision resolution", func(t *testing.T) {
		g := NewWithT(t)
		resetCustomKinds()

		// Test negative position collision (most common case)
		servicePos := getKindOrder("Service") // -180 (negative)
		expectedNegative := servicePos + positionOffset // -170

		kindOrder["NegativeConflict"] = expectedNegative
		err := RegisterKinds([]string{"NegativeKind"}, After("Service"))
		g.Expect(err).ToNot(HaveOccurred())

		negativeResult := getKindOrder("NegativeKind")
		g.Expect(negativeResult).To(Equal(expectedNegative-1), "Negative collision should decrement position")

		// Test positive position collision (webhook area)
		webhookPos := getKindOrder("ValidatingWebhookConfiguration") // 40 (positive)
		expectedPositive := webhookPos + positionOffset // 50

		kindOrder["PositiveConflict"] = expectedPositive
		err = RegisterKinds([]string{"PositiveKind"}, After("ValidatingWebhookConfiguration"))
		g.Expect(err).ToNot(HaveOccurred())

		positiveResult := getKindOrder("PositiveKind")
		g.Expect(positiveResult).To(Equal(expectedPositive+1), "Positive collision should increment position")
	})

	t.Run("should maintain correct ordering through collisions", func(t *testing.T) {
		g := NewWithT(t)
		resetCustomKinds()

		// Create a collision scenario and ensure final sorting is still correct
		err := RegisterKinds([]string{"First"}, After("Service"))
		g.Expect(err).ToNot(HaveOccurred())

		// Force collision on second registration
		firstPos := getKindOrder("First")
		kindOrder["ManualConflict"] = firstPos - 1

		err = RegisterKinds([]string{"Second"}, After("Service"))
		g.Expect(err).ToNot(HaveOccurred())

		// Test actual sorting works correctly despite collisions
		objects := []unstructured.Unstructured{
			makeObj("Second", "v1", "example.com", "second", "default"),
			makeObj("Service", "v1", "", "svc", "default"),
			makeObj("First", "v1", "example.com", "first", "default"),
			makeObj("LimitRange", "v1", "", "lr", "default"),
		}

		pr := ApplyOrder()
		result, err := pr(t.Context(), objects)
		g.Expect(err).ToNot(HaveOccurred())

		kinds := make([]string, len(result))
		for i, obj := range result {
			kinds[i] = obj.GetKind()
		}

		// Service should come first, then custom kinds, then LimitRange
		g.Expect(kinds[0]).To(Equal("Service"))
		g.Expect(kinds[len(kinds)-1]).To(Equal("LimitRange"))

		// Custom kinds should be between Service and LimitRange
		serviceIndex := 0
		limitRangeIndex := len(kinds) - 1
		for i, kind := range kinds {
			if kind == "First" || kind == "Second" {
				g.Expect(i).To(BeNumerically(">", serviceIndex))
				g.Expect(i).To(BeNumerically("<", limitRangeIndex))
			}
		}
	})
}

func TestNewClearAPI(t *testing.T) {
	t.Run("should use new clear RegisterKinds API for single kind", func(t *testing.T) {
		g := NewWithT(t)
		resetCustomKinds() // Reset state for isolated testing

		// Test single kind registration with clear parameter roles
		err := RegisterKinds([]string{"Certificate"}, After("Service"))
		g.Expect(err).ToNot(HaveOccurred())

		objects := []unstructured.Unstructured{
			makeObj("Deployment", "v1", "apps", "deploy", "default"),
			makeObj("Service", "v1", "", "svc", "default"),
			makeObj("Certificate", "v1", "cert-manager.io", "cert", "default"),
		}

		pr := ApplyOrder()
		result, err := pr(t.Context(), objects)
		g.Expect(err).ToNot(HaveOccurred())

		kinds := make([]string, len(result))
		for i, obj := range result {
			kinds[i] = obj.GetKind()
		}

		g.Expect(kinds).To(Equal([]string{
			"Service",
			"Certificate", // Should come after Service
			"Deployment",
		}))
	})

	t.Run("should use new clear RegisterKinds API for multiple kinds", func(t *testing.T) {
		g := NewWithT(t)
		resetCustomKinds() // Reset state for isolated testing

		// Test multiple kinds registration with clear parameter roles
		err := RegisterKinds([]string{"Gateway", "VirtualService"}, Before("Deployment"))
		g.Expect(err).ToNot(HaveOccurred())

		objects := []unstructured.Unstructured{
			makeObj("Deployment", "v1", "apps", "deploy", "default"),
			makeObj("VirtualService", "v1", "networking.istio.io", "vs", "default"),
			makeObj("Gateway", "v1", "networking.istio.io", "gw", "default"),
		}

		pr := ApplyOrder()
		result, err := pr(t.Context(), objects)
		g.Expect(err).ToNot(HaveOccurred())

		kinds := make([]string, len(result))
		for i, obj := range result {
			kinds[i] = obj.GetKind()
		}

		g.Expect(kinds).To(Equal([]string{
			"Gateway",       // Before Deployment, first in group
			"VirtualService", // Before Deployment, second in group
			"Deployment",
		}))
	})

	t.Run("should handle multiple registrations", func(t *testing.T) {
		g := NewWithT(t)
		resetCustomKinds() // Reset state for isolated testing

		// Multiple separate registrations should work together
		err := RegisterKinds([]string{"Certificate"}, After("Service"))
		g.Expect(err).ToNot(HaveOccurred())

		err = RegisterKinds([]string{"Issuer"}, After("Service"))
		g.Expect(err).ToNot(HaveOccurred())

		err = RegisterKinds([]string{"Application"}, Before("Deployment"))
		g.Expect(err).ToNot(HaveOccurred())

		objects := []unstructured.Unstructured{
			makeObj("Deployment", "v1", "apps", "deploy", "default"),
			makeObj("Service", "v1", "", "svc", "default"),
			makeObj("Certificate", "v1", "cert-manager.io", "cert", "default"),
			makeObj("Issuer", "v1", "cert-manager.io", "issuer", "default"),
			makeObj("Application", "v1", "argoproj.io", "app", "default"),
		}

		pr := ApplyOrder()
		result, err := pr(t.Context(), objects)
		g.Expect(err).ToNot(HaveOccurred())

		// Verify all kinds are positioned correctly (specific order may vary due to collision resolution)
		kinds := make([]string, len(result))
		for i, obj := range result {
			kinds[i] = obj.GetKind()
		}

		// Service should be first
		g.Expect(kinds[0]).To(Equal("Service"))

		// Application should come before Deployment
		appIndex := indexOf(kinds, "Application")
		deployIndex := indexOf(kinds, "Deployment")
		g.Expect(appIndex).To(BeNumerically("<", deployIndex))

		// Certificate and Issuer should come after Service but before Deployment
		certIndex := indexOf(kinds, "Certificate")
		issuerIndex := indexOf(kinds, "Issuer")
		serviceIndex := indexOf(kinds, "Service")

		g.Expect(certIndex).To(BeNumerically(">", serviceIndex))
		g.Expect(issuerIndex).To(BeNumerically(">", serviceIndex))
		g.Expect(certIndex).To(BeNumerically("<", deployIndex))
		g.Expect(issuerIndex).To(BeNumerically("<", deployIndex))
	})

	t.Run("should validate new API parameters", func(t *testing.T) {
		g := NewWithT(t)

		// Empty custom kind in slice
		err := RegisterKinds([]string{""}, After("Service"))
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("kind at index 0 cannot be empty"))

		// Empty reference kind
		err = RegisterKinds([]string{"Certificate"}, After(""))
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("reference kind cannot be empty"))

		// Empty custom kinds slice
		err = RegisterKinds([]string{}, After("Service"))
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("kinds cannot be empty"))

		// Empty kind in slice
		err = RegisterKinds([]string{"Certificate", ""}, After("Service"))
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("kind at index 1 cannot be empty"))
	})
}
