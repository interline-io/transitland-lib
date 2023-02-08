package multi

import (
	"github.com/interline-io/transitland-lib/tl"
)

var bufferSize = 1000

type Reader struct {
	readers []tl.Reader
}

func NewReader(readers []tl.Reader) *Reader {
	return &Reader{readers: readers}
}

func (mr *Reader) String() string {
	return "multi"
}

func (mr *Reader) Open() error {
	for _, reader := range mr.readers {
		if err := reader.Open(); err != nil {
			return err
		}
	}
	return nil
}

func (mr *Reader) Close() error {
	for _, reader := range mr.readers {
		if err := reader.Close(); err != nil {
			return err
		}
	}
	return nil
}

func (mr *Reader) ValidateStructure() []error {
	var ret []error
	for _, reader := range mr.readers {
		errs := reader.ValidateStructure()
		ret = append(ret, errs...)
	}
	return ret
}

func (mr *Reader) ReadEntities(c any) error {
	return nil
}

func (mr *Reader) StopTimesByTripID(ids ...string) chan []tl.StopTime {
	return readEntities(
		Map(mr.readers, func(r tl.Reader) chan []tl.StopTime { return r.StopTimesByTripID() }),
		nil,
	)
}

func (mr *Reader) Stops() chan tl.Stop {
	return readEntities(
		Map(mr.readers, func(r tl.Reader) chan tl.Stop { return r.Stops() }),
		func(e *tl.Stop, v int) { e.SetFeedVersionID(v) },
	)
}

func (mr *Reader) StopTimes() chan tl.StopTime {
	return readEntities(
		Map(mr.readers, func(r tl.Reader) chan tl.StopTime { return r.StopTimes() }),
		func(e *tl.StopTime, v int) { e.SetFeedVersionID(v) },
	)
}

func (mr *Reader) Agencies() chan tl.Agency {
	return readEntities(
		Map(mr.readers, func(r tl.Reader) chan tl.Agency { return r.Agencies() }),
		func(e *tl.Agency, v int) { e.SetFeedVersionID(v) },
	)
}

func (mr *Reader) Calendars() chan tl.Calendar {
	return readEntities(
		Map(mr.readers, func(r tl.Reader) chan tl.Calendar { return r.Calendars() }),
		func(e *tl.Calendar, v int) { e.SetFeedVersionID(v) },
	)
}

func (mr *Reader) CalendarDates() chan tl.CalendarDate {
	return readEntities(
		Map(mr.readers, func(r tl.Reader) chan tl.CalendarDate { return r.CalendarDates() }),
		func(e *tl.CalendarDate, v int) { e.SetFeedVersionID(v) },
	)
}

func (mr *Reader) FareAttributes() chan tl.FareAttribute {
	return readEntities(
		Map(mr.readers, func(r tl.Reader) chan tl.FareAttribute { return r.FareAttributes() }),
		func(e *tl.FareAttribute, v int) { e.SetFeedVersionID(v) },
	)
}

func (mr *Reader) FareRules() chan tl.FareRule {
	return readEntities(
		Map(mr.readers, func(r tl.Reader) chan tl.FareRule { return r.FareRules() }),
		func(e *tl.FareRule, v int) { e.SetFeedVersionID(v) },
	)
}

func (mr *Reader) FeedInfos() chan tl.FeedInfo {
	return readEntities(
		Map(mr.readers, func(r tl.Reader) chan tl.FeedInfo { return r.FeedInfos() }),
		func(e *tl.FeedInfo, v int) { e.SetFeedVersionID(v) },
	)
}

func (mr *Reader) Frequencies() chan tl.Frequency {
	return readEntities(
		Map(mr.readers, func(r tl.Reader) chan tl.Frequency { return r.Frequencies() }),
		func(e *tl.Frequency, v int) { e.SetFeedVersionID(v) },
	)
}

func (mr *Reader) Routes() chan tl.Route {
	return readEntities(
		Map(mr.readers, func(r tl.Reader) chan tl.Route { return r.Routes() }),
		func(e *tl.Route, v int) { e.SetFeedVersionID(v) },
	)
}

func (mr *Reader) Shapes() chan tl.Shape {
	return readEntities(
		Map(mr.readers, func(r tl.Reader) chan tl.Shape { return r.Shapes() }),
		func(e *tl.Shape, v int) { e.SetFeedVersionID(v) },
	)
}

