package testutil

import (
	"time"

	"github.com/interline-io/gotransit"
)

func ExampleReader() *MockReader {
	return &MockReader{
		AgencyList: []gotransit.Agency{
			{AgencyID: "agency1", AgencyName: "Agency 1", AgencyTimezone: "America/Los_Angeles", AgencyURL: "http://example.com"},
		},
		RouteList: []gotransit.Route{
			{RouteID: "route1", RouteShortName: "Route 1", RouteType: 1},
		},
		TripList: []gotransit.Trip{
			{TripID: "trip1", RouteID: "route1", ServiceID: "service1"},
		},
		StopList: []gotransit.Stop{
			{StopID: "stop1", StopName: "Stop 1", Geometry: gotransit.NewPoint(0, 0)},
			{StopID: "stop2", StopName: "Stop 2", Geometry: gotransit.NewPoint(0, 0)},
		},
		StopTimeList: []gotransit.StopTime{
			{StopID: "stop1"},
		},
		ShapeList: []gotransit.Shape{
			{ShapeID: "shape1"},
		},
		CalendarList: []gotransit.Calendar{
			{ServiceID: "service1"},
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
			{FareID: "fare1", CurrencyType: "USD"},
		},
		FrequencyList: []gotransit.Frequency{
			{TripID: "trip1", HeadwaySecs: 600},
		},
		TransferList: []gotransit.Transfer{
			{FromStopID: "stop1", ToStopID: "stop2"},
		},
	}
}
