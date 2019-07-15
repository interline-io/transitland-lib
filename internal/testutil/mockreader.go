package testutil

import (
	"github.com/interline-io/gotransit"
)

var bufferSize = 1000

type MockReader struct {
	AgencyList        []gotransit.Agency
	RouteList         []gotransit.Route
	TripList          []gotransit.Trip
	StopList          []gotransit.Stop
	StopTimeList      []gotransit.StopTime
	ShapeList         []gotransit.Shape
	CalendarList      []gotransit.Calendar
	CalendarDateList  []gotransit.CalendarDate
	FeedInfoList      []gotransit.FeedInfo
	FareRuleList      []gotransit.FareRule
	FareAttributeList []gotransit.FareAttribute
	FrequencyList     []gotransit.Frequency
	TransferList      []gotransit.Transfer
}

func (mr *MockReader) Open() error {
	return nil
}

func (mr *MockReader) Close() error {
	return nil
}

func (mr *MockReader) ValidateStructure() []error {
	return []error{}
}

func (mr *MockReader) StopTimesByTripID(...string) chan []gotransit.StopTime {
	c := make(chan []gotransit.StopTime, 1000)
	close(c)
	return c
}

func (mr *MockReader) ShapesByShapeID(...string) chan []gotransit.Shape {
	c := make(chan []gotransit.Shape, 1000)
	close(c)
	return c
}

func (mr *MockReader) ShapeLinesByShapeID(...string) chan gotransit.Shape {
	c := make(chan gotransit.Shape, 1000)
	close(c)
	return c
}

func (mr *MockReader) ReadEntities(c interface{}) error {
	return nil
}

func (mr *MockReader) Stops() chan gotransit.Stop {
	out := make(chan gotransit.Stop, bufferSize)
	go func() {
		for _, ent := range mr.StopList {
			out <- ent
		}
		close(out)
	}()
	return out
}

func (mr *MockReader) StopTimes() chan gotransit.StopTime {
	out := make(chan gotransit.StopTime, bufferSize)
	go func() {
		for _, ent := range mr.StopTimeList {
			out <- ent
		}
		close(out)
	}()
	return out
}

func (mr *MockReader) Agencies() chan gotransit.Agency {
	out := make(chan gotransit.Agency, bufferSize)
	go func() {
		for _, ent := range mr.AgencyList {
			out <- ent
		}
		close(out)
	}()
	return out
}

func (mr *MockReader) Calendars() chan gotransit.Calendar {
	out := make(chan gotransit.Calendar, bufferSize)
	go func() {
		for _, ent := range mr.CalendarList {
			out <- ent
		}
		close(out)
	}()
	return out
}

func (mr *MockReader) CalendarDates() chan gotransit.CalendarDate {
	out := make(chan gotransit.CalendarDate, bufferSize)
	go func() {
		for _, ent := range mr.CalendarDateList {
			out <- ent
		}
		close(out)
	}()
	return out
}

func (mr *MockReader) FareAttributes() chan gotransit.FareAttribute {
	out := make(chan gotransit.FareAttribute, bufferSize)
	go func() {
		for _, ent := range mr.FareAttributeList {
			out <- ent
		}
		close(out)
	}()
	return out
}

func (mr *MockReader) FareRules() chan gotransit.FareRule {
	out := make(chan gotransit.FareRule, bufferSize)
	go func() {
		for _, ent := range mr.FareRuleList {
			out <- ent
		}
		close(out)
	}()
	return out
}

func (mr *MockReader) FeedInfos() chan gotransit.FeedInfo {
	out := make(chan gotransit.FeedInfo, bufferSize)
	go func() {
		for _, ent := range mr.FeedInfoList {
			out <- ent
		}
		close(out)
	}()
	return out
}

func (mr *MockReader) Frequencies() chan gotransit.Frequency {
	out := make(chan gotransit.Frequency, bufferSize)
	go func() {
		for _, ent := range mr.FrequencyList {
			out <- ent
		}
		close(out)
	}()
	return out
}

func (mr *MockReader) Routes() chan gotransit.Route {
	out := make(chan gotransit.Route, bufferSize)
	go func() {
		for _, ent := range mr.RouteList {
			out <- ent
		}
		close(out)
	}()
	return out
}

func (mr *MockReader) Shapes() chan gotransit.Shape {
	out := make(chan gotransit.Shape, bufferSize)
	go func() {
		for _, ent := range mr.ShapeList {
			out <- ent
		}
		close(out)
	}()
	return out
}

func (mr *MockReader) Transfers() chan gotransit.Transfer {
	out := make(chan gotransit.Transfer, bufferSize)
	go func() {
		for _, ent := range mr.TransferList {
			out <- ent
		}
		close(out)
	}()
	return out
}

func (mr *MockReader) Trips() chan gotransit.Trip {
	out := make(chan gotransit.Trip, bufferSize)
	go func() {
		for _, ent := range mr.TripList {
			out <- ent
		}
		close(out)
	}()
	return out
}
