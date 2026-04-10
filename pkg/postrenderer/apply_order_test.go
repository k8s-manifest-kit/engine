package postrenderer_test

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/k8s-manifest-kit/engine/pkg/postrenderer"

	. "github.com/onsi/gomega"
)

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

		pr := postrenderer.ApplyOrder(nil) // Use built-in sorting
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

		pr := postrenderer.ApplyOrder(nil) // Use built-in sorting
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

		pr := postrenderer.ApplyOrder(nil) // Use built-in sorting
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

		pr := postrenderer.ApplyOrder(nil) // Use built-in sorting
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

		pr := postrenderer.ApplyOrder(nil) // Use built-in sorting
		result, err := pr(t.Context(), []unstructured.Unstructured{})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeEmpty())
	})

	t.Run("should handle single object", func(t *testing.T) {
		g := NewWithT(t)

		objects := []unstructured.Unstructured{
			makeObj("Pod", "v1", "", "my-pod", "default"),
		}

		pr := postrenderer.ApplyOrder(nil) // Use built-in sorting
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

		pr := postrenderer.ApplyOrder(nil) // Use built-in sorting
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

func TestApplyOrderWithCallback(t *testing.T) {
	t.Run("should use custom sorting when callback provided", func(t *testing.T) {
		g := NewWithT(t)

		objects := []unstructured.Unstructured{
			makeObj("Service", "v1", "", "svc", "default"),
			makeObj("Namespace", "v1", "", "ns", ""),
			makeObj("Deployment", "v1", "apps", "deploy", "default"),
		}

		// Custom sort: reverse alphabetical by kind
		customSort := func(objs []unstructured.Unstructured) []unstructured.Unstructured {
			result := make([]unstructured.Unstructured, len(objs))
			copy(result, objs)

			// Simple reverse alphabetical sort by kind
			for i := range len(result) - 1 {
				for j := i + 1; j < len(result); j++ {
					if result[i].GetKind() < result[j].GetKind() {
						result[i], result[j] = result[j], result[i]
					}
				}
			}

			return result
		}

		pr := postrenderer.ApplyOrder(customSort)
		result, err := pr(t.Context(), objects)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(HaveLen(3))

		kinds := make([]string, len(result))
		for i, obj := range result {
			kinds[i] = obj.GetKind()
		}

		// Should be reverse alphabetical: Service, Namespace, Deployment
		g.Expect(kinds).To(Equal([]string{
			"Service",
			"Namespace",
			"Deployment",
		}))
	})

	t.Run("should handle nil callback as built-in sorting", func(t *testing.T) {
		g := NewWithT(t)

		objects := []unstructured.Unstructured{
			makeObj("Deployment", "v1", "apps", "deploy", "default"),
			makeObj("Namespace", "v1", "", "ns", ""),
			makeObj("Service", "v1", "", "svc", "default"),
		}

		pr := postrenderer.ApplyOrder(nil)
		result, err := pr(t.Context(), objects)
		g.Expect(err).ToNot(HaveOccurred())

		kinds := make([]string, len(result))
		for i, obj := range result {
			kinds[i] = obj.GetKind()
		}

		// Should use built-in dependency ordering
		g.Expect(kinds).To(Equal([]string{
			"Namespace",
			"Service",
			"Deployment",
		}))
	})

	t.Run("should pass through custom sort results unchanged", func(t *testing.T) {
		g := NewWithT(t)

		objects := []unstructured.Unstructured{
			makeObj("Deployment", "v1", "apps", "deploy", "default"),
			makeObj("Service", "v1", "", "svc", "default"),
		}

		// Custom sort that completely ignores order and returns specific arrangement
		customSort := func(_ []unstructured.Unstructured) []unstructured.Unstructured {
			result := []unstructured.Unstructured{
				makeObj("CustomFirst", "v1", "example.com", "custom", "default"),
				makeObj("CustomSecond", "v1", "example.com", "custom2", "default"),
			}

			return result
		}

		pr := postrenderer.ApplyOrder(customSort)
		result, err := pr(t.Context(), objects)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(HaveLen(2))

		kinds := make([]string, len(result))
		for i, obj := range result {
			kinds[i] = obj.GetKind()
		}

		g.Expect(kinds).To(Equal([]string{
			"CustomFirst",
			"CustomSecond",
		}))
	})
}

// Test practical real-world callback scenarios.
func TestPracticalCallbackScenarios(t *testing.T) {
	t.Run("cert-manager resources before deployments", func(t *testing.T) {
		g := NewWithT(t)

		objects := []unstructured.Unstructured{
			makeObj("Deployment", "v1", "apps", "my-app", "default"),
			makeObj("Service", "v1", "", "my-svc", "default"),
			makeObj("Certificate", "v1", "cert-manager.io", "tls-cert", "default"),
			makeObj("Issuer", "v1", "cert-manager.io", "letsencrypt", "default"),
			makeObj("ClusterIssuer", "v1", "cert-manager.io", "cluster-issuer", ""),
			makeObj("StatefulSet", "v1", "apps", "my-sts", "default"),
		}

		pr := postrenderer.ApplyOrder(certManagerBeforeDeployments)
		result, err := pr(t.Context(), objects)
		g.Expect(err).ToNot(HaveOccurred())

		kinds := make([]string, len(result))
		for i, obj := range result {
			kinds[i] = obj.GetKind()
		}

		// Verify Service comes first
		g.Expect(kinds[0]).To(Equal("Service"))

		// Verify cert-manager resources come after Service but before Deployments
		serviceIndex := indexOf(kinds, "Service")
		deploymentIndex := indexOf(kinds, "Deployment")
		statefulSetIndex := indexOf(kinds, "StatefulSet")

		for _, certKind := range []string{"Issuer", "ClusterIssuer", "Certificate"} {
			certIndex := indexOf(kinds, certKind)
			g.Expect(certIndex).To(BeNumerically(">", serviceIndex), "%s should come after Service", certKind)
			g.Expect(certIndex).To(BeNumerically("<", deploymentIndex), "%s should come before Deployment", certKind)
			g.Expect(certIndex).To(BeNumerically("<", statefulSetIndex), "%s should come before StatefulSet", certKind)
		}
	})

	t.Run("istio resources before deployments", func(t *testing.T) {
		g := NewWithT(t)

		objects := []unstructured.Unstructured{
			makeObj("Deployment", "v1", "apps", "my-app", "default"),
			makeObj("Service", "v1", "", "my-svc", "default"),
			makeObj("Gateway", "v1", "networking.istio.io", "my-gateway", "default"),
			makeObj("VirtualService", "v1", "networking.istio.io", "my-vs", "default"),
			makeObj("DestinationRule", "v1", "networking.istio.io", "my-dr", "default"),
		}

		pr := postrenderer.ApplyOrder(istioBeforeDeployments)
		result, err := pr(t.Context(), objects)
		g.Expect(err).ToNot(HaveOccurred())

		kinds := make([]string, len(result))
		for i, obj := range result {
			kinds[i] = obj.GetKind()
		}

		// Verify Service comes first
		g.Expect(kinds[0]).To(Equal("Service"))

		// Verify Istio resources come after Service but before Deployments
		serviceIndex := indexOf(kinds, "Service")
		deploymentIndex := indexOf(kinds, "Deployment")

		for _, istioKind := range []string{"Gateway", "VirtualService", "DestinationRule"} {
			istioIndex := indexOf(kinds, istioKind)
			g.Expect(istioIndex).To(BeNumerically(">", serviceIndex), "Istio %s should come after Service", istioKind)
			g.Expect(istioIndex).To(BeNumerically("<", deploymentIndex), "Istio %s should come before Deployment", istioKind)
		}
	})

	t.Run("multiple api groups in same gap", func(t *testing.T) {
		g := NewWithT(t)

		objects := []unstructured.Unstructured{
			makeObj("Deployment", "v1", "apps", "my-app", "default"),
			makeObj("Service", "v1", "", "my-svc", "default"),
			// Cert-manager
			makeObj("Certificate", "v1", "cert-manager.io", "tls-cert", "default"),
			makeObj("Issuer", "v1", "cert-manager.io", "issuer", "default"),
			// Istio
			makeObj("Gateway", "v1", "networking.istio.io", "gateway", "default"),
			makeObj("VirtualService", "v1", "networking.istio.io", "vs", "default"),
			// ArgoCD
			makeObj("Application", "v1", "argoproj.io", "app", "default"),
			// Operators
			makeObj("Subscription", "v1", "operators.coreos.com", "sub", "default"),
		}

		pr := postrenderer.ApplyOrder(multipleGroupsBeforeDeployments)
		result, err := pr(t.Context(), objects)
		g.Expect(err).ToNot(HaveOccurred())

		kinds := make([]string, len(result))
		for i, obj := range result {
			kinds[i] = obj.GetKind()
		}

		// All custom resources should be between Service and Deployment
		serviceIndex := indexOf(kinds, "Service")
		deploymentIndex := indexOf(kinds, "Deployment")

		customKinds := []string{"Certificate", "Issuer", "Gateway", "VirtualService", "Application", "Subscription"}
		for _, kind := range customKinds {
			customIndex := indexOf(kinds, kind)
			g.Expect(customIndex).To(BeNumerically(">", serviceIndex), "%s should come after Service", kind)
			g.Expect(customIndex).To(BeNumerically("<", deploymentIndex), "%s should come before Deployment", kind)
		}
	})

	t.Run("validate 10x spacing provides sufficient gaps", func(t *testing.T) {
		g := NewWithT(t)

		// Test that we have enough room between built-in resources
		objects := []unstructured.Unstructured{
			makeObj("Service", "v1", "", "svc", "default"),
			makeObj("LimitRange", "v1", "", "lr", "default"),
			// Insert 8 custom resources in the gap (Service: -90, LimitRange: -80)
			makeObj("Custom1", "v1", "example1.io", "c1", "default"),
			makeObj("Custom2", "v1", "example2.io", "c2", "default"),
			makeObj("Custom3", "v1", "example3.io", "c3", "default"),
			makeObj("Custom4", "v1", "example4.io", "c4", "default"),
			makeObj("Custom5", "v1", "example5.io", "c5", "default"),
			makeObj("Custom6", "v1", "example6.io", "c6", "default"),
			makeObj("Custom7", "v1", "example7.io", "c7", "default"),
			makeObj("Custom8", "v1", "example8.io", "c8", "default"),
		}

		pr := postrenderer.ApplyOrder(gapFillingCallback)
		result, err := pr(t.Context(), objects)
		g.Expect(err).ToNot(HaveOccurred())

		kinds := make([]string, len(result))
		for i, obj := range result {
			kinds[i] = obj.GetKind()
		}

		// Service should be first, LimitRange should be last
		g.Expect(kinds[0]).To(Equal("Service"))
		g.Expect(kinds[len(kinds)-1]).To(Equal("LimitRange"))

		// All custom resources should be between Service and LimitRange
		serviceIndex := indexOf(kinds, "Service")
		limitRangeIndex := indexOf(kinds, "LimitRange")

		for i := 1; i <= 8; i++ {
			customKind := fmt.Sprintf("Custom%d", i)
			customIndex := indexOf(kinds, customKind)
			g.Expect(customIndex).To(BeNumerically(">", serviceIndex), "%s should come after Service", customKind)
			g.Expect(customIndex).To(BeNumerically("<", limitRangeIndex), "%s should come before LimitRange", customKind)
		}
	})

	t.Run("edge case - more than 9 resources in gap should sort by gvk", func(t *testing.T) {
		g := NewWithT(t)

		objects := []unstructured.Unstructured{
			makeObj("Service", "v1", "", "svc", "default"),
			makeObj("LimitRange", "v1", "", "lr", "default"),
			// 12 resources - more than the 9 gaps available
			makeObj("Resource01", "v1", "a.example.com", "r1", "default"),
			makeObj("Resource02", "v1", "b.example.com", "r2", "default"),
			makeObj("Resource03", "v1", "c.example.com", "r3", "default"),
			makeObj("Resource04", "v1", "d.example.com", "r4", "default"),
			makeObj("Resource05", "v1", "e.example.com", "r5", "default"),
			makeObj("Resource06", "v1", "f.example.com", "r6", "default"),
			makeObj("Resource07", "v1", "g.example.com", "r7", "default"),
			makeObj("Resource08", "v1", "h.example.com", "r8", "default"),
			makeObj("Resource09", "v1", "i.example.com", "r9", "default"),
			makeObj("Resource10", "v1", "j.example.com", "r10", "default"),
			makeObj("Resource11", "v1", "k.example.com", "r11", "default"),
			makeObj("Resource12", "v1", "l.example.com", "r12", "default"),
		}

		pr := postrenderer.ApplyOrder(overflowGapCallback)
		result, err := pr(t.Context(), objects)
		g.Expect(err).ToNot(HaveOccurred())

		kinds := make([]string, len(result))
		for i, obj := range result {
			kinds[i] = obj.GetKind()
		}

		// Service first, LimitRange last
		g.Expect(kinds[0]).To(Equal("Service"))
		g.Expect(kinds[len(kinds)-1]).To(Equal("LimitRange"))

		// Resources should be sorted by GVK for stable order
		serviceIndex := indexOf(kinds, "Service")
		limitRangeIndex := indexOf(kinds, "LimitRange")

		// Extract custom resource names and verify they're sorted
		var customKinds []string
		for i := serviceIndex + 1; i < limitRangeIndex; i++ {
			customKinds = append(customKinds, kinds[i])
		}

		// Should be in GVK order (which is alphabetical by group for our test data)
		expected := []string{"Resource01", "Resource02", "Resource03", "Resource04", "Resource05",
			"Resource06", "Resource07", "Resource08", "Resource09", "Resource10", "Resource11", "Resource12"}
		g.Expect(customKinds).To(Equal(expected))
	})
}

// Helper callback functions for tests

func certManagerBeforeDeployments(objects []unstructured.Unstructured) []unstructured.Unstructured {
	result := make([]unstructured.Unstructured, len(objects))
	copy(result, objects)

	sort.SliceStable(result, func(i, j int) bool {
		return compareCertManagerOrder(result[i], result[j])
	})

	return result
}

func compareCertManagerOrder(a, b unstructured.Unstructured) bool {
	orderA := getCertManagerOrder(a)
	orderB := getCertManagerOrder(b)

	if orderA != orderB {
		return orderA < orderB
	}

	return fallbackCompare(a, b)
}

func getCertManagerOrder(obj unstructured.Unstructured) int {
	gvk := obj.GroupVersionKind()

	// Place cert-manager between Service (-90) and LimitRange (-80)
	if gvk.Group == "cert-manager.io" {
		return -85 // Between Service and LimitRange
	}

	// Built-in resource ordering (replicated from main package)
	return getTestBuiltinOrder(obj.GetKind())
}

func istioBeforeDeployments(objects []unstructured.Unstructured) []unstructured.Unstructured {
	result := make([]unstructured.Unstructured, len(objects))
	copy(result, objects)

	sort.SliceStable(result, func(i, j int) bool {
		orderA := getIstioOrder(result[i])
		orderB := getIstioOrder(result[j])

		if orderA != orderB {
			return orderA < orderB
		}

		return fallbackCompare(result[i], result[j])
	})

	return result
}

func getIstioOrder(obj unstructured.Unstructured) int {
	gvk := obj.GroupVersionKind()

	// Place Istio between Service (-90) and LimitRange (-80)
	if gvk.Group == "networking.istio.io" {
		return -82 // Between Service and LimitRange, after cert-manager
	}

	return getTestBuiltinOrder(obj.GetKind())
}

func multipleGroupsBeforeDeployments(objects []unstructured.Unstructured) []unstructured.Unstructured {
	result := make([]unstructured.Unstructured, len(objects))
	copy(result, objects)

	sort.SliceStable(result, func(i, j int) bool {
		orderA := getMultiGroupOrder(result[i])
		orderB := getMultiGroupOrder(result[j])

		if orderA != orderB {
			return orderA < orderB
		}

		return fallbackCompare(result[i], result[j])
	})

	return result
}

func getMultiGroupOrder(obj unstructured.Unstructured) int {
	gvk := obj.GroupVersionKind()

	// Assign different positions for different groups
	switch gvk.Group {
	case "cert-manager.io":
		return -85
	case "networking.istio.io":
		return -84
	case "argoproj.io":
		return -83
	case "operators.coreos.com":
		return -82
	}

	return getTestBuiltinOrder(obj.GetKind())
}

func gapFillingCallback(objects []unstructured.Unstructured) []unstructured.Unstructured {
	result := make([]unstructured.Unstructured, len(objects))
	copy(result, objects)

	sort.SliceStable(result, func(i, j int) bool {
		orderA := getGapFillingOrder(result[i])
		orderB := getGapFillingOrder(result[j])

		if orderA != orderB {
			return orderA < orderB
		}

		return fallbackCompare(result[i], result[j])
	})

	return result
}

func getGapFillingOrder(obj unstructured.Unstructured) int {
	kind := obj.GetKind()

	// Fill gaps between Service (-90) and LimitRange (-80) with positions -89 to -81
	switch kind {
	case "Custom1":
		return -89
	case "Custom2":
		return -88
	case "Custom3":
		return -87
	case "Custom4":
		return -86
	case "Custom5":
		return -85
	case "Custom6":
		return -84
	case "Custom7":
		return -83
	case "Custom8":
		return -82
	}

	return getTestBuiltinOrder(kind)
}

func overflowGapCallback(objects []unstructured.Unstructured) []unstructured.Unstructured {
	result := make([]unstructured.Unstructured, len(objects))
	copy(result, objects)

	sort.SliceStable(result, func(i, j int) bool {
		orderA := getOverflowOrder(result[i])
		orderB := getOverflowOrder(result[j])

		if orderA != orderB {
			return orderA < orderB
		}

		return fallbackCompare(result[i], result[j])
	})

	return result
}

func getOverflowOrder(obj unstructured.Unstructured) int {
	gvk := obj.GroupVersionKind()

	// All custom resources get same priority - they'll sort by GVK
	if strings.HasSuffix(gvk.Group, ".example.com") {
		return -85 // Same position for all - forces GVK sorting
	}

	return getTestBuiltinOrder(obj.GetKind())
}

// Test helper functions to replicate main package logic

func getTestBuiltinOrder(kind string) int {
	// Replicate the built-in ordering from the main package (with 10x spacing)
	orderFirst := []string{
		"Namespace", "ResourceQuota", "StorageClass", "CustomResourceDefinition",
		"ServiceAccount", "PodSecurityPolicy", "Role", "ClusterRole",
		"RoleBinding", "ClusterRoleBinding", "ConfigMap", "Secret",
		"Endpoints", "Service", "LimitRange", "PriorityClass",
		"PersistentVolume", "PersistentVolumeClaim", "Deployment",
		"StatefulSet", "CronJob", "PodDisruptionBudget",
	}

	orderLast := []string{
		"MutatingWebhookConfiguration", "ValidatingWebhookConfiguration",
	}

	for i, k := range orderFirst {
		if k == kind {
			return (i - len(orderFirst)) * 10
		}
	}

	for i, k := range orderLast {
		if k == kind {
			return (i + 1) * 10
		}
	}

	return 0 // Unknown kinds default to 0
}

func fallbackCompare(a, b unstructured.Unstructured) bool {
	// Replicate the fallback comparison logic from main package
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

// indexOf returns the index of target string in slice, or -1 if not found.
func indexOf(slice []string, target string) int {
	for i, item := range slice {
		if item == target {
			return i
		}
	}

	return -1
}

// Test critical edge cases and tiebreaker scenarios.
func TestCriticalEdgeCases(t *testing.T) {
	t.Run("should handle gvk tiebreaker for same kind different groups", func(t *testing.T) {
		g := NewWithT(t)

		objects := []unstructured.Unstructured{
			makeObj("Service", "v1", "kserve.io", "inference", "default"),      // kserve.io/v1/Service
			makeObj("Service", "v1", "", "regular", "default"),                 // v1/Service (core)
			makeObj("Service", "v1beta1", "knative.dev", "knative", "default"), // knative.dev/v1beta1/Service
		}

		pr := postrenderer.ApplyOrder(nil)
		result, err := pr(t.Context(), objects)
		g.Expect(err).ToNot(HaveOccurred())

		kinds := make([]string, len(result))
		groups := make([]string, len(result))
		for i, obj := range result {
			kinds[i] = obj.GetKind()
			groups[i] = obj.GroupVersionKind().Group
		}

		// All should be Service, but ordered by GVK string
		g.Expect(kinds).To(Equal([]string{"Service", "Service", "Service"}))

		// Should be ordered: "" (core) < "knative.dev" < "kserve.io"
		g.Expect(groups[0]).To(Equal(""))            // Core Service first
		g.Expect(groups[1]).To(Equal("knative.dev")) // knative.dev second
		g.Expect(groups[2]).To(Equal("kserve.io"))   // kserve.io last
	})

	t.Run("should handle namespace tiebreaker for cluster-scoped vs namespaced", func(t *testing.T) {
		g := NewWithT(t)

		objects := []unstructured.Unstructured{
			makeObj("MyResource", "v1", "example.com", "resource-ns", "kube-system"),
			makeObj("MyResource", "v1", "example.com", "resource-default", "default"),
			makeObj("MyResource", "v1", "example.com", "resource-cluster", ""), // Cluster-scoped
		}

		pr := postrenderer.ApplyOrder(nil)
		result, err := pr(t.Context(), objects)
		g.Expect(err).ToNot(HaveOccurred())

		// Should be ordered by namespace: "" < "default" < "kube-system"
		g.Expect(result[0].GetNamespace()).To(Equal(""))            // Cluster-scoped first
		g.Expect(result[1].GetNamespace()).To(Equal("default"))     // default namespace second
		g.Expect(result[2].GetNamespace()).To(Equal("kube-system")) // kube-system last
	})

	t.Run("should handle name tiebreaker within same gvk and namespace", func(t *testing.T) {
		g := NewWithT(t)

		objects := []unstructured.Unstructured{
			makeObj("Pod", "v1", "", "pod-zebra", "default"),
			makeObj("Pod", "v1", "", "pod-alpha", "default"),
			makeObj("Pod", "v1", "", "pod-beta", "default"),
		}

		pr := postrenderer.ApplyOrder(nil)
		result, err := pr(t.Context(), objects)
		g.Expect(err).ToNot(HaveOccurred())

		// Should be ordered alphabetically by name
		g.Expect(result[0].GetName()).To(Equal("pod-alpha"))
		g.Expect(result[1].GetName()).To(Equal("pod-beta"))
		g.Expect(result[2].GetName()).To(Equal("pod-zebra"))
	})

	t.Run("should handle unknown kinds with zero priority", func(t *testing.T) {
		g := NewWithT(t)

		objects := []unstructured.Unstructured{
			makeObj("Service", "v1", "", "known", "default"),                                             // Priority -90
			makeObj("UnknownKind", "v1", "example.com", "unknown", "default"),                            // Priority 0
			makeObj("MutatingWebhookConfiguration", "v1", "admissionregistration.k8s.io", "webhook", ""), // Priority 10
		}

		pr := postrenderer.ApplyOrder(nil)
		result, err := pr(t.Context(), objects)
		g.Expect(err).ToNot(HaveOccurred())

		kinds := make([]string, len(result))
		for i, obj := range result {
			kinds[i] = obj.GetKind()
		}

		// Should be: Service (-90) → UnknownKind (0) → MutatingWebhook (10)
		g.Expect(kinds).To(Equal([]string{
			"Service",
			"UnknownKind",
			"MutatingWebhookConfiguration",
		}))
	})

	t.Run("should validate 10x spacing creates correct positions", func(t *testing.T) {
		g := NewWithT(t)

		// Test the actual spacing values match expectations
		objects := []unstructured.Unstructured{
			makeObj("Namespace", "v1", "", "ns", ""),      // Should be -220 (first in orderFirst)
			makeObj("Service", "v1", "", "svc", "default"), // Should be -90
			makeObj("Deployment", "v1", "apps", "deploy", "default"), // Should be -40
			makeObj("MutatingWebhookConfiguration", "v1", "admissionregistration.k8s.io", "mwh", ""), // Should be 10
		}

		// Create a custom callback that validates specific positions
		validatePositions := func(objs []unstructured.Unstructured) []unstructured.Unstructured {
			result := make([]unstructured.Unstructured, len(objs))
			copy(result, objs)

			sort.SliceStable(result, func(i, j int) bool {
				orderA := getTestBuiltinOrder(result[i].GetKind())
				orderB := getTestBuiltinOrder(result[j].GetKind())

				// Validate expected positions
				if result[i].GetKind() == "Namespace" {
					g.Expect(orderA).To(Equal(-220), "Namespace should be at position -220")
				}
				if result[i].GetKind() == "Service" {
					g.Expect(orderA).To(Equal(-90), "Service should be at position -90")
				}
				if result[i].GetKind() == "Deployment" {
					g.Expect(orderA).To(Equal(-40), "Deployment should be at position -40")
				}
				if result[i].GetKind() == "MutatingWebhookConfiguration" {
					g.Expect(orderA).To(Equal(10), "MutatingWebhookConfiguration should be at position 10")
				}

				if orderA != orderB {
					return orderA < orderB
				}

				return fallbackCompare(result[i], result[j])
			})

			return result
		}

		pr := postrenderer.ApplyOrder(validatePositions)
		_, err := pr(t.Context(), objects)
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("should handle callback returning different object count", func(t *testing.T) {
		g := NewWithT(t)

		objects := []unstructured.Unstructured{
			makeObj("Service", "v1", "", "svc1", "default"),
			makeObj("Service", "v1", "", "svc2", "default"),
		}

		// Callback that filters objects
		filterCallback := func(_ []unstructured.Unstructured) []unstructured.Unstructured {
			// Return only one object
			return []unstructured.Unstructured{
				makeObj("FilteredResource", "v1", "example.com", "filtered", "default"),
			}
		}

		pr := postrenderer.ApplyOrder(filterCallback)
		result, err := pr(t.Context(), objects)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(HaveLen(1))
		g.Expect(result[0].GetKind()).To(Equal("FilteredResource"))
	})

	t.Run("should handle callback returning empty slice", func(t *testing.T) {
		g := NewWithT(t)

		objects := []unstructured.Unstructured{
			makeObj("Service", "v1", "", "svc", "default"),
		}

		// Callback that returns nothing
		emptyCallback := func(_ []unstructured.Unstructured) []unstructured.Unstructured {
			return []unstructured.Unstructured{}
		}

		pr := postrenderer.ApplyOrder(emptyCallback)
		result, err := pr(t.Context(), objects)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeEmpty())
	})


	t.Run("should maintain stable sort for identical objects", func(t *testing.T) {
		g := NewWithT(t)

		objects := []unstructured.Unstructured{
			makeObj("Pod", "v1", "", "same-pod", "default"),
			makeObj("Pod", "v1", "", "same-pod", "default"), // Identical object
			makeObj("Pod", "v1", "", "same-pod", "default"), // Identical object
		}

		pr := postrenderer.ApplyOrder(nil)
		result, err := pr(t.Context(), objects)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(HaveLen(3))

		// All should remain pods with same name (stable sort preserves input order for identical items)
		for i := range 3 {
			g.Expect(result[i].GetKind()).To(Equal("Pod"))
			g.Expect(result[i].GetName()).To(Equal("same-pod"))
		}
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
