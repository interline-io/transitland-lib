package multi

import (
	"fmt"

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
	return readNullEntities[[]tl.StopTime](mr)
}

func (mr *Reader) Stops() chan tl.Stop {
	return readNullEntities[tl.Stop](mr)
}

func (mr *Reader) StopTimes() chan tl.StopTime {
	return readNullEntities[tl.StopTime](mr)
}

func (mr *Reader) Agencies() chan tl.Agency {
	out := make(chan tl.Agency, bufferSize)
	go func() {
		for _, reader := range mr.readers {
			fmt.Println("agencies:", reader.String())
			for ent := range reader.Agencies() {
				fmt.Println("ent", ent)
				out <- ent
			}
		}
		close(out)
	}()
	return out
}

func (mr *Reader) Calendars() chan tl.Calendar {
	return readNullEntities[tl.Calendar](mr)
}

func (mr *Reader) CalendarDates() chan tl.CalendarDate {
	return readNullEntities[tl.CalendarDate](mr)
}

func (mr *Reader) FareAttributes() chan tl.FareAttribute {
	return readNullEntities[tl.FareAttribute](mr)
}

func (mr *Reader) FareRules() chan tl.FareRule {
	return readNullEntities[tl.FareRule](mr)
}

func (mr *Reader) FeedInfos() chan tl.FeedInfo {
	return readNullEntities[tl.FeedInfo](mr)
}

func (mr *Reader) Frequencies() chan tl.Frequency {
	return readNullEntities[tl.Frequency](mr)
}

func (mr *Reader) Routes() chan tl.Route {
	fmt.Println("routes")
	return readNullEntities[tl.Route](mr)
}

func (mr *Reader) Shapes() chan tl.Shape {
	return readNullEntities[tl.Shape](mr)
}

func (mr *Reader) Transfers() chan tl.Transfer {
	return readNullEntities[tl.Transfer](mr)
}

func (mr *Reader) Pathways() chan tl.Pathway {
	return readNullEntities[tl.Pathway](mr)
}

func (mr *Reader) Levels() chan tl.Level {
	return readNullEntities[tl.Level](mr)
}

func (mr *Reader) Trips() chan tl.Trip {
	return readNullEntities[tl.Trip](mr)
}

func (mr *Reader) Attributions() chan tl.Attribution {
	return readNullEntities[tl.Attribution](mr)
}

func (mr *Reader) Translations() chan tl.Translation {
	return readNullEntities[tl.Translation](mr)
}

func (mr *Reader) Areas() chan tl.Area {
	return readNullEntities[tl.Area](mr)
}

func (mr *Reader) StopAreas() chan tl.StopArea {
	return readNullEntities[tl.StopArea](mr)
}

func (mr *Reader) FareLegRules() chan tl.FareLegRule {
	return readNullEntities[tl.FareLegRule](mr)
}

func (mr *Reader) FareTransferRules() chan tl.FareTransferRule {
	return readNullEntities[tl.FareTransferRule](mr)
}

func (mr *Reader) FareContainers() chan tl.FareContainer {
	return readNullEntities[tl.FareContainer](mr)
}

func (mr *Reader) FareProducts() chan tl.FareProduct {
	return readNullEntities[tl.FareProduct](mr)
}

func (mr *Reader) RiderCategories() chan tl.RiderCategory {
	return readNullEntities[tl.RiderCategory](mr)
}

func readNullEntities[T any](reader *Reader) chan T {
	out := make(chan T, bufferSize)
	go func() {
		close(out)
	}()
	return out
}
