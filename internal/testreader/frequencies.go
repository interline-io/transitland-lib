package testreader

import (
	"time"

	"github.com/interline-io/transitland-lib/adapters/direct"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

// NewFrequencyTestFeed returns a feed of four trips that cover how frequencies expand.
//
// The trips run every day from 2024-01-01 to 2024-01-07, and each takes 10 minutes
// from stop1 to stop2, so all four share one journey pattern:
//   - regular: no frequencies; runs at 05:00:00
//   - offset: a template at 06:00:00, every 30 minutes from 07:00:00 to 08:00:00; runs at 07:00:00 and 07:30:00
//   - meet: two periods that meet at 11:00:00, and an empty one at 12:30:00; runs at 10:00:00, 10:30:00, 11:00:00 and 11:30:00
//   - empty: one empty period, at 13:00:00; never runs
func NewFrequencyTestFeed() *direct.Reader {
	r := &direct.Reader{
		AgencyList: []gtfs.Agency{
			{AgencyID: tt.NewString("agency1"), AgencyName: tt.NewString("Agency 1"), AgencyTimezone: tt.NewTimezone("America/Los_Angeles"), AgencyURL: tt.NewUrl("http://example.com")},
		},
		RouteList: []gtfs.Route{
			{RouteID: tt.NewString("route1"), RouteShortName: tt.NewString("Route 1"), RouteType: tt.NewInt(3), AgencyID: tt.NewKey("agency1")},
		},
		StopList: []gtfs.Stop{
			{StopID: tt.NewString("stop1"), StopName: tt.NewString("Stop 1"), Geometry: tt.NewPoint(-122.0, 37.0)},
			{StopID: tt.NewString("stop2"), StopName: tt.NewString("Stop 2"), Geometry: tt.NewPoint(-122.0, 37.1)},
		},
		CalendarList: []gtfs.Calendar{
			{
				ServiceID: tt.NewString("service1"),
				StartDate: tt.NewDate(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
				EndDate:   tt.NewDate(time.Date(2024, 1, 7, 0, 0, 0, 0, time.UTC)),
				Monday:    tt.NewInt(1), Tuesday: tt.NewInt(1), Wednesday: tt.NewInt(1), Thursday: tt.NewInt(1), Friday: tt.NewInt(1), Saturday: tt.NewInt(1), Sunday: tt.NewInt(1),
			},
		},
		FrequencyList: []gtfs.Frequency{
			newTestFrequency("offset", 7*3600, 8*3600, 1800),
			newTestFrequency("meet", 10*3600, 11*3600, 1800),
			newTestFrequency("meet", 11*3600, 12*3600, 1800),
			newTestFrequency("meet", 12*3600+1800, 12*3600+1800, 600),
			newTestFrequency("empty", 13*3600, 13*3600, 600),
		},
	}
	for _, trip := range []struct {
		tripID string
		start  int
	}{
		{"regular", 5 * 3600},
		{"offset", 6 * 3600},
		{"meet", 10 * 3600},
		{"empty", 14 * 3600},
	} {
		r.TripList = append(r.TripList, gtfs.Trip{TripID: tt.NewString(trip.tripID), RouteID: tt.NewKey("route1"), ServiceID: tt.NewKey("service1")})
		r.StopTimeList = append(r.StopTimeList,
			gtfs.StopTime{TripID: tt.NewString(trip.tripID), StopID: tt.NewKey("stop1"), StopSequence: tt.NewInt(1), ArrivalTime: tt.NewSeconds(trip.start), DepartureTime: tt.NewSeconds(trip.start)},
			gtfs.StopTime{TripID: tt.NewString(trip.tripID), StopID: tt.NewKey("stop2"), StopSequence: tt.NewInt(2), ArrivalTime: tt.NewSeconds(trip.start + 600), DepartureTime: tt.NewSeconds(trip.start + 600)},
		)
	}
	return r
}

func newTestFrequency(tripID string, start int, end int, headway int) gtfs.Frequency {
	return gtfs.Frequency{TripID: tt.NewString(tripID), StartTime: tt.NewSeconds(start), EndTime: tt.NewSeconds(end), HeadwaySecs: tt.NewInt(headway)}
}
