package jq_test

import (
	"fmt"
	"testing"

	jqu "github.com/k8s-manifest-kit/pkg/util/jq"
	jqmatcher "github.com/lburgazzoli/gomega-matchers/pkg/matchers/jq"
	"github.com/onsi/gomega/types"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/k8s-manifest-kit/engine/pkg/transformer/jq"

	. "github.com/onsi/gomega"
)

func toUnstructured(t *testing.T, obj runtime.Object) unstructured.Unstructured {
	t.Helper()

	unstr, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)

	NewWithT(t).Expect(err).ShouldNot(HaveOccurred())

	return unstructured.Unstructured{Object: unstr}
}

func TestTransformer(t *testing.T) {
	g := NewWithT(t)

	tests := []struct {
		name           string
		expression     string
		opts           []jqu.Option
		inputObject    runtime.Object
		validation     types.GomegaMatcher
		expectNewErr   bool
		expectTransErr bool
	}{
		{
			name:       "should transform object using simple expression",
			expression: `.metadata.labels["new-key"] = "new-value"`,
			inputObject: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"key1": "value1"},
				},
			},
			validation: And(
				jqmatcher.Match(`.metadata.labels["key1"] == "value1"`),
				jqmatcher.Match(`.metadata.labels["new-key"] == "new-value"`),
			),
		},
		{
			name:       "should transform object using complex expression",
			expression: `.metadata.labels += {"key2": "value2"} | .metadata.annotations = {"anno1": "value1"}`,
			inputObject: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"key1": "value1"},
				},
			},
			validation: And(
				jqmatcher.Match(`.metadata.labels["key1"] == "value1"`),
				jqmatcher.Match(`.metadata.labels["key2"] == "value2"`),
				jqmatcher.Match(`.metadata.annotations["anno1"] == "value1"`),
			),
		},
		{
			name:       "should handle invalid JQ expression",
			expression: `invalid jq expression`,
			inputObject: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{},
			},
			expectNewErr: true,
		},
		{
			name:       "should handle expression that returns non-object",
			expression: `"string"`,
			inputObject: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{},
			},
			expectTransErr: true,
		},
		{
			name:       "should handle expression that returns no results",
			expression: `empty`,
			inputObject: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{},
			},
			expectTransErr: true,
		},
		{
			name:       "should transform nested fields",
			expression: `.spec.template.spec.containers[0].image = "new-image:tag"`,
			inputObject: &corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image: "old-image:tag",
						},
					},
				},
			},
			validation: And(
				jqmatcher.Match(`.spec.template.spec.containers[0].image == "new-image:tag"`),
				jqmatcher.Match(`.spec.template.spec.containers | length == 1`),
			),
		},
		{
			name:       "should use custom function to transform content",
			expression: `addPrefixToLabels("env-")`,
			opts: []jqu.Option{
				jqu.WithFunction("addPrefixToLabels", 1, 1, func(input any, args []any) any {
					obj, ok := input.(map[string]any)
					if !ok {
						return fmt.Errorf("expected object, got %T", input)
					}

					prefix, ok := args[0].(string)
					if !ok {
						return fmt.Errorf("expected string prefix, got %T", args[0])
					}

					// Get metadata.labels
					metadata, ok := obj["metadata"].(map[string]any)
					if !ok {
						return fmt.Errorf("expected metadata object, got %T", obj["metadata"])
					}

					labels, ok := metadata["labels"].(map[string]any)
					if !ok {
						return fmt.Errorf("expected labels object, got %T", metadata["labels"])
					}

					// Create new labels with prefix
					newLabels := make(map[string]any)
					for k, v := range labels {
						newLabels[prefix+k] = v
					}

					// Update the labels
					metadata["labels"] = newLabels

					return obj
				}),
			},
			inputObject: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"key1": "value1",
						"key2": "value2",
					},
				},
			},
			validation: And(
				jqmatcher.Match(`.metadata.labels["env-key1"] == "value1"`),
				jqmatcher.Match(`.metadata.labels["env-key2"] == "value2"`),
				jqmatcher.Match(`.metadata.labels | length == 2`),
			),
		},
		{
			name:       "should use custom function to transform labels",
			expression: `.metadata.labels = (.metadata.labels | addPrefixToLabels("env-"))`,
			opts: []jqu.Option{
				jqu.WithFunction("addPrefixToLabels", 1, 1, func(input any, args []any) any {
					prefix := args[0].(string)

					labels, ok := input.(map[string]any)
					if !ok {
						return nil
					}

					result := make(map[string]any)
					for k, v := range labels {
						result[prefix+k] = v
					}

					return result
				}),
			},
			inputObject: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cm",
					Labels: map[string]string{
						"app":     "myapp",
						"version": "v1",
					},
				},
				Data: map[string]string{
					"key": "value",
				},
			},
			validation: And(
				jqmatcher.Match(`.metadata.labels["env-app"] == "myapp"`),
				jqmatcher.Match(`.metadata.labels["env-version"] == "v1"`),
				jqmatcher.Match(`.metadata.labels | length == 2`),
			),
		},
		{
			name:       "should use variables in expression",
			expression: `.data.greeting = $greeting`,
			opts: []jqu.Option{
				jqu.WithVariable("greeting", "Hello, World!"),
			},
			inputObject: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cm",
					Namespace: "default",
				},
				Data: map[string]string{
					"existing": "value",
				},
			},
			validation: And(
				jqmatcher.Match(`.data.greeting == "Hello, World!"`),
				jqmatcher.Match(`.data.existing == "value"`),
			),
		},
		{
			name:       "should use multiple variables in expression",
			expression: `setpath(["data", "greeting"]; $greeting) | setpath(["data", "count"]; $count)`,
			opts: []jqu.Option{
				jqu.WithVariable("greeting", "Hello, World!"),
				jqu.WithVariable("count", 42),
			},
			inputObject: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cm",
					Namespace: "default",
				},
				Data: map[string]string{
					"existing": "value",
				},
			},
			validation: And(
				jqmatcher.Match(`.data.greeting == "Hello, World!"`),
				jqmatcher.Match(`.data.count == 42`),
				jqmatcher.Match(`.data.existing == "value"`),
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transformer, err := jq.Transform(tt.expression, tt.opts...)
			if tt.expectNewErr {
				g.Expect(err).To(HaveOccurred())

				return
			}

			g.Expect(err).ToNot(HaveOccurred())

			unstrObj := toUnstructured(t, tt.inputObject)

			transformed, err := transformer(t.Context(), unstrObj)
			if tt.expectTransErr {
				g.Expect(err).To(HaveOccurred())

				return
			}

			g.Expect(err).ToNot(HaveOccurred())

			if tt.validation != nil {
				g.Expect(transformed.Object).To(tt.validation)
			}
		})
	}
}
