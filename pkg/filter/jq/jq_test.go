package jq_test

import (
	"testing"

	utiljq "github.com/k8s-manifest-kit/pkg/util/jq"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/k8s-manifest-kit/engine/pkg/filter/jq"

	. "github.com/onsi/gomega"
)

func TestFilter(t *testing.T) {
	ctx := t.Context()

	t.Run("should filter by kind", func(t *testing.T) {
		g := NewWithT(t)
		filter, err := jq.Filter(`.kind == "Pod"`)
		g.Expect(err).ToNot(HaveOccurred())

		pod := unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "v1",
				"kind":       "Pod",
				"metadata": map[string]any{
					"name": "test-pod",
				},
			},
		}

		result, err := filter(ctx, pod)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeTrue())

		service := unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "v1",
				"kind":       "Service",
				"metadata": map[string]any{
					"name": "test-service",
				},
			},
		}

		result, err = filter(ctx, service)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeFalse())
	})

	t.Run("should filter by namespace", func(t *testing.T) {
		g := NewWithT(t)
		filter, err := jq.Filter(`.metadata.namespace == "default"`)
		g.Expect(err).ToNot(HaveOccurred())

		defaultNs := unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "v1",
				"kind":       "Pod",
				"metadata": map[string]any{
					"name":      "test-pod",
					"namespace": "default",
				},
			},
		}

		result, err := filter(ctx, defaultNs)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeTrue())

		kubeSystem := unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "v1",
				"kind":       "Pod",
				"metadata": map[string]any{
					"name":      "test-pod",
					"namespace": "kube-system",
				},
			},
		}

		result, err = filter(ctx, kubeSystem)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeFalse())
	})

	t.Run("should filter by label", func(t *testing.T) {
		g := NewWithT(t)
		filter, err := jq.Filter(`.metadata.labels.app == "nginx"`)
		g.Expect(err).ToNot(HaveOccurred())

		withLabel := unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "v1",
				"kind":       "Pod",
				"metadata": map[string]any{
					"name": "test-pod",
					"labels": map[string]any{
						"app": "nginx",
					},
				},
			},
		}

		result, err := filter(ctx, withLabel)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeTrue())

		withoutLabel := unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "v1",
				"kind":       "Pod",
				"metadata": map[string]any{
					"name": "test-pod",
				},
			},
		}

		result, err = filter(ctx, withoutLabel)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeFalse())
	})

	t.Run("should filter by complex expression", func(t *testing.T) {
		g := NewWithT(t)
		filter, err := jq.Filter(`.kind == "Deployment" and .spec.replicas > 1`)
		g.Expect(err).ToNot(HaveOccurred())

		matching := unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"metadata": map[string]any{
					"name": "test-deployment",
				},
				"spec": map[string]any{
					"replicas": float64(3),
				},
			},
		}

		result, err := filter(ctx, matching)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeTrue())

		notMatching := unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"metadata": map[string]any{
					"name": "test-deployment",
				},
				"spec": map[string]any{
					"replicas": float64(1),
				},
			},
		}

		result, err = filter(ctx, notMatching)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeFalse())
	})

	t.Run("should filter with or logic", func(t *testing.T) {
		g := NewWithT(t)
		filter, err := jq.Filter(`.kind == "Pod" or .kind == "Service"`)
		g.Expect(err).ToNot(HaveOccurred())

		pod := unstructured.Unstructured{
			Object: map[string]any{
				"kind": "Pod",
			},
		}

		result, err := filter(ctx, pod)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeTrue())

		service := unstructured.Unstructured{
			Object: map[string]any{
				"kind": "Service",
			},
		}

		result, err = filter(ctx, service)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeTrue())

		deployment := unstructured.Unstructured{
			Object: map[string]any{
				"kind": "Deployment",
			},
		}

		result, err = filter(ctx, deployment)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeFalse())
	})

	t.Run("should filter with has function", func(t *testing.T) {
		g := NewWithT(t)
		filter, err := jq.Filter(`has("metadata") and (.metadata | has("labels"))`)
		g.Expect(err).ToNot(HaveOccurred())

		withLabels := unstructured.Unstructured{
			Object: map[string]any{
				"metadata": map[string]any{
					"labels": map[string]any{
						"app": "test",
					},
				},
			},
		}

		result, err := filter(ctx, withLabels)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeTrue())

		withoutLabels := unstructured.Unstructured{
			Object: map[string]any{
				"metadata": map[string]any{
					"name": "test",
				},
			},
		}

		result, err = filter(ctx, withoutLabels)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeFalse())
	})

	t.Run("should filter with variable", func(t *testing.T) {
		g := NewWithT(t)
		filter, err := jq.Filter(
			`.kind == $expectedKind`,
			utiljq.WithVariable("expectedKind", "Pod"),
		)
		g.Expect(err).ToNot(HaveOccurred())

		pod := unstructured.Unstructured{
			Object: map[string]any{
				"kind": "Pod",
			},
		}

		result, err := filter(ctx, pod)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeTrue())
	})

	t.Run("should return error for invalid expression", func(t *testing.T) {
		g := NewWithT(t)
		filter, err := jq.Filter(`invalid jq expression[[[`)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("error creating jq engine"))
		g.Expect(filter).To(BeNil())
	})

	t.Run("should return error for non-boolean result", func(t *testing.T) {
		g := NewWithT(t)
		filter, err := jq.Filter(`.kind`)
		g.Expect(err).ToNot(HaveOccurred())

		obj := unstructured.Unstructured{
			Object: map[string]any{
				"kind": "Pod",
			},
		}

		result, err := filter(ctx, obj)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("must return a boolean"))
		g.Expect(result).To(BeFalse())
	})

	t.Run("should return error for execution failure", func(t *testing.T) {
		g := NewWithT(t)
		filter, err := jq.Filter(`.value / 0`)
		g.Expect(err).ToNot(HaveOccurred())

		obj := unstructured.Unstructured{
			Object: map[string]any{
				"value": float64(10),
			},
		}

		result, err := filter(ctx, obj)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("error executing jq expression"))
		g.Expect(result).To(BeFalse())
	})

	t.Run("should filter by apiVersion", func(t *testing.T) {
		g := NewWithT(t)
		filter, err := jq.Filter(`.apiVersion == "apps/v1"`)
		g.Expect(err).ToNot(HaveOccurred())

		appsV1 := unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
			},
		}

		result, err := filter(ctx, appsV1)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeTrue())

		coreV1 := unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "v1",
				"kind":       "Pod",
			},
		}

		result, err = filter(ctx, coreV1)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeFalse())
	})

	t.Run("should handle null values gracefully", func(t *testing.T) {
		g := NewWithT(t)
		filter, err := jq.Filter(`.metadata.annotations.special == null`)
		g.Expect(err).ToNot(HaveOccurred())

		withoutAnnotation := unstructured.Unstructured{
			Object: map[string]any{
				"metadata": map[string]any{
					"name": "test",
				},
			},
		}

		result, err := filter(ctx, withoutAnnotation)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeTrue())
	})
}
