package copier

import (
	"fmt"
	"testing"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// keyTrip is a trip on one route and service through a stop for each pair of
// arrival and departure seconds.
func keyTrip(times ...[2]int) *gtfs.Trip {
	trip := &gtfs.Trip{RouteID: tt.NewKey("R"), ServiceID: tt.NewKey("S")}
	for i, t := range times {
		trip.StopTimes = append(trip.StopTimes, gtfs.StopTime{
			StopID:        tt.NewKey(fmt.Sprintf("stop%d", i)),
			StopSequence:  tt.NewInt(i + 1),
			ArrivalTime:   tt.NewSeconds(t[0]),
			DepartureTime: tt.NewSeconds(t[1]),
		})
	}
	return trip
}

func TestJourneyPatternKey(t *testing.T) {
	// The same stops and times an hour apart share a pattern.
	base := keyTrip([2]int{0, 0}, [2]int{600, 600})
	require.NotEmpty(t, journeyPatternKey(base))
	require.Equal(t, journeyPatternKey(base), journeyPatternKey(keyTrip([2]int{3600, 3600}, [2]int{4200, 4200})))

	t.Run("departures keyed from the first arrival", func(t *testing.T) {
		// Two minutes more dwell at every stop. The copier offsets a trip by its
		// first arrival, so sharing a pattern would give it the other's departures.
		dwell := keyTrip([2]int{3600, 3720}, [2]int{4200, 4320})
		assert.NotEqual(t, journeyPatternKey(base), journeyPatternKey(dwell))
	})
	t.Run("a pickup window makes a pattern of its own", func(t *testing.T) {
		// A window from midnight to midnight is still a window.
		win := keyTrip([2]int{0, 0}, [2]int{600, 600})
		win.StopTimes[1].StartPickupDropOffWindow = tt.NewSeconds(0)
		win.StopTimes[1].EndPickupDropOffWindow = tt.NewSeconds(0)
		assert.Empty(t, journeyPatternKey(win))
	})
	t.Run("stop sequences are part of the key", func(t *testing.T) {
		renumbered := keyTrip([2]int{0, 0}, [2]int{600, 600})
		renumbered.StopTimes[1].StopSequence = tt.NewInt(5)
		assert.NotEqual(t, journeyPatternKey(base), journeyPatternKey(renumbered))
	})
	t.Run("a trip's own fields don't split patterns", func(t *testing.T) {
		own := keyTrip([2]int{0, 0}, [2]int{600, 600})
		own.TripHeadsign = tt.NewString("Downtown")
		own.TripShortName = tt.NewString("101")
		own.DirectionID = tt.NewInt(1)
		own.BlockID = tt.NewString("b7")
		own.WheelchairAccessible = tt.NewInt(1)
		own.BikesAllowed = tt.NewInt(2)
		own.CarsAllowed = tt.NewInt(1)
		assert.Equal(t, journeyPatternKey(base), journeyPatternKey(own))
	})
	t.Run("every stored stop time field splits patterns", func(t *testing.T) {
		for name, set := range map[string]func(*gtfs.StopTime){
			"shape_dist_traveled": func(st *gtfs.StopTime) { st.ShapeDistTraveled = tt.NewFloat(0) },
			"continuous_pickup":   func(st *gtfs.StopTime) { st.ContinuousPickup = tt.NewInt(0) },
			"continuous_drop_off": func(st *gtfs.StopTime) { st.ContinuousDropOff = tt.NewInt(0) },
			"timepoint":           func(st *gtfs.StopTime) { st.Timepoint = tt.NewInt(0) },
			"interpolated":        func(st *gtfs.StopTime) { st.Interpolated = tt.NewInt(1) },
		} {
			// Set to 0 where 0 differs from unset, as for these fields.
			other := keyTrip([2]int{0, 0}, [2]int{600, 600})
			set(&other.StopTimes[1])
			assert.NotEqual(t, journeyPatternKey(base), journeyPatternKey(other), name)
		}
	})
}

func Benchmark_stopPatternKey(b *testing.B) {
	stoptimes := []gtfs.StopTime{}
	for i := 0; i < 50; i++ {
		stoptimes = append(stoptimes, gtfs.StopTime{StopID: tt.NewKey(fmt.Sprintf("%d", i*100))})
	}
	m := map[string]int{}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		key := stopPatternKey(stoptimes)
		m[key]++
	}
}
