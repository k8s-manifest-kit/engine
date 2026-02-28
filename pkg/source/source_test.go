package source_test

import (
	"context"
	"errors"
	"testing"

	"github.com/k8s-manifest-kit/engine/pkg/source"

	. "github.com/onsi/gomega"
)

type helmSource struct {
	Chart       string
	ReleaseName string
}

type yamlSource struct {
	Path string
}

func TestSelector(t *testing.T) {
	ctx := t.Context()

	t.Run("should call function for matching concrete type", func(t *testing.T) {
		g := NewWithT(t)

		sel := source.Selector[helmSource](func(_ context.Context, s helmSource) (bool, error) {
			return s.ReleaseName == "my-app", nil
		})

		ok, err := sel(ctx, helmSource{Chart: "nginx", ReleaseName: "my-app"})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(ok).To(BeTrue())
	})

	t.Run("should reject when function returns false", func(t *testing.T) {
		g := NewWithT(t)

		sel := source.Selector[helmSource](func(_ context.Context, s helmSource) (bool, error) {
			return s.ReleaseName == "wanted", nil
		})

		ok, err := sel(ctx, helmSource{Chart: "nginx", ReleaseName: "unwanted"})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(ok).To(BeFalse())
	})

	t.Run("should pass through non-matching types", func(t *testing.T) {
		g := NewWithT(t)

		sel := source.Selector[helmSource](func(_ context.Context, _ helmSource) (bool, error) {
			return false, errors.New("should not be called")
		})

		ok, err := sel(ctx, yamlSource{Path: "/manifests"})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(ok).To(BeTrue())
	})

	t.Run("should propagate error from selector function", func(t *testing.T) {
		g := NewWithT(t)

		sel := source.Selector[helmSource](func(_ context.Context, _ helmSource) (bool, error) {
			return false, errors.New("validation failed")
		})

		ok, err := sel(ctx, helmSource{Chart: "nginx", ReleaseName: "my-app"})
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("validation failed"))
		g.Expect(ok).To(BeFalse())
	})

	t.Run("should match interface types", func(t *testing.T) {
		g := NewWithT(t)

		type named interface{ GetName() string }

		sel := source.Selector[named](func(_ context.Context, s named) (bool, error) {
			return s.GetName() == "match", nil
		})

		ok, err := sel(ctx, &namedSource{name: "match"})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(ok).To(BeTrue())

		ok, err = sel(ctx, &namedSource{name: "no-match"})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(ok).To(BeFalse())
	})

	t.Run("should pass through string source for struct selector", func(t *testing.T) {
		g := NewWithT(t)

		sel := source.Selector[helmSource](func(_ context.Context, _ helmSource) (bool, error) {
			return false, errors.New("should not be called")
		})

		ok, err := sel(ctx, "just-a-string")
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(ok).To(BeTrue())
	})
}

type namedSource struct{ name string }

func (n *namedSource) GetName() string { return n.name }
