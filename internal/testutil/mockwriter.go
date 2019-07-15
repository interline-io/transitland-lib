package testutil

import (
	"fmt"

	"github.com/interline-io/gotransit"
)

type MockWriter struct {
	Reader MockReader
}

func (mw *MockWriter) Open() error {
	return nil
}

func (mw *MockWriter) Close() error {
	return nil
}

func (w *MockWriter) Create() error {
	return nil
}

func (mw *MockWriter) Delete() error {
	return nil
}

func (mw *MockWriter) NewReader() (gotransit.Reader, error) {
	return &mw.Reader, nil
}

func (mw *MockWriter) AddEntity(ent gotransit.Entity) (string, error) {
	// fmt.Printf("writing: %#v\n", ent)
	switch v := ent.(type) {
	case *gotransit.Stop:
		mw.Reader.StopList = append(mw.Reader.StopList, *v)
	case *gotransit.StopTime:
		mw.Reader.StopTimeList = append(mw.Reader.StopTimeList, *v)
	case *gotransit.Agency:
		mw.Reader.AgencyList = append(mw.Reader.AgencyList, *v)
	case *gotransit.Calendar:
		mw.Reader.CalendarList = append(mw.Reader.CalendarList, *v)
	case *gotransit.CalendarDate:
		mw.Reader.CalendarDateList = append(mw.Reader.CalendarDateList, *v)
	case *gotransit.FareAttribute:
		mw.Reader.FareAttributeList = append(mw.Reader.FareAttributeList, *v)
	case *gotransit.FareRule:
		mw.Reader.FareRuleList = append(mw.Reader.FareRuleList, *v)
	case *gotransit.FeedInfo:
		mw.Reader.FeedInfoList = append(mw.Reader.FeedInfoList, *v)
	case *gotransit.Frequency:
		mw.Reader.FrequencyList = append(mw.Reader.FrequencyList, *v)
	case *gotransit.Route:
		mw.Reader.RouteList = append(mw.Reader.RouteList, *v)
	case *gotransit.Shape:
		mw.Reader.ShapeList = append(mw.Reader.ShapeList, *v)
	case *gotransit.Transfer:
		mw.Reader.TransferList = append(mw.Reader.TransferList, *v)
	case *gotransit.Trip:
		mw.Reader.TripList = append(mw.Reader.TripList, *v)
	default:
		fmt.Printf("mockreader cannot handle type: %T\n", v)
	}
	return ent.EntityID(), nil
}

func (mw *MockWriter) AddEntities([]gotransit.Entity) error {
	return nil
}
