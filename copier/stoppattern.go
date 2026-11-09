package copier

import (
	"crypto/sha1"
	"fmt"
	"strings"

	"github.com/interline-io/transitland-lib/gtfs"
)

func stopPatternKey(stoptimes []gtfs.StopTime) string {
	// Skip flex trips - return empty key which won't match other patterns
	if !gtfs.CheckFlexStopTimes(stoptimes).AllStopsHaveStopID {
		return ""
	}
	key := make([]string, 0, len(stoptimes))
	for i := 0; i < len(stoptimes); i++ {
		key = append(key, stoptimes[i].StopID.Val)
	}
	return strings.Join(key, string(byte(0)))
}

// journeyPatternKey groups trips whose stop times match once each is offset by
// its first arrival: the same route, service and shape, and the same fields at
// every stop. A trip's other fields, its headsign or block say, stay on its own
// row, so they don't split patterns.
func journeyPatternKey(trip *gtfs.Trip) string {
	// Skip flex trips - return empty key to prevent deduplication. A stop time
	// with a pickup and drop-off window is flex too, even one from midnight to
	// midnight.
	if !gtfs.CheckFlexStopTimes(trip.StopTimes).AllStopsHaveStopID {
		return ""
	}
	for i := range trip.StopTimes {
		if trip.StopTimes[i].StartPickupDropOffWindow.Valid || trip.StopTimes[i].EndPickupDropOffWindow.Valid {
			return ""
		}
	}
	m := sha1.New()
	// Departures are keyed from the first arrival, as arrivals are, because the
	// copier offsets each trip by its first arrival.
	a := trip.StopTimes[0].ArrivalTime
	m.Write([]byte(fmt.Sprintf(
		"%s-%s-%s",
		trip.RouteID.Val,
		trip.ServiceID.Val,
		trip.ShapeID.Val,
	)))
	for i := 0; i < len(trip.StopTimes); i++ {
		st := trip.StopTimes[i]
		stopID := ""
		if st.StopID.Valid {
			stopID = st.StopID.Val
		}
		// Unset is kept apart from 0: an unset timepoint means exact times and
		// an unset continuous pickup means none. A float prints its value even
		// when unset, so an unset distance is written as nothing.
		dist := ""
		if st.ShapeDistTraveled.Valid {
			dist = st.ShapeDistTraveled.String()
		}
		m.Write([]byte(fmt.Sprintf(
			"%d-%d-%s-%s-%s-%s-%s-%d-%s-%s-%s-%s",
			st.ArrivalTime.Val-a.Val,
			st.DepartureTime.Val-a.Val,
			stopID,
			st.StopHeadsign.Val,
			st.PickupType.String(),
			st.DropOffType.String(),
			st.Timepoint.String(),
			st.StopSequence.Val,
			dist,
			st.ContinuousPickup.String(),
			st.ContinuousDropOff.String(),
			st.Interpolated.String(),
		)))
	}
	return fmt.Sprintf("%x", m.Sum(nil))
}
