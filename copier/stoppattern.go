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
	stkey := make([]string, len(stoptimes))
	a := stoptimes[0].ArrivalTime
	b := stoptimes[0].DepartureTime
	for i := range stoptimes {
		st := stoptimes[i]
		stkey[i] = fmt.Sprintf(
			"%d-%d-%s-%s-%d-%d-%d-%0.2f",
			st.ArrivalTime-a,
			st.DepartureTime-b,
			st.StopID,
			st.StopHeadsign.String,
			st.PickupType.Int32,
			st.DropOffType.Int32,
			st.Timepoint.Int32,
			st.ShapeDistTraveled.Float64,
		)
	}
	return strings.Join(stkey, "-")
}
