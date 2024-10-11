package copier

import (
	"crypto/sha1"
	"fmt"
	"strings"

	"github.com/interline-io/transitland-lib/gtfs"
)

func stopPatternKey(stoptimes []gtfs.StopTime) string {
	key := make([]string, len(stoptimes))
	for i := 0; i < len(stoptimes); i++ {
		key[i] = stoptimes[i].StopID
	}
	return strings.Join(key, string(byte(0)))
}

func journeyPatternKey(trip *gtfs.Trip) string {
	m := sha1.New()
	a := trip.StopTimes[0].ArrivalTime
	b := trip.StopTimes[0].DepartureTime
	m.Write([]byte(fmt.Sprintf(
		"%s-%s-%s-%s-%s-%d-%d-%d-%s",
		trip.RouteID,
		trip.ServiceID,
		trip.TripHeadsign,
		trip.TripShortName,
		trip.ShapeID.Val,
		trip.DirectionID,
		trip.WheelchairAccessible,
		trip.BikesAllowed,
		trip.BlockID,
	)))
	for i := 0; i < len(trip.StopTimes); i++ {
		st := trip.StopTimes[i]
		m.Write([]byte(fmt.Sprintf(
			"%d-%d-%s-%s-%d-%d-%d",
			st.ArrivalTime.Val-a.Val,
			st.DepartureTime.Val-b.Val,
			st.StopID,
			st.StopHeadsign.Val,
			st.PickupType.Val,
			st.DropOffType.Val,
			st.Timepoint.Val,
		)))
	}
	return fmt.Sprintf("%x", m.Sum(nil))
}
