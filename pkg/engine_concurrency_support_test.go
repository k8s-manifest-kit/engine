package engine_test

import (
	"context"
	"fmt"

	"github.com/k8s-manifest-kit/pkg/util/k8s"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/k8s-manifest-kit/engine/pkg/types"

	. "github.com/onsi/gomega"
)

// concurrentRequests is the number of overlapping Render calls issued against
// one shared engine during the concurrent phase.
const concurrentRequests = 8

// expectRequestResult asserts that one concurrent request returned exactly
// its own objects, hook results, and values, untouched by any other request
// or renderer.
func expectRequestResult(
	g *WithT,
	id int,
	objects []unstructured.Unstructured,
	renderErr error,
	input types.Values,
	alpha *recordingRenderer,
	beta *recordingRenderer,
) {
	label := requestLabel(id)

	g.Expect(renderErr).ToNot(HaveOccurred())
	g.Expect(objects).To(HaveLen(2))
	g.Expect(objectNames(objects)).To(ConsistOf(
		fmt.Sprintf("pod-request-%d-alpha", id),
		fmt.Sprintf("pod-request-%d-beta", id),
	))

	for _, object := range objects {
		g.Expect(object.GetLabels()).To(Equal(map[string]string{"request": label}))
		g.Expect(object.GetAnnotations()).To(Equal(map[string]string{"request": label}))
	}

	// Both renderers received exactly this request's values. alpha mutated
	// its own nested copy; beta's copy and the caller's input stayed pristine.
	g.Expect(alpha.received[id]).To(Equal(mutatedRequestValues(id, alpha.name)))
	g.Expect(beta.received[id]).To(Equal(requestValues(id)))
	g.Expect(input).To(Equal(requestValues(id)))
}

// recordingRenderer is a concurrency-safe types.Renderer for concurrency
// tests. It records the values received per request slot, mutates its own
// nested copy of them when enabled, and returns freshly created objects
// derived from the request.
type recordingRenderer struct {
	name string
	// received stores the values of the last Process call per request id.
	// The final slot is reserved for ordinary renders that pass no values.
	// Concurrent callers only ever write to their own slot, so no locking
	// is needed.
	received []types.Values
	// mutateNested controls whether Process mutates the nested request map
	// of the values it receives.
	mutateNested bool
}

// newRecordingRenderer creates a renderer recording up to concurrentRequests
// distinct requests plus one ordinary render, mutating its received values
// when mutateNested is set.
func newRecordingRenderer(name string, mutateNested bool) *recordingRenderer {
	return &recordingRenderer{
		name:         name,
		received:     make([]types.Values, concurrentRequests+1),
		mutateNested: mutateNested,
	}
}

// Name implements types.Renderer.
func (r *recordingRenderer) Name() string {
	return r.name
}

// Process implements types.Renderer. It records the received values, mutates
// its own nested copy when enabled, and returns fresh objects per call.
func (r *recordingRenderer) Process(_ context.Context, values types.Values) ([]unstructured.Unstructured, error) {
	request, hasRequest := values["request"].(map[string]any)

	r.received[requestSlot(values)] = values

	if r.mutateNested && hasRequest {
		request["mutated-by"] = r.name
	}

	return []unstructured.Unstructured{makePod(r.objectName(values))}, nil
}

// objectName returns the name of the object this renderer creates for the
// given values: request renders embed the request id, ordinary renders do not.
func (r *recordingRenderer) objectName(values types.Values) string {
	id, ok := requestID(values)
	if !ok {
		return "pod-" + r.name
	}

	return fmt.Sprintf("pod-request-%d-%s", id, r.name)
}

// requestSlot returns the recording slot for a Process call: the request id
// for request renders, or the reserved final slot for ordinary renders that
// pass no values.
func requestSlot(values types.Values) int {
	id, ok := requestID(values)
	if !ok {
		return concurrentRequests
	}

	return id
}

// requestID returns the numeric id carried by a request's values.
func requestID(values types.Values) (int, bool) {
	request, ok := values["request"].(map[string]any)
	if !ok {
		return 0, false
	}

	id, ok := request["id"].(int)

	return id, ok
}

// requestValues returns the render-time values for the request with the given
// id. Each call returns a fresh, deeply independent map.
func requestValues(id int) types.Values {
	return types.Values{
		"request": map[string]any{
			"id":   id,
			"name": requestLabel(id),
			"labels": map[string]any{
				"tier": "web",
			},
		},
	}
}

// mutatedRequestValues returns the values a renderer is expected to record
// after mutating its own nested copy of the request values.
func mutatedRequestValues(id int, rendererName string) types.Values {
	values := requestValues(id)
	values["request"].(map[string]any)["mutated-by"] = rendererName

	return values
}

// requestLabel returns the unique label identifying the request with the
// given id.
func requestLabel(id int) string {
	return fmt.Sprintf("request-%d", id)
}

// tagObjects returns a render-time post-renderer that annotates every object
// with the request label.
func tagObjects(label string) types.PostRenderer {
	return func(_ context.Context, objects []unstructured.Unstructured) ([]unstructured.Unstructured, error) {
		for i := range objects {
			k8s.SetAnnotation(&objects[i], "request", label)
		}

		return objects, nil
	}
}

// objectNames returns the names of the given objects.
func objectNames(objects []unstructured.Unstructured) []string {
	names := make([]string, 0, len(objects))

	for _, object := range objects {
		names = append(names, object.GetName())
	}

	return names
}
