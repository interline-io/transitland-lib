package testutil

import (
	"time"

	"github.com/interline-io/transitland-lib/adapters/direct"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tt"
)

// NewMinimalTestFeed returns a minimal mock Reader & ReaderTester values.
func NewMinimalTestFeed() (*ReaderTester, *direct.Reader) {
	r := &direct.Reader{
		AgencyList: []tl.Agency{
			{AgencyID: "agency1", AgencyName: "Agency 1", AgencyTimezone: "America/Los_Angeles", AgencyURL: "http://example.com"},
		},
		RouteList: []tl.Route{
			{RouteID: "route1", RouteShortName: "Route 1", RouteType: 1, AgencyID: "agency1"},
		},
		TripList: []tl.Trip{
			{TripID: "trip1", RouteID: "route1", ServiceID: "service1"},
		},
		StopList: []tl.Stop{
			{StopID: "stop1", StopName: "Stop 1", Geometry: tt.NewPoint(1, 2)},
			{StopID: "stop2", StopName: "Stop 2", Geometry: tt.NewPoint(3, 4)},
		},
		StopTimeList: []tl.StopTime{
			{StopID: "stop1", TripID: "trip1", StopSequence: 1, ArrivalTime: tt.NewSeconds(0), DepartureTime: tt.NewSeconds(5)},
			{StopID: "stop2", TripID: "trip1", StopSequence: 2, ArrivalTime: tt.NewSeconds(10), DepartureTime: tt.NewSeconds(15)},
		},
		ShapeList: []tl.Shape{
			{ShapeID: "shape1", Geometry: tt.NewLineStringFromFlatCoords([]float64{1, 2, 0, 3, 4, 0})},
		},
		CalendarList: []tl.Calendar{
			{ServiceID: "service1", StartDate: time.Now(), EndDate: time.Now()},
		},
		CalendarDateList: []tl.CalendarDate{
			{ServiceID: "service1", ExceptionType: 1, Date: time.Now()},
		},
		FeedInfoList: []tl.FeedInfo{
			{FeedVersion: "123", FeedPublisherURL: "http://example.com", FeedLang: "en-US", FeedPublisherName: "Example"},
		},
		FareRuleList: []tl.FareRule{
			{FareID: "fare1"},
		},
		FareAttributeList: []tl.FareAttribute{
			{FareID: "fare1", CurrencyType: "USD", Price: 1.0, PaymentMethod: 1, Transfers: tt.NewInt(1)},
		},
		FrequencyList: []tl.Frequency{
			{TripID: "trip1", HeadwaySecs: 600, StartTime: tt.NewSeconds(3600), EndTime: tt.NewSeconds(7200)},
		},
		TransferList: []tl.Transfer{
			{FromStopID: "stop1", ToStopID: "stop2", TransferType: 1},
		},
	}
	fe := &ReaderTester{
		Counts: map[string]int{
			"agency.txt":          1,
			"routes.txt":          1,
			"trips.txt":           1,
			"stops.txt":           2,
			"stop_times.txt":      2,
			"shapes.txt":          1,
			"calendar.txt":        1,
			"calendar_dates.txt":  1,
			"feed_info.txt":       1,
			"fare_rules.txt":      1,
			"fare_attributes.txt": 1,
			"frequency.txt":       1,
			"transfers.txt":       1,
		},
		EntityIDs: map[string][]string{
			"agency.txt":          {"agency1"},
			"routes.txt":          {"route1"},
			"trips.txt":           {"trip1"},
			"stops.txt":           {"stop1", "stop2"},
			"shapes.txt":          {"shape1"},
			"calendar.txt":        {"service1"},
			"fare_attributes.txt": {"fare1"},
		},
	}
	return fe, r
}
