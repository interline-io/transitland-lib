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

func journeyPatternKey(trip tl.Trip, stoptimes []tl.StopTime) string {
	tripkey := fmt.Sprintf(
		"%s-%s-%s-%s-%d-%s-%s-%d-%d",
		trip.RouteID,
		trip.ServiceID,
		trip.TripHeadsign,
		trip.TripShortName,
		trip.DirectionID,
		trip.BlockID,
		trip.ShapeID.Key,
		trip.WheelchairAccessible,
		trip.BikesAllowed,
	)
	if len(stoptimes) == 0 {
		return tripkey
	}
	stkey := make([]string, len(stoptimes))
	a := stoptimes[0].ArrivalTime
	b := stoptimes[0].DepartureTime
	for i := range stoptimes {
		st := stoptimes[i]
		stkey[i] = fmt.Sprintf(
			"%d-%d-%s-%s-%d-%d-%d",
			st.ArrivalTime-a,
			st.DepartureTime-b,
			st.StopID,
			st.StopHeadsign,
			st.PickupType,
			st.DropOffType,
			st.Timepoint,
		)
	}
	return tripkey + strings.Join(stkey, "-")
}
