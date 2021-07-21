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
	stkey := make([]string, len(trip.StopTimes))
	a := trip.StopTimes[0].ArrivalTime
	b := trip.StopTimes[0].DepartureTime
	for i := 0; i < len(trip.StopTimes); i++ {
		st := trip.StopTimes[i]
		stkey[i] = fmt.Sprintf(
			"%d-%d-%s-%s-%d-%d-%d",
			st.ArrivalTime-a,
			st.DepartureTime-b,
			st.StopID,
			st.StopHeadsign.String,
			st.PickupType.Int,
			st.DropOffType.Int,
			st.Timepoint.Int,
		)
	}
	return strings.Join(stkey, "|")
}
