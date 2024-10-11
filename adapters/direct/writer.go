package direct

import (
	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/service"
	"github.com/interline-io/transitland-lib/tt"
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
func (mw *Writer) NewReader() (adapters.Reader, error) {
	return &mw.Reader, nil
}

// AddEntity .
func (mw *Writer) AddEntity(ent tt.Entity) (string, error) {
	switch v := ent.(type) {
	case *gtfs.Stop:
		mw.Reader.StopList = append(mw.Reader.StopList, *v)
	case *gtfs.StopTime:
		mw.Reader.StopTimeList = append(mw.Reader.StopTimeList, *v)
	case *gtfs.Agency:
		mw.Reader.AgencyList = append(mw.Reader.AgencyList, *v)
	case *service.Service:
		mw.Reader.CalendarList = append(mw.Reader.CalendarList, v.Calendar)
	case *gtfs.Calendar:
		mw.Reader.CalendarList = append(mw.Reader.CalendarList, *v)
	case *gtfs.CalendarDate:
		mw.Reader.CalendarDateList = append(mw.Reader.CalendarDateList, *v)
	case *gtfs.FareAttribute:
		mw.Reader.FareAttributeList = append(mw.Reader.FareAttributeList, *v)
	case *gtfs.FareRule:
		mw.Reader.FareRuleList = append(mw.Reader.FareRuleList, *v)
	case *gtfs.FeedInfo:
		mw.Reader.FeedInfoList = append(mw.Reader.FeedInfoList, *v)
	case *gtfs.Frequency:
		mw.Reader.FrequencyList = append(mw.Reader.FrequencyList, *v)
	case *gtfs.Route:
		mw.Reader.RouteList = append(mw.Reader.RouteList, *v)
	case *gtfs.Shape:
		mw.Reader.ShapeList = append(mw.Reader.ShapeList, *v)
	case *gtfs.Transfer:
		mw.Reader.TransferList = append(mw.Reader.TransferList, *v)
	case *gtfs.Trip:
		mw.Reader.TripList = append(mw.Reader.TripList, *v)
	case *gtfs.Translation:
		mw.Reader.TranslationList = append(mw.Reader.TranslationList, *v)
	case *gtfs.Attribution:
		mw.Reader.AttributionList = append(mw.Reader.AttributionList, *v)
	case *gtfs.Area:
		mw.Reader.AreaList = append(mw.Reader.AreaList, *v)
	case *gtfs.StopArea:
		mw.Reader.StopAreaList = append(mw.Reader.StopAreaList, *v)
	case *gtfs.FareLegRule:
		mw.Reader.FareLegRuleList = append(mw.Reader.FareLegRuleList, *v)
	case *gtfs.FareTransferRule:
		mw.Reader.FareTransferRuleList = append(mw.Reader.FareTransferRuleList, *v)
	case *gtfs.FareMedia:
		mw.Reader.FareMediaList = append(mw.Reader.FareMediaList, *v)
	case *gtfs.RiderCategory:
		mw.Reader.RiderCategoryList = append(mw.Reader.RiderCategoryList, *v)
	case *gtfs.FareProduct:
		mw.Reader.FareProductList = append(mw.Reader.FareProductList, *v)
	default:
		mw.Reader.OtherList = append(mw.Reader.OtherList, v)
	}
	return ent.EntityID(), nil
}

// AddEntities .
func (mw *Writer) AddEntities(ents []tt.Entity) ([]string, error) {
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
