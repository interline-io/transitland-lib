package mock

import (
	"fmt"

	"github.com/interline-io/gotransit"
)

// Writer is a mocked up Writer used in tests.
type Writer struct {
	Reader Reader
}

// Open .
func (mw *Writer) Open() error {
	return nil
}

// Close .
func (mw *Writer) Close() error {
	return nil
}

// Create .
func (mw *Writer) Create() error {
	return nil
}

// Delete .
func (mw *Writer) Delete() error {
	return nil
}

// NewReader .
func (mw *Writer) NewReader() (gotransit.Reader, error) {
	return &mw.Reader, nil
}

// AddEntity .
func (mw *Writer) AddEntity(ent gotransit.Entity) (string, error) {
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
		return "", fmt.Errorf("mockreader cannot handle type: %T", v)
	}
	return ent.EntityID(), nil
}

// AddEntities .
func (mw *Writer) AddEntities(ents []gotransit.Entity) error {
	for _, ent := range ents {
		if _, err := mw.AddEntity(ent); err != nil {
			return err
		}
	}
	return nil
}
