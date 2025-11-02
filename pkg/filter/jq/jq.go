package jq

import (
	"context"
	"errors"
	"fmt"

	"github.com/k8s-manifest-kit/pkg/util/jq"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/k8s-manifest-kit/engine/pkg/filter"
	"github.com/k8s-manifest-kit/engine/pkg/types"
)

var (
	// ErrJqMustReturnBoolean is returned when a JQ expression doesn't return a boolean.
	ErrJqMustReturnBoolean = errors.New("jq expression must return a boolean")
)

// Filter creates a new JQ filter with the given expression and options.
func Filter(expression string, opts ...jq.Option) (types.Filter, error) {
	// Create a new JQ engine
	engine, err := jq.NewEngine(expression, opts...)
	if err != nil {
		return nil, fmt.Errorf("error creating jq engine: %w", err)
	}

	return func(_ context.Context, obj unstructured.Unstructured) (bool, error) {
		// Run the JQ program and get a single value
		v, err := engine.Run(obj.Object)
		if err != nil {
			return false, &filter.Error{
				Object: obj,
				Err:    fmt.Errorf("error executing jq expression: %w", err),
			}
		}

		// Convert the result to a boolean
		if b, ok := v.(bool); ok {
			return b, nil
		}

		return false, &filter.Error{
			Object: obj,
			Err:    fmt.Errorf("%w, got %T", ErrJqMustReturnBoolean, v),
		}
	}, nil
}
