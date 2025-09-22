package empty

import (
	"github.com/interline-io/transitland-lib/gtfs"
)

var bufferSize = 1000

// Reader is a mocked up Reader used for testing.
type Reader struct {
}

func NewReader() *Reader {
	return &Reader{}
}

func (mr *Reader) String() string {
	return "empty"
}

func (mr *Reader) Open() error {
	return nil
}

func (mr *Reader) Close() error {
	return nil
}

func (mr *Reader) ValidateStructure() []error {
	return []error{}
}

func (mr *Reader) ReadEntities(c any) error {
	return nil
}

func (mr *Reader) StopTimesByTripID(ids ...string) chan []gtfs.StopTime {
	return readNullEntities[[]gtfs.StopTime](mr)
}

// ShapesByShapeID .
func (mr *Reader) ShapesByShapeID(...string) chan []gtfs.Shape {
	return readNullEntities[[]gtfs.Shape](mr)
}

func (mr *Reader) Stops() chan gtfs.Stop {
	return readNullEntities[gtfs.Stop](mr)
}

func (mr *Reader) StopTimes() chan gtfs.StopTime {
	return readNullEntities[gtfs.StopTime](mr)
}

func (mr *Reader) Agencies() chan gtfs.Agency {
	return readNullEntities[gtfs.Agency](mr)
}

func (mr *Reader) Calendars() chan gtfs.Calendar {
	return readNullEntities[gtfs.Calendar](mr)
}

func (mr *Reader) CalendarDates() chan gtfs.CalendarDate {
	return readNullEntities[gtfs.CalendarDate](mr)
}

func (mr *Reader) FareAttributes() chan gtfs.FareAttribute {
	return readNullEntities[gtfs.FareAttribute](mr)
}

func (mr *Reader) FareRules() chan gtfs.FareRule {
	return readNullEntities[gtfs.FareRule](mr)
}

func (mr *Reader) FeedInfos() chan gtfs.FeedInfo {
	return readNullEntities[gtfs.FeedInfo](mr)
}

func (mr *Reader) Frequencies() chan gtfs.Frequency {
	return readNullEntities[gtfs.Frequency](mr)
}

func (mr *Reader) Routes() chan gtfs.Route {
	return readNullEntities[gtfs.Route](mr)
}

func (mr *Reader) Shapes() chan gtfs.Shape {
	return readNullEntities[gtfs.Shape](mr)
}

func (mr *Reader) Transfers() chan gtfs.Transfer {
	return readNullEntities[gtfs.Transfer](mr)
}

func (mr *Reader) Pathways() chan gtfs.Pathway {
	return readNullEntities[gtfs.Pathway](mr)
}

func (mr *Reader) Levels() chan gtfs.Level {
	return readNullEntities[gtfs.Level](mr)
}

func (mr *Reader) Trips() chan gtfs.Trip {
	return readNullEntities[gtfs.Trip](mr)
}

func (mr *Reader) Attributions() chan gtfs.Attribution {
	return readNullEntities[gtfs.Attribution](mr)
}

func (mr *Reader) Translations() chan gtfs.Translation {
	return readNullEntities[gtfs.Translation](mr)
}

func (mr *Reader) Areas() chan gtfs.Area {
	return readNullEntities[gtfs.Area](mr)
}

func (mr *Reader) StopAreas() chan gtfs.StopArea {
	return readNullEntities[gtfs.StopArea](mr)
}

func (mr *Reader) FareLegRules() chan gtfs.FareLegRule {
	return readNullEntities[gtfs.FareLegRule](mr)
}

func (mr *Reader) FareTransferRules() chan gtfs.FareTransferRule {
	return readNullEntities[gtfs.FareTransferRule](mr)
}

func (mr *Reader) FareMedia() chan gtfs.FareMedia {
	return readNullEntities[gtfs.FareMedia](mr)
}

func (mr *Reader) FareProducts() chan gtfs.FareProduct {
	return readNullEntities[gtfs.FareProduct](mr)
}

func (mr *Reader) RiderCategories() chan gtfs.RiderCategory {
	return readNullEntities[gtfs.RiderCategory](mr)
}

func (mr *Reader) FareLegJoinRules() chan gtfs.FareLegJoinRule {
	return readNullEntities[gtfs.FareLegJoinRule](mr)
}

func readNullEntities[T any](reader *Reader) chan T {
	out := make(chan T, bufferSize)
	go func() {
		close(out)
	}()
	return out
}
