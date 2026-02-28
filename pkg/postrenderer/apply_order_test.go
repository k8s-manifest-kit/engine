package postrenderer_test

import (
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

		pr := postrenderer.ApplyOrder()
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

		pr := postrenderer.ApplyOrder()
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

		pr := postrenderer.ApplyOrder()
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

		pr := postrenderer.ApplyOrder()
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

		pr := postrenderer.ApplyOrder()
		result, err := pr(t.Context(), []unstructured.Unstructured{})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeEmpty())
	})

	t.Run("should handle single object", func(t *testing.T) {
		g := NewWithT(t)

		objects := []unstructured.Unstructured{
			makeObj("Pod", "v1", "", "my-pod", "default"),
		}

		pr := postrenderer.ApplyOrder()
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

		pr := postrenderer.ApplyOrder()
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
