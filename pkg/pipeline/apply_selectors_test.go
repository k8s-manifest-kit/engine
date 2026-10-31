package pipeline_test

import (
	"context"
	"errors"
	"testing"

	"github.com/k8s-manifest-kit/engine/pkg/pipeline"

	. "github.com/onsi/gomega"
)

type testSource struct{ Name string }

type testSelector = func(context.Context, testSource) (bool, error)

func TestApplySourceSelectors(t *testing.T) {
	ctx := t.Context()

	t.Run("should return true when no selectors", func(t *testing.T) {
		g := NewWithT(t)

		ok, err := pipeline.ApplySourceSelectors(ctx, testSource{Name: "any"}, nil)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(ok).To(BeTrue())
	})

	t.Run("should return true when selector accepts", func(t *testing.T) {
		g := NewWithT(t)

		accept := func(_ context.Context, _ testSource) (bool, error) {
			return true, nil
		}

		ok, err := pipeline.ApplySourceSelectors(ctx, testSource{}, []testSelector{accept})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(ok).To(BeTrue())
	})

	t.Run("should return false when selector rejects", func(t *testing.T) {
		g := NewWithT(t)

		reject := func(_ context.Context, _ testSource) (bool, error) {
			return false, nil
		}

		ok, err := pipeline.ApplySourceSelectors(ctx, testSource{}, []testSelector{reject})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(ok).To(BeFalse())
	})

	t.Run("should require all selectors to accept (AND logic)", func(t *testing.T) {
		g := NewWithT(t)

		accept := func(_ context.Context, _ testSource) (bool, error) {
			return true, nil
		}

		reject := func(_ context.Context, _ testSource) (bool, error) {
			return false, nil
		}

		ok, err := pipeline.ApplySourceSelectors(ctx, testSource{}, []testSelector{accept, reject})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(ok).To(BeFalse())
	})

	t.Run("should short-circuit on first rejection", func(t *testing.T) {
		g := NewWithT(t)

		secondCalled := false

		reject := func(_ context.Context, _ testSource) (bool, error) {
			return false, nil
		}

		second := func(_ context.Context, _ testSource) (bool, error) {
			secondCalled = true

			return true, nil
		}

		ok, err := pipeline.ApplySourceSelectors(ctx, testSource{}, []testSelector{reject, second})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(ok).To(BeFalse())
		g.Expect(secondCalled).To(BeFalse())
	})

	t.Run("should return error from failing selector", func(t *testing.T) {
		g := NewWithT(t)

		failing := func(_ context.Context, _ testSource) (bool, error) {
			return false, errors.New("selector error")
		}

		ok, err := pipeline.ApplySourceSelectors(ctx, testSource{}, []testSelector{failing})
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("selector error"))
		g.Expect(ok).To(BeFalse())
	})

	t.Run("should pass source to selector", func(t *testing.T) {
		g := NewWithT(t)

		src := testSource{Name: "my-source"}

		selector := func(_ context.Context, s testSource) (bool, error) {
			g.Expect(s.Name).To(Equal("my-source"))

			return true, nil
		}

		ok, err := pipeline.ApplySourceSelectors(ctx, src, []testSelector{selector})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(ok).To(BeTrue())
	})
}
