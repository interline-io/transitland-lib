package direct

import (
	"reflect"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

var bufferSize = 1000

// Reader is a mocked up Reader used for testing.
type Reader struct {
	AgencyList           []gtfs.Agency
	RouteList            []gtfs.Route
	TripList             []gtfs.Trip
	StopList             []gtfs.Stop
	StopTimeList         []gtfs.StopTime
	ShapeList            []gtfs.Shape
	CalendarList         []gtfs.Calendar
	CalendarDateList     []gtfs.CalendarDate
	FeedInfoList         []gtfs.FeedInfo
	FareRuleList         []gtfs.FareRule
	FareAttributeList    []gtfs.FareAttribute
	FrequencyList        []gtfs.Frequency
	TransferList         []gtfs.Transfer
	LevelList            []gtfs.Level
	PathwayList          []gtfs.Pathway
	AttributionList      []gtfs.Attribution
	TranslationList      []gtfs.Translation
	AreaList             []gtfs.Area
	StopAreaList         []gtfs.StopArea
	FareLegRuleList      []gtfs.FareLegRule
	FareTransferRuleList []gtfs.FareTransferRule
	FareMediaList        []gtfs.FareMedia
	FareProductList      []gtfs.FareProduct
	RiderCategoryList    []gtfs.RiderCategory
	OtherList            []tt.Entity
}

// NewReader returns a new Reader.
func NewReader() *Reader {
	return &Reader{}
}

func (mr *Reader) String() string {
	return "mock"
}

// Open .
func (mr *Reader) Open() error {
	return nil
}

// Close .
func (mr *Reader) Close() error {
	return nil
}

// ValidateStructure .
func (mr *Reader) ValidateStructure() []error {
	return []error{}
}

// StopTimesByTripID .
func (mr *Reader) StopTimesByTripID(...string) chan []gtfs.StopTime {
	out := make(chan []gtfs.StopTime, 1000)
	go func() {
		sts := map[string][]gtfs.StopTime{}
		for _, ent := range mr.StopTimeList {
			sts[ent.TripID] = append(sts[ent.TripID], ent)
		}
		for _, v := range sts {
			out <- v
		}
		close(out)
	}()
	return out
}

// ShapesByShapeID .
func (mr *Reader) ShapesByShapeID(...string) chan []gtfs.Shape {
	c := make(chan []gtfs.Shape, 1000)
	close(c)
	return c
}

// ShapeLinesByShapeID .
func (mr *Reader) ShapeLinesByShapeID(...string) chan gtfs.Shape {
	out := make(chan gtfs.Shape, 1000)
	go func() {
		for _, ent := range mr.ShapeList {
			out <- ent
		}
		close(out)
	}()
	return out
}

// ReadEntities - Only StopTimes are supported; just for passing tests!!
func (mr *Reader) ReadEntities(c interface{}) error {
	outValue := reflect.ValueOf(c)
	for _, ent := range mr.StopTimeList {
		outValue.Send(reflect.ValueOf(ent))
	}
	outValue.Close()
	return nil
}

// Stops .
func (mr *Reader) Stops() chan gtfs.Stop {
	out := make(chan gtfs.Stop, bufferSize)
	go func() {
		for _, ent := range mr.StopList {
			out <- ent
		}
		close(out)
	}()
	return out
}

// StopTimes .
func (mr *Reader) StopTimes() chan gtfs.StopTime {
	out := make(chan gtfs.StopTime, bufferSize)
	go func() {
		for _, ent := range mr.StopTimeList {
			out <- ent
		}
		close(out)
	}()
	return out
}

// Agencies .
func (mr *Reader) Agencies() chan gtfs.Agency {
	out := make(chan gtfs.Agency, bufferSize)
	go func() {
		for _, ent := range mr.AgencyList {
			out <- ent
		}
		close(out)
	}()
	return out
}

// Calendars .
func (mr *Reader) Calendars() chan gtfs.Calendar {
	out := make(chan gtfs.Calendar, bufferSize)
	go func() {
		for _, ent := range mr.CalendarList {
			out <- ent
		}
		close(out)
	}()
	return out
}

// CalendarDates .
func (mr *Reader) CalendarDates() chan gtfs.CalendarDate {
	out := make(chan gtfs.CalendarDate, bufferSize)
	go func() {
		for _, ent := range mr.CalendarDateList {
			out <- ent
		}
		close(out)
	}()
	return out
}

// FareAttributes .
func (mr *Reader) FareAttributes() chan gtfs.FareAttribute {
	out := make(chan gtfs.FareAttribute, bufferSize)
	go func() {
		for _, ent := range mr.FareAttributeList {
			out <- ent
		}
		close(out)
	}()
	return out
}

// FareRules .
func (mr *Reader) FareRules() chan gtfs.FareRule {
	out := make(chan gtfs.FareRule, bufferSize)
	go func() {
		for _, ent := range mr.FareRuleList {
			out <- ent
		}
		close(out)
	}()
	return out
}

// FeedInfos .
func (mr *Reader) FeedInfos() chan gtfs.FeedInfo {
	out := make(chan gtfs.FeedInfo, bufferSize)
	go func() {
		for _, ent := range mr.FeedInfoList {
			out <- ent
		}
		close(out)
	}()
	return out
}

// Frequencies .
func (mr *Reader) Frequencies() chan gtfs.Frequency {
	out := make(chan gtfs.Frequency, bufferSize)
	go func() {
		for _, ent := range mr.FrequencyList {
			out <- ent
		}
		close(out)
	}()
	return out
}

// Routes .
func (mr *Reader) Routes() chan gtfs.Route {
	out := make(chan gtfs.Route, bufferSize)
	go func() {
		for _, ent := range mr.RouteList {
			out <- ent
		}
		close(out)
	}()
	return out
}

// Shapes .
func (mr *Reader) Shapes() chan gtfs.Shape {
	out := make(chan gtfs.Shape, bufferSize)
	go func() {
		for _, ent := range mr.ShapeList {
			out <- ent
		}
		close(out)
	}()
	return out
}

// Transfers .
func (mr *Reader) Transfers() chan gtfs.Transfer {
	out := make(chan gtfs.Transfer, bufferSize)
	go func() {
		for _, ent := range mr.TransferList {
			out <- ent
		}
		close(out)
	}()
	return out
}

// Pathways .
func (mr *Reader) Pathways() chan gtfs.Pathway {
	out := make(chan gtfs.Pathway, bufferSize)
	go func() {
		for _, ent := range mr.PathwayList {
			out <- ent
		}
		close(out)
	}()
	return out
}

// Levels .
func (mr *Reader) Levels() chan gtfs.Level {
	out := make(chan gtfs.Level, bufferSize)
	go func() {
		for _, ent := range mr.LevelList {
			out <- ent
		}
		close(out)
	}()
	return out
}

// Trips .
func (mr *Reader) Trips() chan gtfs.Trip {
	out := make(chan gtfs.Trip, bufferSize)
	go func() {
		for _, ent := range mr.TripList {
			out <- ent
		}
		close(out)
	}()
	return out
}

// Attributions .
func (mr *Reader) Attributions() chan gtfs.Attribution {
	out := make(chan gtfs.Attribution, bufferSize)
	go func() {
		for _, ent := range mr.AttributionList {
			out <- ent
		}
		close(out)
	}()
	return out
}

// Translations .
func (mr *Reader) Translations() chan gtfs.Translation {
	out := make(chan gtfs.Translation, bufferSize)
	go func() {
		for _, ent := range mr.TranslationList {
			out <- ent
		}
		close(out)
	}()
	return out
}

func (mr *Reader) Areas() chan gtfs.Area {
	out := make(chan gtfs.Area, bufferSize)
	go func() {
		for _, ent := range mr.AreaList {
			out <- ent
		}
		close(out)
	}()
	return out
}

func (mr *Reader) StopAreas() chan gtfs.StopArea {
	out := make(chan gtfs.StopArea, bufferSize)
	go func() {
		for _, ent := range mr.StopAreaList {
			out <- ent
		}
		close(out)
	}()
	return out
}

func (mr *Reader) FareLegRules() chan gtfs.FareLegRule {
	out := make(chan gtfs.FareLegRule, bufferSize)
	go func() {
		for _, ent := range mr.FareLegRuleList {
			out <- ent
		}
		close(out)
	}()
	return out
}

func (mr *Reader) FareTransferRules() chan gtfs.FareTransferRule {
	out := make(chan gtfs.FareTransferRule, bufferSize)
	go func() {
		for _, ent := range mr.FareTransferRuleList {
			out <- ent
		}
		close(out)
	}()
	return out
}

func (mr *Reader) FareMedia() chan gtfs.FareMedia {
	out := make(chan gtfs.FareMedia, bufferSize)
	go func() {
		for _, ent := range mr.FareMediaList {
			out <- ent
		}
		close(out)
	}()
	return out
}

func (mr *Reader) FareProducts() chan gtfs.FareProduct {
	out := make(chan gtfs.FareProduct, bufferSize)
	go func() {
		for _, ent := range mr.FareProductList {
			out <- ent
		}
		close(out)
	}()
	return out
}

func (mr *Reader) RiderCategories() chan gtfs.RiderCategory {
	out := make(chan gtfs.RiderCategory, bufferSize)
	go func() {
		for _, ent := range mr.RiderCategoryList {
			out <- ent
		}
		close(out)
	}()
	return out
}
