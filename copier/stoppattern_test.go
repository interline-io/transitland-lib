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
