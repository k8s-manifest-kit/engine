package gvk_test

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/k8s-manifest-kit/engine/pkg/filter/meta/gvk"

	. "github.com/onsi/gomega"
)

func TestFilter(t *testing.T) {
	ctx := t.Context()

	t.Run("should filter single GVK", func(t *testing.T) {
		g := NewWithT(t)
		filter := gvk.Filter(corev1.SchemeGroupVersion.WithKind("Pod"))

		pod := makeObject("v1", "Pod", "test-pod")
		result, err := filter(ctx, pod)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeTrue())

		service := makeObject("v1", "Service", "test-service")
		result, err = filter(ctx, service)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeFalse())
	})

	t.Run("should filter multiple GVKs", func(t *testing.T) {
		g := NewWithT(t)
		filter := gvk.Filter(
			corev1.SchemeGroupVersion.WithKind("Pod"),
			corev1.SchemeGroupVersion.WithKind("Service"),
		)

		pod := makeObject("v1", "Pod", "test-pod")
		result, err := filter(ctx, pod)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeTrue())

		service := makeObject("v1", "Service", "test-service")
		result, err = filter(ctx, service)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeTrue())

		configMap := makeObject("v1", "ConfigMap", "test-config")
		result, err = filter(ctx, configMap)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeFalse())
	})

	t.Run("should filter apps/v1 resources", func(t *testing.T) {
		g := NewWithT(t)
		filter := gvk.Filter(
			appsv1.SchemeGroupVersion.WithKind("Deployment"),
			appsv1.SchemeGroupVersion.WithKind("StatefulSet"),
		)

		deployment := makeObject("apps/v1", "Deployment", "test-deployment")
		result, err := filter(ctx, deployment)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeTrue())

		statefulSet := makeObject("apps/v1", "StatefulSet", "test-statefulset")
		result, err = filter(ctx, statefulSet)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeTrue())

		daemonSet := makeObject("apps/v1", "DaemonSet", "test-daemonset")
		result, err = filter(ctx, daemonSet)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeFalse())
	})

	t.Run("should distinguish between different API versions", func(t *testing.T) {
		g := NewWithT(t)
		filter := gvk.Filter(appsv1.SchemeGroupVersion.WithKind("Deployment"))

		appsV1Deployment := makeObject("apps/v1", "Deployment", "test-deployment")
		result, err := filter(ctx, appsV1Deployment)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeTrue())

		// apps/v1beta1 should not match
		appsV1Beta1Deployment := makeObject("apps/v1beta1", "Deployment", "test-deployment")
		result, err = filter(ctx, appsV1Beta1Deployment)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeFalse())
	})

	t.Run("should handle empty GVK list", func(t *testing.T) {
		g := NewWithT(t)
		filter := gvk.Filter()

		pod := makeObject("v1", "Pod", "test-pod")
		result, err := filter(ctx, pod)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeFalse())
	})

	t.Run("should handle custom resources", func(t *testing.T) {
		g := NewWithT(t)
		customGVK := schema.GroupVersionKind{
			Group:   "example.com",
			Version: "v1alpha1",
			Kind:    "MyCustomResource",
		}

		filter := gvk.Filter(customGVK)

		customResource := makeObject("example.com/v1alpha1", "MyCustomResource", "test-custom")
		result, err := filter(ctx, customResource)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeTrue())

		pod := makeObject("v1", "Pod", "test-pod")
		result, err = filter(ctx, pod)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeFalse())
	})

	t.Run("should handle objects without GVK", func(t *testing.T) {
		g := NewWithT(t)
		filter := gvk.Filter(corev1.SchemeGroupVersion.WithKind("Pod"))

		obj := unstructured.Unstructured{
			Object: map[string]any{
				"metadata": map[string]any{
					"name": "test",
				},
			},
		}

		result, err := filter(ctx, obj)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeFalse())
	})

	t.Run("should filter core v1 resources", func(t *testing.T) {
		g := NewWithT(t)
		filter := gvk.Filter(
			corev1.SchemeGroupVersion.WithKind("Pod"),
			corev1.SchemeGroupVersion.WithKind("Service"),
			corev1.SchemeGroupVersion.WithKind("ConfigMap"),
			corev1.SchemeGroupVersion.WithKind("Secret"),
		)

		pod := makeObject("v1", "Pod", "test-pod")
		result, err := filter(ctx, pod)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeTrue())

		service := makeObject("v1", "Service", "test-service")
		result, err = filter(ctx, service)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeTrue())

		configMap := makeObject("v1", "ConfigMap", "test-config")
		result, err = filter(ctx, configMap)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeTrue())

		secret := makeObject("v1", "Secret", "test-secret")
		result, err = filter(ctx, secret)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeTrue())

		deployment := makeObject("apps/v1", "Deployment", "test-deployment")
		result, err = filter(ctx, deployment)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeFalse())
	})

	t.Run("should handle duplicate GVKs", func(t *testing.T) {
		g := NewWithT(t)
		filter := gvk.Filter(
			corev1.SchemeGroupVersion.WithKind("Pod"),
			corev1.SchemeGroupVersion.WithKind("Pod"), // duplicate
		)

		pod := makeObject("v1", "Pod", "test-pod")
		result, err := filter(ctx, pod)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeTrue())
	})

	t.Run("should be case sensitive", func(t *testing.T) {
		g := NewWithT(t)
		filter := gvk.Filter(corev1.SchemeGroupVersion.WithKind("Pod"))

		// Correct case
		pod := makeObject("v1", "Pod", "test-pod")
		result, err := filter(ctx, pod)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeTrue())

		// Wrong case
		podLowercase := makeObject("v1", "pod", "test-pod")
		result, err = filter(ctx, podLowercase)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeFalse())
	})
}

func makeObject(apiVersion string, kind string, name string) unstructured.Unstructured {
	obj := unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": apiVersion,
			"kind":       kind,
			"metadata": map[string]any{
				"name": name,
			},
		},
	}
	// Set the GVK which is what the filter checks
	gv, _ := schema.ParseGroupVersion(apiVersion)
	obj.SetGroupVersionKind(gv.WithKind(kind))

	return obj
}
