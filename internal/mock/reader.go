package mock

import (
	"reflect"

	"github.com/interline-io/gotransit"
)

var bufferSize = 1000

// Reader is a mocked up Reader used for testing.
type Reader struct {
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

// NewReader returns a new Reader.
func NewReader() *Reader {
	return &Reader{}
}

// Open .
func (mr *Reader) Open() error {
	return nil
}

// Close .
func (mr *Reader) Close() error {
	return nil
}

// ValidateStructure .
func (mr *Reader) ValidateStructure() []error {
	return []error{}
}

// StopTimesByTripID .
func (mr *Reader) StopTimesByTripID(...string) chan []gotransit.StopTime {
	out := make(chan []gotransit.StopTime, 1000)
	go func() {
		sts := map[string][]gotransit.StopTime{}
		for _, ent := range mr.StopTimeList {
			sts[ent.TripID] = append(sts[ent.TripID], ent)
		}
		for _, v := range sts {
			out <- v
		}
		close(out)
	}()
	return out
}

// ShapesByShapeID .
func (mr *Reader) ShapesByShapeID(...string) chan []gotransit.Shape {
	c := make(chan []gotransit.Shape, 1000)
	close(c)
	return c
}

// ShapeLinesByShapeID .
func (mr *Reader) ShapeLinesByShapeID(...string) chan gotransit.Shape {
	out := make(chan gotransit.Shape, 1000)
	go func() {
		for _, ent := range mr.ShapeList {
			out <- ent
		}
		close(out)
	}()
	return out
}

// ReadEntities - Only StopTimes are supported; just for passing tests!!
func (mr *Reader) ReadEntities(c interface{}) error {
	outValue := reflect.ValueOf(c)
	for _, ent := range mr.StopTimeList {
		outValue.Send(reflect.ValueOf(ent))
	}
	outValue.Close()
	return nil
}

// Stops .
func (mr *Reader) Stops() chan gotransit.Stop {
	out := make(chan gotransit.Stop, bufferSize)
	go func() {
		for _, ent := range mr.StopList {
			out <- ent
		}
		close(out)
	}()
	return out
}

// StopTimes .
func (mr *Reader) StopTimes() chan gotransit.StopTime {
	out := make(chan gotransit.StopTime, bufferSize)
	go func() {
		for _, ent := range mr.StopTimeList {
			out <- ent
		}
		close(out)
	}()
	return out
}

// Agencies .
func (mr *Reader) Agencies() chan gotransit.Agency {
	out := make(chan gotransit.Agency, bufferSize)
	go func() {
		for _, ent := range mr.AgencyList {
			out <- ent
		}
		close(out)
	}()
	return out
}

// Calendars .
func (mr *Reader) Calendars() chan gotransit.Calendar {
	out := make(chan gotransit.Calendar, bufferSize)
	go func() {
		for _, ent := range mr.CalendarList {
			out <- ent
		}
		close(out)
	}()
	return out
}

// CalendarDates .
func (mr *Reader) CalendarDates() chan gotransit.CalendarDate {
	out := make(chan gotransit.CalendarDate, bufferSize)
	go func() {
		for _, ent := range mr.CalendarDateList {
			out <- ent
		}
		close(out)
	}()
	return out
}

// FareAttributes .
func (mr *Reader) FareAttributes() chan gotransit.FareAttribute {
	out := make(chan gotransit.FareAttribute, bufferSize)
	go func() {
		for _, ent := range mr.FareAttributeList {
			out <- ent
		}
		close(out)
	}()
	return out
}

// FareRules .
func (mr *Reader) FareRules() chan gotransit.FareRule {
	out := make(chan gotransit.FareRule, bufferSize)
	go func() {
		for _, ent := range mr.FareRuleList {
			out <- ent
		}
		close(out)
	}()
	return out
}

// FeedInfos .
func (mr *Reader) FeedInfos() chan gotransit.FeedInfo {
	out := make(chan gotransit.FeedInfo, bufferSize)
	go func() {
		for _, ent := range mr.FeedInfoList {
			out <- ent
		}
		close(out)
	}()
	return out
}

// Frequencies .
func (mr *Reader) Frequencies() chan gotransit.Frequency {
	out := make(chan gotransit.Frequency, bufferSize)
	go func() {
		for _, ent := range mr.FrequencyList {
			out <- ent
		}
		close(out)
	}()
	return out
}

// Routes .
func (mr *Reader) Routes() chan gotransit.Route {
	out := make(chan gotransit.Route, bufferSize)
	go func() {
		for _, ent := range mr.RouteList {
			out <- ent
		}
		close(out)
	}()
	return out
}

// Shapes .
func (mr *Reader) Shapes() chan gotransit.Shape {
	out := make(chan gotransit.Shape, bufferSize)
	go func() {
		for _, ent := range mr.ShapeList {
			out <- ent
		}
		close(out)
	}()
	return out
}

// Transfers .
func (mr *Reader) Transfers() chan gotransit.Transfer {
	out := make(chan gotransit.Transfer, bufferSize)
	go func() {
		for _, ent := range mr.TransferList {
			out <- ent
		}
		close(out)
	}()
	return out
}

// Trips .
func (mr *Reader) Trips() chan gotransit.Trip {
	out := make(chan gotransit.Trip, bufferSize)
	go func() {
		for _, ent := range mr.TripList {
			out <- ent
		}
		close(out)
	}()
	return out
}