func (mr *Reader) Transfers() chan tl.Transfer {
	return readEntities(
		Map(mr.readers, func(r tl.Reader) chan tl.Transfer { return r.Transfers() }),
		func(e *tl.Transfer, v int) { e.SetFeedVersionID(v) },
	)
}

func (mr *Reader) Pathways() chan tl.Pathway {
	return readEntities(
		Map(mr.readers, func(r tl.Reader) chan tl.Pathway { return r.Pathways() }),
		func(e *tl.Pathway, v int) { e.SetFeedVersionID(v) },
	)
}

func (mr *Reader) Levels() chan tl.Level {
	return readEntities(
		Map(mr.readers, func(r tl.Reader) chan tl.Level { return r.Levels() }),
		func(e *tl.Level, v int) { e.SetFeedVersionID(v) },
	)
}

func (mr *Reader) Trips() chan tl.Trip {
	return readEntities(
		Map(mr.readers, func(r tl.Reader) chan tl.Trip { return r.Trips() }),
		func(e *tl.Trip, v int) { e.SetFeedVersionID(v) },
	)
}

func (mr *Reader) Attributions() chan tl.Attribution {
	return readEntities(
		Map(mr.readers, func(r tl.Reader) chan tl.Attribution { return r.Attributions() }),
		func(e *tl.Attribution, v int) { e.SetFeedVersionID(v) },
	)
}

func (mr *Reader) Translations() chan tl.Translation {
	return readEntities(
		Map(mr.readers, func(r tl.Reader) chan tl.Translation { return r.Translations() }),
		func(e *tl.Translation, v int) { e.SetFeedVersionID(v) },
	)
}

func (mr *Reader) Areas() chan tl.Area {
	return readEntities(
		Map(mr.readers, func(r tl.Reader) chan tl.Area { return r.Areas() }),
		func(e *tl.Area, v int) { e.SetFeedVersionID(v) },
	)
}

func (mr *Reader) StopAreas() chan tl.StopArea {
	return readEntities(
		Map(mr.readers, func(r tl.Reader) chan tl.StopArea { return r.StopAreas() }),
		func(e *tl.StopArea, v int) { e.SetFeedVersionID(v) },
	)
}

func (mr *Reader) FareLegRules() chan tl.FareLegRule {
	return readEntities(
		Map(mr.readers, func(r tl.Reader) chan tl.FareLegRule { return r.FareLegRules() }),
		func(e *tl.FareLegRule, v int) { e.SetFeedVersionID(v) },
	)
}

func (mr *Reader) FareTransferRules() chan tl.FareTransferRule {
	return readEntities(
		Map(mr.readers, func(r tl.Reader) chan tl.FareTransferRule { return r.FareTransferRules() }),
		func(e *tl.FareTransferRule, v int) { e.SetFeedVersionID(v) },
	)
}

func (mr *Reader) FareContainers() chan tl.FareContainer {
	return readEntities(
		Map(mr.readers, func(r tl.Reader) chan tl.FareContainer { return r.FareContainers() }),
		func(e *tl.FareContainer, v int) { e.SetFeedVersionID(v) },
	)
}

func (mr *Reader) FareProducts() chan tl.FareProduct {
	return readEntities(
		Map(mr.readers, func(r tl.Reader) chan tl.FareProduct { return r.FareProducts() }),
		func(e *tl.FareProduct, v int) { e.SetFeedVersionID(v) },
	)
}

func (mr *Reader) RiderCategories() chan tl.RiderCategory {
	return readEntities(
		Map(mr.readers, func(r tl.Reader) chan tl.RiderCategory { return r.RiderCategories() }),
		func(e *tl.RiderCategory, v int) { e.SetFeedVersionID(v) },
	)
}

func readNullEntities[T any](reader *Reader) chan T {
	out := make(chan T, bufferSize)
	go func() {
		close(out)
	}()
	return out
}

func readEntities[T any](inchans []chan T, cb func(*T, int)) chan T {
	out := make(chan T, bufferSize)
	go func() {
		for fvid, inchan := range inchans {
			for ent := range inchan {
				if cb != nil {
					cb(&ent, fvid)
				}
				out <- ent
			}
		}
		close(out)
	}()
	return out
}

func Map[T, V any](ts []T, fn func(T) V) []V {
	result := make([]V, len(ts))
	for i, t := range ts {
		result[i] = fn(t)
	}
	return result
}
