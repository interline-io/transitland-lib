package directionstest

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/finders/directions"
	"github.com/interline-io/transitland-lib/model"
	"github.com/stretchr/testify/assert"
)

var BaseFrom = model.WaypointInput{Lon: -122.401001, Lat: 37.789001}
var BaseTo = model.WaypointInput{Lon: -122.446999, Lat: 37.782001}
var BaseTime = time.Unix(1234567890, 0)

func MakeBasicTests() map[string]model.DirectionRequest {
	m := map[string]model.DirectionRequest{
		"ped": {
			Mode:     model.StepModeWalk,
			From:     &BaseFrom,
			To:       &BaseTo,
			DepartAt: &BaseTime,
		},
		"bike": {
			Mode:     model.StepModeBicycle,
			From:     &BaseFrom,
			To:       &BaseTo,
			DepartAt: &BaseTime,
		},
		"auto": {
			Mode:     model.StepModeAuto,
			From:     &BaseFrom,
			To:       &BaseTo,
			DepartAt: &BaseTime,
		},
		"transit": {
			Mode:     model.StepModeTransit,
			From:     &BaseFrom,
			To:       &BaseTo,
			DepartAt: &BaseTime,
		},
		"no_dest_fail": {
			Mode:     model.StepModeWalk,
			From:     &BaseFrom,
			DepartAt: &BaseTime,
		},
		"no_routable_dest_fail": {
			Mode:     model.StepModeWalk,
			From:     &BaseFrom,
			To:       &model.WaypointInput{Lon: -123.54949951171876, Lat: 37.703380457832374},
			DepartAt: &BaseTime,
		},
	}
	return m
}

type TestCase struct {
	Name     string
	Req      model.DirectionRequest
	Success  bool
	Duration float64
	Distance float64
	ResJson  string
}

func HandlerTest(t *testing.T, h directions.Handler, tc TestCase) *model.Directions {
	ret, err := h.Request(context.Background(), tc.Req)
	if err != nil {
		t.Fatal(err)
	}
	if ret.Success != tc.Success {
		t.Errorf("got success '%t', expected '%t'", ret.Success, tc.Success)
	} else if ret.Success {
		assert.InDelta(t, ret.Duration.Duration, tc.Duration, 1.0, "duration")
		assert.InDelta(t, ret.Distance.Distance, tc.Distance, 1.0, "distance")
	}
	_ = time.Now()
	resJson, err := json.MarshalIndent(ret, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	// t.Log("response:", string(resJson))

	if tc.ResJson != "" {
		a, err := os.ReadFile(tc.ResJson)
		if err != nil {
			t.Fatal(err)
		}
		if !assert.JSONEq(t, string(a), string(resJson)) {
			t.Log("expected json file:", tc.ResJson)
			t.Log("expected json was:")
			t.Log(string(a))
			t.Log("json response was:")
			t.Log(string(resJson))
		}
	}
	return ret
}
