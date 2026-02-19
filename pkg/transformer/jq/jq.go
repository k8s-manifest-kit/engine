package jq

import (
	"context"
	"errors"
	"fmt"

	"github.com/k8s-manifest-kit/pkg/util/jq"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/k8s-manifest-kit/engine/pkg/transformer"
	"github.com/k8s-manifest-kit/engine/pkg/types"
)

var (
	// ErrJqMustReturnObject is returned when a JQ expression doesn't return an object.
	ErrJqMustReturnObject = errors.New("jq expression must return an object")
)

// Transform creates a new JQ transformer with the given expression and options.
func Transform(expression string, opts ...jq.Option) (types.Transformer, error) {
	// Create a new JQ engine
	engine, err := jq.NewEngine(expression, opts...)
	if err != nil {
		return nil, fmt.Errorf("error creating jq engine: %w", err)
	}

	return func(_ context.Context, obj unstructured.Unstructured) (unstructured.Unstructured, error) {
		v, err := engine.Run(obj.Object)
		if err != nil {
			return unstructured.Unstructured{}, &transformer.Error{
				Object: obj,
				Err:    fmt.Errorf("error execuring jq expression: %w", err),
			}
		}

		ret := unstructured.Unstructured{}

		switch v := v.(type) {
		case map[string]any:
			data, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&v)
			if err != nil {
				return ret, &transformer.Error{
					Object: obj,
					Err:    fmt.Errorf("failed to convert jq result to unstructured: %w", err),
				}
			}

			ret.SetUnstructuredContent(data)

			return ret, nil
		default:
			return ret, &transformer.Error{
				Object: obj,
				Err:    fmt.Errorf("%w, got %T", ErrJqMustReturnObject, v),
			}
		}
	}, nil
}
