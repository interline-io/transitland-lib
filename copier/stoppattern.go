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
		"%s-%s-%s-%s-%s-%d-%d-%d-%d-%s",
		trip.RouteID.Val,
		trip.ServiceID.Val,
		trip.TripHeadsign.Val,
		trip.TripShortName.Val,
		trip.ShapeID.Val,
		trip.DirectionID.Val,
		trip.WheelchairAccessible.Val,
		trip.BikesAllowed.Val,
		trip.CarsAllowed.Val,
		trip.BlockID.Val,
	)))
	for i := 0; i < len(trip.StopTimes); i++ {
		st := trip.StopTimes[i]
		stopID := ""
		if st.StopID.Valid {
			stopID = st.StopID.Val
		}
		m.Write([]byte(fmt.Sprintf(
			"%d-%d-%s-%s-%d-%d-%d-%d",
			st.ArrivalTime.Val-a.Val,
			st.DepartureTime.Val-a.Val,
			stopID,
			st.StopHeadsign.Val,
			st.PickupType.Val,
			st.DropOffType.Val,
			st.Timepoint.Val,
			st.StopSequence.Val,
		)))
	}
	return fmt.Sprintf("%x", m.Sum(nil))
}
