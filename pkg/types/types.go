package types

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var (
	// ErrRendererNil is returned when a renderer is nil.
	ErrRendererNil = errors.New("renderer cannot be nil")

	// ErrRendererNameEmpty is returned when a renderer name is empty.
	ErrRendererNameEmpty = errors.New("renderer must return a non-empty name")
)

// Filter is a function type that processes a single unstructured.Unstructured object
// and returns true if the object should be kept, or false if it should be discarded.
type Filter func(ctx context.Context, object unstructured.Unstructured) (bool, error)

// Transformer is a function type that processes a single unstructured.Unstructured object
// and returns the transformed object.
type Transformer func(ctx context.Context, object unstructured.Unstructured) (unstructured.Unstructured, error)

// Renderer is a non-generic interface that concrete renderer types implement.
// This allows the Engine to manage them heterogeneously.
type Renderer interface {
	// Process executes the rendering logic for all configured inputs of this renderer.
	// The values parameter contains render-time values from engine.Render(ctx, engine.WithValues(...)).
	// Renderers that support dynamic values (Helm, Kustomize, GoTemplate) should deep merge
	// these values with Source-level values, with render-time values taking precedence.
	Process(ctx context.Context, values map[string]any) ([]unstructured.Unstructured, error)

	// Name returns the renderer type identifier for metrics and logging.
	// Examples: "helm", "kustomize", "gotemplate", "yaml", "mem"
	Name() string
}

// ValidateRenderer checks if a Renderer implementation is valid.
// Returns an error if the renderer is nil or if Name() returns an empty string.
func ValidateRenderer(r Renderer) error {
	if r == nil {
		return ErrRendererNil
	}

	name := r.Name()
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("%w: %T", ErrRendererNameEmpty, r)
	}

	return nil
}
