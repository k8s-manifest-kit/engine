package engine_test

import (
	"testing"

	"github.com/stretchr/testify/mock"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	engine "github.com/k8s-manifest-kit/engine/pkg"
	"github.com/k8s-manifest-kit/engine/pkg/types"

	. "github.com/onsi/gomega"
)

func TestValidateRenderer(t *testing.T) {

	t.Run("should accept valid renderer", func(t *testing.T) {
		g := NewWithT(t)
		renderer := new(mockRenderer)
		renderer.On("Process", mock.Anything, mock.Anything).Return([]unstructured.Unstructured{makePod("pod1")}, nil)
		renderer.On("Name").Return("mock")

		err := types.ValidateRenderer(renderer)
		g.Expect(err).ShouldNot(HaveOccurred())
	})

	t.Run("should reject nil renderer", func(t *testing.T) {
		g := NewWithT(t)
		err := types.ValidateRenderer(nil)
		g.Expect(err).Should(HaveOccurred())
		g.Expect(err.Error()).Should(ContainSubstring("renderer cannot be nil"))
	})

	t.Run("should reject renderer with empty name", func(t *testing.T) {
		g := NewWithT(t)
		renderer := new(mockRenderer)
		renderer.On("Process", mock.Anything, mock.Anything).Return([]unstructured.Unstructured{makePod("pod1")}, nil)
		renderer.On("Name").Return("")

		err := types.ValidateRenderer(renderer)
		g.Expect(err).Should(HaveOccurred())
		g.Expect(err.Error()).Should(ContainSubstring("must return a non-empty name"))
	})

	t.Run("should reject renderer with whitespace-only name", func(t *testing.T) {
		g := NewWithT(t)
		renderer := new(mockRenderer)
		renderer.On("Process", mock.Anything, mock.Anything).Return([]unstructured.Unstructured{makePod("pod1")}, nil)
		renderer.On("Name").Return("   \t\n  ")

		err := types.ValidateRenderer(renderer)
		g.Expect(err).Should(HaveOccurred())
		g.Expect(err.Error()).Should(ContainSubstring("must return a non-empty name"))
	})
}

func TestNewValidatesRenderers(t *testing.T) {

	t.Run("should reject engine creation with nil renderer", func(t *testing.T) {
		g := NewWithT(t)
		e, err := engine.New(engine.WithRenderer(nil))
		g.Expect(err).Should(HaveOccurred())
		g.Expect(err.Error()).Should(ContainSubstring("invalid renderer"))
		g.Expect(err.Error()).Should(ContainSubstring("renderer cannot be nil"))
		g.Expect(e).Should(BeNil())
	})

	t.Run("should reject engine creation with renderer with empty name", func(t *testing.T) {
		g := NewWithT(t)
		renderer := new(mockRenderer)
		renderer.On("Process", mock.Anything, mock.Anything).Return([]unstructured.Unstructured{makePod("pod1")}, nil)
		renderer.On("Name").Return("")

		e, err := engine.New(engine.WithRenderer(renderer))
		g.Expect(err).Should(HaveOccurred())
		g.Expect(err.Error()).Should(ContainSubstring("invalid renderer"))
		g.Expect(err.Error()).Should(ContainSubstring("must return a non-empty name"))
		g.Expect(e).Should(BeNil())
	})

	t.Run("should accept engine creation with valid renderer", func(t *testing.T) {
		g := NewWithT(t)
		renderer := new(mockRenderer)
		renderer.On("Process", mock.Anything, mock.Anything).Return([]unstructured.Unstructured{makePod("pod1")}, nil)
		renderer.On("Name").Return("mock")

		e, err := engine.New(engine.WithRenderer(renderer))
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(e).ShouldNot(BeNil())
	})
}
