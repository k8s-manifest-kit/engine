package pipeline_test

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	kindPod        = "Pod"
	labelValueTrue = "true"
)

// Helper functions

func makeObject(kind string, name string) unstructured.Unstructured {
	return unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       kind,
			"metadata": map[string]any{
				"name": name,
			},
		},
	}
}

func makeObjectWithNamespace(kind string, name string, namespace string) unstructured.Unstructured {
	return unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       kind,
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
			},
		},
	}
}
