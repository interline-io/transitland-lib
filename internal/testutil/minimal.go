package testutil

import (
	"time"

	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/internal/mock"
)

// NewMinimalExpect returns a minimal MockReader & Expect values.
func NewMinimalExpect() (*ExpectEntities, *mock.Reader) {
	r := &mock.Reader{
		AgencyList: []gotransit.Agency{
			{AgencyID: "agency1", AgencyName: "Agency 1", AgencyTimezone: "America/Los_Angeles", AgencyURL: "http://example.com"},
		},
		RouteList: []gotransit.Route{
			{RouteID: "route1", RouteShortName: "Route 1", RouteType: 1, AgencyID: "agency1"},
		},
		TripList: []gotransit.Trip{
			{TripID: "trip1", RouteID: "route1", ServiceID: "service1"},
		},
		StopList: []gotransit.Stop{
			{StopID: "stop1", StopName: "Stop 1", Geometry: gotransit.NewPoint(1, 2)},
			{StopID: "stop2", StopName: "Stop 2", Geometry: gotransit.NewPoint(3, 4)},
		},
		StopTimeList: []gotransit.StopTime{
			{StopID: "stop1", TripID: "trip1", StopSequence: 1, ArrivalTime: 0, DepartureTime: 5},
			{StopID: "stop2", TripID: "trip1", StopSequence: 2, ArrivalTime: 10, DepartureTime: 15},
		},
		ShapeList: []gotransit.Shape{
			{ShapeID: "shape1", Geometry: gotransit.NewLineStringFromFlatCoords([]float64{1, 2, 0, 3, 4, 0})},
		},
		CalendarList: []gotransit.Calendar{
			{ServiceID: "service1", StartDate: time.Now(), EndDate: time.Now()},
		},
		CalendarDateList: []gotransit.CalendarDate{
			{ServiceID: "service1", ExceptionType: 1, Date: time.Now()},
		},
		FeedInfoList: []gotransit.FeedInfo{
			{FeedVersion: "123", FeedPublisherURL: "http://example.com", FeedLang: "en-US", FeedPublisherName: "Example"},
		},
		FareRuleList: []gotransit.FareRule{
			{FareID: "fare1"},
		},
		FareAttributeList: []gotransit.FareAttribute{
			{FareID: "fare1", CurrencyType: "USD", Price: 1.0, PaymentMethod: 1, Transfers: "1"},
		},
		FrequencyList: []gotransit.Frequency{
			{TripID: "trip1", HeadwaySecs: 600, StartTime: gotransit.WideTime{Seconds: 3600}, EndTime: gotransit.WideTime{Seconds: 7200}},
		},
		TransferList: []gotransit.Transfer{
			{FromStopID: "stop1", ToStopID: "stop2", TransferType: 1},
		},
	}
	fe := &ExpectEntities{
		AgencyCount:        1,
		RouteCount:         1,
		TripCount:          1,
		StopCount:          2,
		StopTimeCount:      2,
		ShapeCount:         1,
		CalendarCount:      1,
		CalendarDateCount:  1,
		FeedInfoCount:      1,
		FareRuleCount:      1,
		FareAttributeCount: 1,
		FrequencyCount:     1,
		TransferCount:      1,
		ExpectAgencyIDs:    []string{"agency1"},
		ExpectRouteIDs:     []string{"route1"},
		ExpectTripIDs:      []string{"trip1"},
		ExpectStopIDs:      []string{"stop1", "stop2"},
		ExpectShapeIDs:     []string{"shape1"},
		ExpectCalendarIDs:  []string{"service1"},
		ExpectFareIDs:      []string{"fare1"},
	}
	return fe, r
}
