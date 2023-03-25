package direct

import (
	"github.com/interline-io/transitland-lib/tl"
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

func (mw *Writer) String() string {
	return "mock"
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
	case *tl.Service:
		mw.Reader.CalendarList = append(mw.Reader.CalendarList, v.Calendar)
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
	case *tl.Translation:
		mw.Reader.TranslationList = append(mw.Reader.TranslationList, *v)
	case *tl.Attribution:
		mw.Reader.AttributionList = append(mw.Reader.AttributionList, *v)
	case *tl.Area:
		mw.Reader.AreaList = append(mw.Reader.AreaList, *v)
	case *tl.StopArea:
		mw.Reader.StopAreaList = append(mw.Reader.StopAreaList, *v)
	case *tl.FareLegRule:
		mw.Reader.FareLegRuleList = append(mw.Reader.FareLegRuleList, *v)
	case *tl.FareTransferRule:
		mw.Reader.FareTransferRuleList = append(mw.Reader.FareTransferRuleList, *v)
	case *tl.FareMedia:
		mw.Reader.FareMediaList = append(mw.Reader.FareMediaList, *v)
	case *tl.RiderCategory:
		mw.Reader.RiderCategoryList = append(mw.Reader.RiderCategoryList, *v)
	case *tl.FareProduct:
		mw.Reader.FareProductList = append(mw.Reader.FareProductList, *v)
	default:
		mw.Reader.OtherList = append(mw.Reader.OtherList, v)
	}
	return ent.EntityID(), nil
}

// AddEntities .
func (mw *Writer) AddEntities(ents []tl.Entity) ([]string, error) {
	retids := []string{}
	for _, ent := range ents {
		eid, err := mw.AddEntity(ent)
		if err != nil {
			return retids, err
		}
		retids = append(retids, eid)
	}
	return retids, nil
}
