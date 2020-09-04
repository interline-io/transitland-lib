package mock

import (
	"fmt"

	tl "github.com/interline-io/transitland-lib"
)

// Writer is a mocked up Writer used in tests.
type Writer struct {
	Reader Reader
}

// NewWriter returns a new Writer.
func NewWriter() *Writer {
	return &Writer{
		Reader: *NewReader(),
	}
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
func (mw *Writer) NewReader() (tl.Reader, error) {
	return &mw.Reader, nil
}

// AddEntity .
func (mw *Writer) AddEntity(ent tl.Entity) (string, error) {
	switch v := ent.(type) {
	case *tl.Stop:
		mw.Reader.StopList = append(mw.Reader.StopList, *v)
	case *tl.StopTime:
		mw.Reader.StopTimeList = append(mw.Reader.StopTimeList, *v)
	case *tl.Agency:
		mw.Reader.AgencyList = append(mw.Reader.AgencyList, *v)
	case *tl.Calendar:
		mw.Reader.CalendarList = append(mw.Reader.CalendarList, *v)
	case *tl.CalendarDate:
		mw.Reader.CalendarDateList = append(mw.Reader.CalendarDateList, *v)
	case *tl.FareAttribute:
		mw.Reader.FareAttributeList = append(mw.Reader.FareAttributeList, *v)
	case *tl.FareRule:
		mw.Reader.FareRuleList = append(mw.Reader.FareRuleList, *v)
	case *tl.FeedInfo:
		mw.Reader.FeedInfoList = append(mw.Reader.FeedInfoList, *v)
	case *tl.Frequency:
		mw.Reader.FrequencyList = append(mw.Reader.FrequencyList, *v)
	case *tl.Route:
		mw.Reader.RouteList = append(mw.Reader.RouteList, *v)
	case *tl.Shape:
		mw.Reader.ShapeList = append(mw.Reader.ShapeList, *v)
	case *tl.Transfer:
		mw.Reader.TransferList = append(mw.Reader.TransferList, *v)
	case *tl.Trip:
		mw.Reader.TripList = append(mw.Reader.TripList, *v)
	default:
		return "", fmt.Errorf("mockreader cannot handle type: %T", v)
	}
	return ent.EntityID(), nil
}

// AddEntities .
func (mw *Writer) AddEntities(ents []tl.Entity) error {
	for _, ent := range ents {
		if _, err := mw.AddEntity(ent); err != nil {
			return err
		}
	}
	return nil
}
