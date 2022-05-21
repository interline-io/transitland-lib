package copier

import (
	"fmt"
	"strings"

	"github.com/interline-io/transitland-lib/tl"
)

func stopPatternKey(stoptimes []tl.StopTime) string {
	key := make([]string, len(stoptimes))
	for i := 0; i < len(stoptimes); i++ {
		key[i] = stoptimes[i].StopID
	}
	return strings.Join(key, string(byte(0)))
}

func journeyPatternKey(trip *tl.Trip) string {
	stkey := make([]string, len(trip.StopTimes)+1)
	a := trip.StopTimes[0].ArrivalTime
	b := trip.StopTimes[0].DepartureTime
	stkey[0] = fmt.Sprintf(
		"%s-%s-%s-%s-%s-%d-%d-%d-%s",
		trip.RouteID,
		trip.ServiceID,
		trip.TripHeadsign,
		trip.TripShortName,
		trip.ShapeID.Key,
		trip.DirectionID,
		trip.WheelchairAccessible,
		trip.BikesAllowed,
		trip.BlockID,
	)
	for i := 0; i < len(trip.StopTimes); i++ {
		st := trip.StopTimes[i]
		stkey[i+1] = fmt.Sprintf(
			"%d-%d-%s-%s-%d-%d-%d",
			st.ArrivalTime.Seconds-a.Seconds,
			st.DepartureTime.Seconds-b.Seconds,
			st.StopID,
			st.StopHeadsign.String,
			st.PickupType.Int,
			st.DropOffType.Int,
			st.Timepoint.Int,
		)
	}
	return strings.Join(stkey, "|")
}
