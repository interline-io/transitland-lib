package multireader

import (
	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/gtfs"
)

var bufferSize = 1000

func init() {
	var _ adapters.Reader = &Reader{}
}

// Reader is a mocked up Reader used for testing.
type Reader struct {
	Readers []adapters.Reader
}

func NewReader(readers ...adapters.Reader) *Reader {
	return &Reader{Readers: readers}
}

func (mr *Reader) String() string {
	return "multireader"
}

func (mr *Reader) Open() error {
	for _, r := range mr.Readers {
		if err := r.Open(); err != nil {
			return err
		}
	}
	return nil
}

func (mr *Reader) Close() error {
	for _, r := range mr.Readers {
		if err := r.Close(); err != nil {
			return err
		}
	}
	return nil
}

func (mr *Reader) ValidateStructure() []error {
	var ret []error
	for _, reader := range mr.Readers {
		errs := reader.ValidateStructure()
		ret = append(ret, errs...)
	}
	return ret
}

func (mr *Reader) ReadEntities(c any) error {
	for _, r := range mr.Readers {
		if err := r.ReadEntities(c); err != nil {
			return err
		}
	}
	return nil
}

func (mr *Reader) StopTimesByTripID(ids ...string) chan []gtfs.StopTime {
	return readEntities(mr, func(r adapters.Reader) chan []gtfs.StopTime { return r.StopTimesByTripID(ids...) }, nil)
}

func (mr *Reader) ShapesByShapeID(ids ...string) chan []gtfs.Shape {
	return readEntities(mr, func(r adapters.Reader) chan []gtfs.Shape { return r.ShapesByShapeID(ids...) }, nil)
}

func (mr *Reader) Stops() chan gtfs.Stop {
	return readEntities(mr, func(r adapters.Reader) chan gtfs.Stop { return r.Stops() }, setFv[*gtfs.Stop])
}

func (mr *Reader) StopTimes() chan gtfs.StopTime {
	return readEntities(mr, func(r adapters.Reader) chan gtfs.StopTime { return r.StopTimes() }, setFv[*gtfs.StopTime])
}

func (mr *Reader) Agencies() chan gtfs.Agency {
	return readEntities(mr, func(r adapters.Reader) chan gtfs.Agency { return r.Agencies() }, setFv[*gtfs.Agency])
}

func (mr *Reader) Calendars() chan gtfs.Calendar {
	return readEntities(mr, func(r adapters.Reader) chan gtfs.Calendar { return r.Calendars() }, setFv[*gtfs.Calendar])
}

func (mr *Reader) CalendarDates() chan gtfs.CalendarDate {
	return readEntities(mr, func(r adapters.Reader) chan gtfs.CalendarDate { return r.CalendarDates() }, setFv[*gtfs.CalendarDate])
}

func (mr *Reader) FareAttributes() chan gtfs.FareAttribute {
	return readEntities(mr, func(r adapters.Reader) chan gtfs.FareAttribute { return r.FareAttributes() }, setFv[*gtfs.FareAttribute])
}

func (mr *Reader) FareRules() chan gtfs.FareRule {
	return readEntities(mr, func(r adapters.Reader) chan gtfs.FareRule { return r.FareRules() }, setFv[*gtfs.FareRule])
}

func (mr *Reader) FeedInfos() chan gtfs.FeedInfo {
	return readEntities(mr, func(r adapters.Reader) chan gtfs.FeedInfo { return r.FeedInfos() }, setFv[*gtfs.FeedInfo])
}

func (mr *Reader) Frequencies() chan gtfs.Frequency {
	return readEntities(mr, func(r adapters.Reader) chan gtfs.Frequency { return r.Frequencies() }, setFv[*gtfs.Frequency])
}

func (mr *Reader) Routes() chan gtfs.Route {
	return readEntities(mr, func(r adapters.Reader) chan gtfs.Route { return r.Routes() }, setFv[*gtfs.Route])
}

func (mr *Reader) Shapes() chan gtfs.Shape {
	return readEntities(mr, func(r adapters.Reader) chan gtfs.Shape { return r.Shapes() }, setFv[*gtfs.Shape])
}

func (mr *Reader) Transfers() chan gtfs.Transfer {
	return readEntities(mr, func(r adapters.Reader) chan gtfs.Transfer { return r.Transfers() }, setFv[*gtfs.Transfer])
}

func (mr *Reader) Pathways() chan gtfs.Pathway {
	return readEntities(mr, func(r adapters.Reader) chan gtfs.Pathway { return r.Pathways() }, setFv[*gtfs.Pathway])
}

