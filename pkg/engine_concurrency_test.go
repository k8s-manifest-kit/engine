package engine_test

import (
	"sync"
	"testing"

	"github.com/k8s-manifest-kit/pkg/util/k8s"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	engine "github.com/k8s-manifest-kit/engine/pkg"
	"github.com/k8s-manifest-kit/engine/pkg/render"
	"github.com/k8s-manifest-kit/engine/pkg/types"

	. "github.com/onsi/gomega"
)

// TestRenderConcurrentRequests verifies that overlapping Render calls on a
// single engine keep render-time values, hooks, and outputs isolated per
// request under the race detector.
func TestRenderConcurrentRequests(t *testing.T) {
	g := NewWithT(t)

	alpha := newRecordingRenderer("alpha", true)
	beta := newRecordingRenderer("beta", false)

	e, err := engine.New(
		engine.WithRenderer(alpha),
		engine.WithRenderer(beta),
	)
	g.Expect(err).ToNot(HaveOccurred())

	// Successful cold initialization before concurrent use.
	coldObjects, err := e.Render(t.Context())
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(objectNames(coldObjects)).To(ConsistOf("pod-alpha", "pod-beta"))

	// Warm concurrent phase: every worker renders with its own values and
	// additive render-time hooks once the start barrier opens.
	ctx := t.Context()

	start := make(chan struct{})
	var done sync.WaitGroup

	inputs := make([]types.Values, concurrentRequests)
	results := make([][]unstructured.Unstructured, concurrentRequests)
	renderErrs := make([]error, concurrentRequests)

	for id := range concurrentRequests {
		done.Add(1)

		go func() {
			defer done.Done()
			<-start

			values := requestValues(id)
			inputs[id] = values

			objects, err := e.Render(ctx,
				render.WithValues(values),
				render.WithTransformer(addLabels(map[string]string{"request": requestLabel(id)})),
				render.WithPostRenderer(tagObjects(requestLabel(id))),
			)

			results[id] = objects
			renderErrs[id] = err
		}()
	}

	close(start)
	done.Wait()

	for id := range concurrentRequests {
		expectRequestResult(g, id, results[id], renderErrs[id], inputs[id], alpha, beta)
	}

	// Outputs are independently created per render: mutating one request's
	// result affects neither another request's result...
	k8s.SetLabel(&results[0][0], "mutated", "true")
	g.Expect(results[1][0].GetLabels()).ToNot(HaveKey("mutated"))

	// ...nor a later ordinary render, which also sees no concurrent hooks.
	finalObjects, err := e.Render(t.Context())
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(objectNames(finalObjects)).To(ConsistOf("pod-alpha", "pod-beta"))

	for _, object := range finalObjects {
		g.Expect(object.GetLabels()).To(BeEmpty())
		g.Expect(object.GetAnnotations()).To(BeEmpty())
	}

	// The ordinary render received empty values: no concurrent request's
	// values leaked into it.
	g.Expect(alpha.received[concurrentRequests]).To(BeEmpty())
	g.Expect(beta.received[concurrentRequests]).To(BeEmpty())
}
