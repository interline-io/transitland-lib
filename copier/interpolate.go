package copier

import (
	"database/sql"

	"github.com/interline-io/transitland-lib/tl"
)

// InterpolateStopTimes sets missing ArrivalTime, DestinationTime values.
// StopTimes must be sorted and valid.
func InterpolateStopTimes(stoptimes []tl.StopTime) ([]tl.StopTime, error) {
	// Look for gaps
	for start := 0; start < len(stoptimes)-1; {
		// find the next stoptime with arrivaltime
		end := start + 1
		for ; end < len(stoptimes)-1; end++ {
			if stoptimes[end].ArrivalTime > 0 {
				break
			}
		}
		if end-start > 1 {
			interpolateGap(&stoptimes, start, end)
		}
		start = end
	}
	return stoptimes, nil
}

func interpolateGap(stoptimes *[]tl.StopTime, start int, end int) {
	if start == end {
		return
	}
	sts := *stoptimes
	stStart := sts[start]
	stEnd := sts[end]
	t := float64(stEnd.ArrivalTime - stStart.DepartureTime)
	x := stEnd.ShapeDistTraveled.Float64 - stStart.ShapeDistTraveled.Float64
	// For StopTimes *between* start and end
	// log.Trace(
	// 	"trip '%s' interpolating %d stoptimes: index %d -> %d time: %d .. %d = %f distance: %f .. %f = %f",
	// 	sts[0].TripID,
	// 	end-start-1,
	// 	start, end,
	// 	stStart.DepartureTime, stEnd.ArrivalTime, t,
	// 	stStart.ShapeDistTraveled, stEnd.ShapeDistTraveled, x,
	// )
	for i := start + 1; i < end; i++ {
		dx := (sts[i].ShapeDistTraveled.Float64 - stStart.ShapeDistTraveled.Float64) / x
		dt := stStart.DepartureTime + int(t*dx)
		// log.Trace(
		// 	"\tindex: %d traveled: %f dx: %f dt: %d",
		// 	i, sts[i].ShapeDistTraveled, dx, dt,
		// )
		sts[i].ArrivalTime = dt
		sts[i].DepartureTime = dt
		sts[i].Interpolated = sql.NullInt32{Int32: 1, Valid: true}
	}
}
