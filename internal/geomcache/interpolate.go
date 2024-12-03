package geomcache

import (
	"github.com/interline-io/transitland-lib/gtfs"
)

// InterpolateStopTimes sets missing ArrivalTime, DestinationTime values.
// StopTimes must be sorted and valid.
func InterpolateStopTimes(stoptimes []gtfs.StopTime) ([]gtfs.StopTime, error) {
	// Look for gaps
	for start := 0; start < len(stoptimes)-1; {
		// find the next stoptime with arrivaltime
		end := start + 1
		for ; end < len(stoptimes)-1; end++ {
			if stoptimes[end].ArrivalTime.Int() > 0 {
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

func interpolateGap(stoptimes *[]gtfs.StopTime, start int, end int) {
	if start == end {
		return
	}
	sts := *stoptimes
	stStart := sts[start]
	stEnd := sts[end]
	t := float64((stEnd.ArrivalTime.Int() - stStart.DepartureTime.Int()))
	x := stEnd.ShapeDistTraveled.Val - stStart.ShapeDistTraveled.Val
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
		dx := (sts[i].ShapeDistTraveled.Val - stStart.ShapeDistTraveled.Val) / x
		dt := stStart.DepartureTime.Int() + int(t*dx)
		// log.Trace(
		// 	"\tindex: %d traveled: %f dx: %f dt: %d",
		// 	i, sts[i].ShapeDistTraveled, dx, dt,
		// )
		sts[i].ArrivalTime.SetInt(dt)
		sts[i].DepartureTime.SetInt(dt)
		sts[i].Interpolated.Set(1)
	}
}
