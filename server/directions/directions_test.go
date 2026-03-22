package directions

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/server/model"
	"github.com/stretchr/testify/assert"
)

type mockHandler struct {
	caps Capabilities
}

func (h *mockHandler) Capabilities() Capabilities { return h.caps }
func (h *mockHandler) Request(_ context.Context, _ model.DirectionRequest) (*model.Directions, error) {
	return &model.Directions{Success: true}, nil
}

func init() {
	RegisterRouter("test-no-arrive-by", func() Handler {
		return &mockHandler{caps: Capabilities{SupportsArriveBy: false}}
	})
	RegisterRouter("test-arrive-by", func() Handler {
		return &mockHandler{caps: Capabilities{SupportsArriveBy: true}}
	})
}

func ptrBool(v bool) *bool { return &v }

func baseReq() model.DirectionRequest {
	// Use BICYCLE mode so HandleRequest doesn't override pref via env vars.
	return model.DirectionRequest{
		Mode: model.StepModeBicycle,
		From: &model.WaypointInput{Lat: 37.78, Lon: -122.40},
		To:   &model.WaypointInput{Lat: 37.79, Lon: -122.41},
	}
}

func TestArriveByCapabilityValidation(t *testing.T) {
	t.Run("rejected when unsupported", func(t *testing.T) {
		req := baseReq()
		req.ArriveBy = ptrBool(true)
		res, err := HandleRequest(context.Background(), "test-no-arrive-by", req)
		assert.NoError(t, err)
		assert.False(t, res.Success)
		assert.Contains(t, *res.Exception, "arrive_by is not supported")
	})

	t.Run("allowed when supported", func(t *testing.T) {
		req := baseReq()
		req.ArriveBy = ptrBool(true)
		res, err := HandleRequest(context.Background(), "test-arrive-by", req)
		assert.NoError(t, err)
		assert.True(t, res.Success)
	})

	t.Run("false arrive_by always passes through", func(t *testing.T) {
		req := baseReq()
		req.ArriveBy = ptrBool(false)
		res, err := HandleRequest(context.Background(), "test-no-arrive-by", req)
		assert.NoError(t, err)
		assert.True(t, res.Success)
	})
}