func (mr *Reader) Levels() chan gtfs.Level {
	return readEntities(mr, func(r adapters.Reader) chan gtfs.Level { return r.Levels() }, setFv[*gtfs.Level])
}

func (mr *Reader) Trips() chan gtfs.Trip {
	return readEntities(mr, func(r adapters.Reader) chan gtfs.Trip { return r.Trips() }, setFv[*gtfs.Trip])
}

func (mr *Reader) Attributions() chan gtfs.Attribution {
	return readEntities(mr, func(r adapters.Reader) chan gtfs.Attribution { return r.Attributions() }, setFv[*gtfs.Attribution])
}

func (mr *Reader) Translations() chan gtfs.Translation {
	return readEntities(mr, func(r adapters.Reader) chan gtfs.Translation { return r.Translations() }, setFv[*gtfs.Translation])
}

func (mr *Reader) Areas() chan gtfs.Area {
	return readEntities(mr, func(r adapters.Reader) chan gtfs.Area { return r.Areas() }, setFv[*gtfs.Area])
}

func (mr *Reader) StopAreas() chan gtfs.StopArea {
	return readEntities(mr, func(r adapters.Reader) chan gtfs.StopArea { return r.StopAreas() }, setFv[*gtfs.StopArea])
}

func (mr *Reader) FareLegRules() chan gtfs.FareLegRule {
	return readEntities(mr, func(r adapters.Reader) chan gtfs.FareLegRule { return r.FareLegRules() }, setFv[*gtfs.FareLegRule])
}

func (mr *Reader) FareTransferRules() chan gtfs.FareTransferRule {
	return readEntities(mr, func(r adapters.Reader) chan gtfs.FareTransferRule { return r.FareTransferRules() }, setFv[*gtfs.FareTransferRule])
}

func (mr *Reader) FareMedia() chan gtfs.FareMedia {
	return readEntities(mr, func(r adapters.Reader) chan gtfs.FareMedia { return r.FareMedia() }, setFv[*gtfs.FareMedia])
}

func (mr *Reader) FareProducts() chan gtfs.FareProduct {
	return readEntities(mr, func(r adapters.Reader) chan gtfs.FareProduct { return r.FareProducts() }, setFv[*gtfs.FareProduct])
}

func (mr *Reader) RiderCategories() chan gtfs.RiderCategory {
	return readEntities(mr, func(r adapters.Reader) chan gtfs.RiderCategory { return r.RiderCategories() }, setFv[*gtfs.RiderCategory])
}

func (mr *Reader) Timeframes() chan gtfs.Timeframe {
	return readEntities(mr, func(r adapters.Reader) chan gtfs.Timeframe { return r.Timeframes() }, setFv[*gtfs.Timeframe])
}

func (mr *Reader) Networks() chan gtfs.Network {
	return readEntities(mr, func(r adapters.Reader) chan gtfs.Network { return r.Networks() }, setFv[*gtfs.Network])
}

func (mr *Reader) RouteNetworks() chan gtfs.RouteNetwork {
	return readEntities(mr, func(r adapters.Reader) chan gtfs.RouteNetwork { return r.RouteNetworks() }, setFv[*gtfs.RouteNetwork])
}

func (mr *Reader) FareLegJoinRules() chan gtfs.FareLegJoinRule {
	return readEntities(mr, func(r adapters.Reader) chan gtfs.FareLegJoinRule { return r.FareLegJoinRules() }, setFv[*gtfs.FareLegJoinRule])
}

type canSetFV interface {
	SetFeedVersionID(int)
}

func setFv[T canSetFV](fvid int, ent T) {
	ent.SetFeedVersionID(fvid)
}

// Note this could be slightly simplified...
//
//	func readEntities[T any, T2 interface {
//			*T
//			canSetFV
//	}](reader *Reader, cf func(r adapters.Reader) chan T) {
//
//			...
//			var t2 T2
//	 	t2 = &ent
//	 	t2.SetFeedVersionID(i, &ent)
//		}
func readEntities[T any](reader *Reader, cf func(r adapters.Reader) chan T, cb func(int, *T)) chan T {
	out := make(chan T, bufferSize)
	go func() {
		for i, r := range reader.Readers {
			readin := cf(r)
			for ent := range readin {
				if cb != nil {
					cb(i, &ent)
				}
				out <- ent
			}
		}
		close(out)
	}()
	return out
}
