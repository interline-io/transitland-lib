package copier

import (
	"testing"

	"github.com/interline-io/transitland-lib/tl"
)

func testExpectInt(t *testing.T, result, expect int) {
	if expect != result {
		t.Errorf("got %d, expect %d", result, expect)
	}
}

type expectStopTime struct {
	ArrivalTime         int
	DepartureTime       int
	ShapeDistTraveled   float64
	ExpectArrivalTime   int
	ExpectDepartureTime int
}

func expectTripToStopTime(e []expectStopTime) []tl.StopTime {
	ret := []tl.StopTime{}
	for _, i := range e {
		ret = append(ret, tl.StopTime{
			ArrivalTime:       i.ArrivalTime,
			DepartureTime:     i.DepartureTime,
			ShapeDistTraveled: i.ShapeDistTraveled,
		})
	}
	return ret
}

func TestInterpolateStopTimes(t *testing.T) {
	expectTrips := [][]expectStopTime{
		// one gap
		[]expectStopTime{
			{0, 20, 0.0, 0, 0},
			{0, 0, 10.0, 60, 60},
			{100, 120, 20.0, 0, 0},
		},
		// two gaps
		[]expectStopTime{
			{0, 10, 0.0, 0, 0},
			{0, 0, 10.0, 12, 12},
			{20, 40, 50.0, 0, 0},
			{0, 0, 60.0, 52, 52},
			{64, 64, 70.0, 0, 0},
		},
		// one gap, three stops
		[]expectStopTime{
			{10, 10, 10.0, 0, 0},
			{0, 0, 20.0, 20, 20},
			{0, 0, 30.0, 30, 30},
			{0, 0, 40.0, 40, 40},
			{50, 50, 50.0, 0, 0},
		},
	}
	for _, e := range expectTrips {
		stoptimes := expectTripToStopTime(e)
		stoptimes2, err := InterpolateStopTimes(stoptimes)
		if err != nil {
			t.Error(err)
		}
		for j, st := range stoptimes2 {
			if e[j].ExpectArrivalTime > 0 {
				testExpectInt(t, st.ArrivalTime, e[j].ExpectArrivalTime)
			}
			if e[j].ExpectDepartureTime > 0 {
				testExpectInt(t, st.DepartureTime, e[j].ExpectDepartureTime)
			}
		}
	}
}
