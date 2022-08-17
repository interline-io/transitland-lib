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
	return []error{}
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
	return ReadEntities(mr, func(r tl.Reader) chan []tl.StopTime { return r.StopTimesByTripID(ids...) })
}

func (mr *Reader) Stops() chan tl.Stop {
	return ReadEntities(mr, func(r tl.Reader) chan tl.Stop { return r.Stops() })
}

func (mr *Reader) StopTimes() chan tl.StopTime {
	return ReadEntities(mr, func(r tl.Reader) chan tl.StopTime { return r.StopTimes() })
}

func (mr *Reader) Agencies() chan tl.Agency {
	return ReadEntities(mr, func(r tl.Reader) chan tl.Agency { return r.Agencies() })
}

func (mr *Reader) Calendars() chan tl.Calendar {
	return ReadEntities(mr, func(r tl.Reader) chan tl.Calendar { return r.Calendars() })
}

func (mr *Reader) CalendarDates() chan tl.CalendarDate {
	return ReadEntities(mr, func(r tl.Reader) chan tl.CalendarDate { return r.CalendarDates() })
}

func (mr *Reader) FareAttributes() chan tl.FareAttribute {
	return ReadEntities(mr, func(r tl.Reader) chan tl.FareAttribute { return r.FareAttributes() })
}

func (mr *Reader) FareRules() chan tl.FareRule {
	return ReadEntities(mr, func(r tl.Reader) chan tl.FareRule { return r.FareRules() })
}

func (mr *Reader) FeedInfos() chan tl.FeedInfo {
	return ReadEntities(mr, func(r tl.Reader) chan tl.FeedInfo { return r.FeedInfos() })
}

func (mr *Reader) Frequencies() chan tl.Frequency {
	return ReadEntities(mr, func(r tl.Reader) chan tl.Frequency { return r.Frequencies() })
}

func (mr *Reader) Routes() chan tl.Route {
	return ReadEntities(mr, func(r tl.Reader) chan tl.Route { return r.Routes() })
}

func (mr *Reader) Shapes() chan tl.Shape {
	return ReadEntities(mr, func(r tl.Reader) chan tl.Shape { return r.Shapes() })
}

func (mr *Reader) Transfers() chan tl.Transfer {
	return ReadEntities(mr, func(r tl.Reader) chan tl.Transfer { return r.Transfers() })
}

func (mr *Reader) Pathways() chan tl.Pathway {
	return ReadEntities(mr, func(r tl.Reader) chan tl.Pathway { return r.Pathways() })
}

func (mr *Reader) Levels() chan tl.Level {
	return ReadEntities(mr, func(r tl.Reader) chan tl.Level { return r.Levels() })
}

func (mr *Reader) Trips() chan tl.Trip {
	return ReadEntities(mr, func(r tl.Reader) chan tl.Trip { return r.Trips() })
}

func (mr *Reader) Attributions() chan tl.Attribution {
	return ReadEntities(mr, func(r tl.Reader) chan tl.Attribution { return r.Attributions() })
}

func (mr *Reader) Translations() chan tl.Translation {
	return ReadEntities(mr, func(r tl.Reader) chan tl.Translation { return r.Translations() })
}

func (mr *Reader) Areas() chan tl.Area {
	return ReadEntities(mr, func(r tl.Reader) chan tl.Area { return r.Areas() })
}

func (mr *Reader) StopAreas() chan tl.StopArea {
	return ReadEntities(mr, func(r tl.Reader) chan tl.StopArea { return r.StopAreas() })
}

func (mr *Reader) FareLegRules() chan tl.FareLegRule {
	return ReadEntities(mr, func(r tl.Reader) chan tl.FareLegRule { return r.FareLegRules() })
}

func (mr *Reader) FareTransferRules() chan tl.FareTransferRule {
	return ReadEntities(mr, func(r tl.Reader) chan tl.FareTransferRule { return r.FareTransferRules() })
}

func (mr *Reader) FareContainers() chan tl.FareContainer {
	return ReadEntities(mr, func(r tl.Reader) chan tl.FareContainer { return r.FareContainers() })
}

func (mr *Reader) FareProducts() chan tl.FareProduct {
	return ReadEntities(mr, func(r tl.Reader) chan tl.FareProduct { return r.FareProducts() })
}

func (mr *Reader) RiderCategories() chan tl.RiderCategory {
	return ReadEntities(mr, func(r tl.Reader) chan tl.RiderCategory { return r.RiderCategories() })
}

func ReadEntities[T any](reader *Reader, cf func(r tl.Reader) chan T) chan T {
	out := make(chan T, bufferSize)

	go func() {
		for _, r := range reader.Readers {
			readin := cf(r)
			for ent := range readin {
				out <- ent
			}
		}
		close(out)
	}()
	return out
}
