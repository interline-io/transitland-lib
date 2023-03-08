package multireader

import (
	"github.com/interline-io/transitland-lib/tl"
)

var bufferSize = 1000

func init() {
	var v tl.Reader
	v = &Reader{}
	_ = v
}

// Reader is a mocked up Reader used for testing.
type Reader struct {
	Readers []tl.Reader
}

func NewReader(readers ...tl.Reader) *Reader {
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

func (mr *Reader) StopTimesByTripID(ids ...string) chan []tl.StopTime {
	return readEntities(mr, func(r tl.Reader) chan []tl.StopTime { return r.StopTimesByTripID(ids...) }, nil)
}

func (mr *Reader) Stops() chan tl.Stop {
	return readEntities(mr, func(r tl.Reader) chan tl.Stop { return r.Stops() }, setFv[*tl.Stop])
}

func (mr *Reader) StopTimes() chan tl.StopTime {
	return readEntities(mr, func(r tl.Reader) chan tl.StopTime { return r.StopTimes() }, setFv[*tl.StopTime])
}

func (mr *Reader) Agencies() chan tl.Agency {
	return readEntities(mr, func(r tl.Reader) chan tl.Agency { return r.Agencies() }, setFv[*tl.Agency])
}

func (mr *Reader) Calendars() chan tl.Calendar {
	return readEntities(mr, func(r tl.Reader) chan tl.Calendar { return r.Calendars() }, setFv[*tl.Calendar])
}

func (mr *Reader) CalendarDates() chan tl.CalendarDate {
	return readEntities(mr, func(r tl.Reader) chan tl.CalendarDate { return r.CalendarDates() }, setFv[*tl.CalendarDate])
}

func (mr *Reader) FareAttributes() chan tl.FareAttribute {
	return readEntities(mr, func(r tl.Reader) chan tl.FareAttribute { return r.FareAttributes() }, setFv[*tl.FareAttribute])
}

func (mr *Reader) FareRules() chan tl.FareRule {
	return readEntities(mr, func(r tl.Reader) chan tl.FareRule { return r.FareRules() }, setFv[*tl.FareRule])
}

func (mr *Reader) FeedInfos() chan tl.FeedInfo {
	return readEntities(mr, func(r tl.Reader) chan tl.FeedInfo { return r.FeedInfos() }, setFv[*tl.FeedInfo])
}

func (mr *Reader) Frequencies() chan tl.Frequency {
	return readEntities(mr, func(r tl.Reader) chan tl.Frequency { return r.Frequencies() }, setFv[*tl.Frequency])
}

func (mr *Reader) Routes() chan tl.Route {
	return readEntities(mr, func(r tl.Reader) chan tl.Route { return r.Routes() }, setFv[*tl.Route])
}

func (mr *Reader) Shapes() chan tl.Shape {
	return readEntities(mr, func(r tl.Reader) chan tl.Shape { return r.Shapes() }, setFv[*tl.Shape])
}

func (mr *Reader) Transfers() chan tl.Transfer {
	return readEntities(mr, func(r tl.Reader) chan tl.Transfer { return r.Transfers() }, setFv[*tl.Transfer])
}

func (mr *Reader) Pathways() chan tl.Pathway {
	return readEntities(mr, func(r tl.Reader) chan tl.Pathway { return r.Pathways() }, setFv[*tl.Pathway])
}

func (mr *Reader) Levels() chan tl.Level {
	return readEntities(mr, func(r tl.Reader) chan tl.Level { return r.Levels() }, setFv[*tl.Level])
}

func (mr *Reader) Trips() chan tl.Trip {
	return readEntities(mr, func(r tl.Reader) chan tl.Trip { return r.Trips() }, setFv[*tl.Trip])
}

func (mr *Reader) Attributions() chan tl.Attribution {
	return readEntities(mr, func(r tl.Reader) chan tl.Attribution { return r.Attributions() }, setFv[*tl.Attribution])
}

func (mr *Reader) Translations() chan tl.Translation {
	return readEntities(mr, func(r tl.Reader) chan tl.Translation { return r.Translations() }, setFv[*tl.Translation])
}

func (mr *Reader) Areas() chan tl.Area {
	return readEntities(mr, func(r tl.Reader) chan tl.Area { return r.Areas() }, setFv[*tl.Area])
}

func (mr *Reader) StopAreas() chan tl.StopArea {
	return readEntities(mr, func(r tl.Reader) chan tl.StopArea { return r.StopAreas() }, setFv[*tl.StopArea])
}

func (mr *Reader) FareLegRules() chan tl.FareLegRule {
	return readEntities(mr, func(r tl.Reader) chan tl.FareLegRule { return r.FareLegRules() }, setFv[*tl.FareLegRule])
}

func (mr *Reader) FareTransferRules() chan tl.FareTransferRule {
	return readEntities(mr, func(r tl.Reader) chan tl.FareTransferRule { return r.FareTransferRules() }, setFv[*tl.FareTransferRule])
}

func (mr *Reader) FareContainers() chan tl.FareContainer {
	return readEntities(mr, func(r tl.Reader) chan tl.FareContainer { return r.FareContainers() }, setFv[*tl.FareContainer])
}

func (mr *Reader) FareProducts() chan tl.FareProduct {
	return readEntities(mr, func(r tl.Reader) chan tl.FareProduct { return r.FareProducts() }, setFv[*tl.FareProduct])
}

func (mr *Reader) RiderCategories() chan tl.RiderCategory {
	return readEntities(mr, func(r tl.Reader) chan tl.RiderCategory { return r.RiderCategories() }, setFv[*tl.RiderCategory])
}

type canSetFV interface {
	SetFeedVersionID(int)
}

func setFv[T canSetFV](fvid int, ent T) {
	ent.SetFeedVersionID(fvid)
}

// Note this could be slightly simplified...
// func readEntities[T any, T2 interface {
//		*T
//		canSetFV
// }](reader *Reader, cf func(r tl.Reader) chan T) {
//		...
//		var t2 T2
//  	t2 = &ent
//  	t2.SetFeedVersionID(i, &ent)
//	}
func readEntities[T any](reader *Reader, cf func(r tl.Reader) chan T, cb func(int, *T)) chan T {
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
