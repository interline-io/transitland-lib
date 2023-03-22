package direct

import (
	"reflect"

	"github.com/interline-io/transitland-lib/tl"
)

var bufferSize = 1000

// Reader is a mocked up Reader used for testing.
type Reader struct {
	AgencyList           []tl.Agency
	RouteList            []tl.Route
	TripList             []tl.Trip
	StopList             []tl.Stop
	StopTimeList         []tl.StopTime
	ShapeList            []tl.Shape
	CalendarList         []tl.Calendar
	CalendarDateList     []tl.CalendarDate
	FeedInfoList         []tl.FeedInfo
	FareRuleList         []tl.FareRule
	FareAttributeList    []tl.FareAttribute
	FrequencyList        []tl.Frequency
	TransferList         []tl.Transfer
	LevelList            []tl.Level
	PathwayList          []tl.Pathway
	AttributionList      []tl.Attribution
	TranslationList      []tl.Translation
	AreaList             []tl.Area
	StopAreaList         []tl.StopArea
	FareLegRuleList      []tl.FareLegRule
	FareTransferRuleList []tl.FareTransferRule
	FareMediaList        []tl.FareMedia
	FareProductList      []tl.FareProduct
	RiderCategoryList    []tl.RiderCategory
	OtherList            []tl.Entity
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
func (mr *Reader) StopTimesByTripID(...string) chan []tl.StopTime {
	out := make(chan []tl.StopTime, 1000)
	go func() {
		sts := map[string][]tl.StopTime{}
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
func (mr *Reader) ShapesByShapeID(...string) chan []tl.Shape {
	c := make(chan []tl.Shape, 1000)
	close(c)
	return c
}

// ShapeLinesByShapeID .
func (mr *Reader) ShapeLinesByShapeID(...string) chan tl.Shape {
	out := make(chan tl.Shape, 1000)
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
func (mr *Reader) Stops() chan tl.Stop {
	out := make(chan tl.Stop, bufferSize)
	go func() {
		for _, ent := range mr.StopList {
			out <- ent
		}
		close(out)
	}()
	return out
}

// StopTimes .
func (mr *Reader) StopTimes() chan tl.StopTime {
	out := make(chan tl.StopTime, bufferSize)
	go func() {
		for _, ent := range mr.StopTimeList {
			out <- ent
		}
		close(out)
	}()
	return out
}

// Agencies .
func (mr *Reader) Agencies() chan tl.Agency {
	out := make(chan tl.Agency, bufferSize)
	go func() {
		for _, ent := range mr.AgencyList {
			out <- ent
		}
		close(out)
	}()
	return out
}

// Calendars .
func (mr *Reader) Calendars() chan tl.Calendar {
	out := make(chan tl.Calendar, bufferSize)
	go func() {
		for _, ent := range mr.CalendarList {
			out <- ent
		}
		close(out)
	}()
	return out
}

// CalendarDates .
func (mr *Reader) CalendarDates() chan tl.CalendarDate {
	out := make(chan tl.CalendarDate, bufferSize)
	go func() {
		for _, ent := range mr.CalendarDateList {
			out <- ent
		}
		close(out)
	}()
	return out
}

// FareAttributes .
func (mr *Reader) FareAttributes() chan tl.FareAttribute {
	out := make(chan tl.FareAttribute, bufferSize)
	go func() {
		for _, ent := range mr.FareAttributeList {
			out <- ent
		}
		close(out)
	}()
	return out
}

// FareRules .
func (mr *Reader) FareRules() chan tl.FareRule {
	out := make(chan tl.FareRule, bufferSize)
	go func() {
		for _, ent := range mr.FareRuleList {
			out <- ent
		}
		close(out)
	}()
	return out
}

// FeedInfos .
func (mr *Reader) FeedInfos() chan tl.FeedInfo {
	out := make(chan tl.FeedInfo, bufferSize)
	go func() {
		for _, ent := range mr.FeedInfoList {
			out <- ent
		}
		close(out)
	}()
	return out
}

// Frequencies .
func (mr *Reader) Frequencies() chan tl.Frequency {
	out := make(chan tl.Frequency, bufferSize)
	go func() {
		for _, ent := range mr.FrequencyList {
			out <- ent
		}
		close(out)
	}()
	return out
}

// Routes .
func (mr *Reader) Routes() chan tl.Route {
	out := make(chan tl.Route, bufferSize)
	go func() {
		for _, ent := range mr.RouteList {
			out <- ent
		}
		close(out)
	}()
	return out
}

// Shapes .
func (mr *Reader) Shapes() chan tl.Shape {
	out := make(chan tl.Shape, bufferSize)
	go func() {
		for _, ent := range mr.ShapeList {
			out <- ent
		}
		close(out)
	}()
	return out
}

// Transfers .
func (mr *Reader) Transfers() chan tl.Transfer {
	out := make(chan tl.Transfer, bufferSize)
	go func() {
		for _, ent := range mr.TransferList {
			out <- ent
		}
		close(out)
	}()
	return out
}

// Pathways .
func (mr *Reader) Pathways() chan tl.Pathway {
	out := make(chan tl.Pathway, bufferSize)
	go func() {
		for _, ent := range mr.PathwayList {
			out <- ent
		}
		close(out)
	}()
	return out
}

// Levels .
func (mr *Reader) Levels() chan tl.Level {
	out := make(chan tl.Level, bufferSize)
	go func() {
		for _, ent := range mr.LevelList {
			out <- ent
		}
		close(out)
	}()
	return out
}

// Trips .
func (mr *Reader) Trips() chan tl.Trip {
	out := make(chan tl.Trip, bufferSize)
	go func() {
		for _, ent := range mr.TripList {
			out <- ent
		}
		close(out)
	}()
	return out
}

// Attributions .
func (mr *Reader) Attributions() chan tl.Attribution {
	out := make(chan tl.Attribution, bufferSize)
	go func() {
		for _, ent := range mr.AttributionList {
			out <- ent
		}
		close(out)
	}()
	return out
}

// Translations .
func (mr *Reader) Translations() chan tl.Translation {
	out := make(chan tl.Translation, bufferSize)
	go func() {
		for _, ent := range mr.TranslationList {
			out <- ent
		}
		close(out)
	}()
	return out
}

func (mr *Reader) Areas() chan tl.Area {
	out := make(chan tl.Area, bufferSize)
	go func() {
		for _, ent := range mr.AreaList {
			out <- ent
		}
		close(out)
	}()
	return out
}

func (mr *Reader) StopAreas() chan tl.StopArea {
	out := make(chan tl.StopArea, bufferSize)
	go func() {
		for _, ent := range mr.StopAreaList {
			out <- ent
		}
		close(out)
	}()
	return out
}

func (mr *Reader) FareLegRules() chan tl.FareLegRule {
	out := make(chan tl.FareLegRule, bufferSize)
	go func() {
		for _, ent := range mr.FareLegRuleList {
			out <- ent
		}
		close(out)
	}()
	return out
}

func (mr *Reader) FareTransferRules() chan tl.FareTransferRule {
	out := make(chan tl.FareTransferRule, bufferSize)
	go func() {
		for _, ent := range mr.FareTransferRuleList {
			out <- ent
		}
		close(out)
	}()
	return out
}

func (mr *Reader) FareMedia() chan tl.FareMedia {
	out := make(chan tl.FareMedia, bufferSize)
	go func() {
		for _, ent := range mr.FareMediaList {
			out <- ent
		}
		close(out)
	}()
	return out
}

func (mr *Reader) FareProducts() chan tl.FareProduct {
	out := make(chan tl.FareProduct, bufferSize)
	go func() {
		for _, ent := range mr.FareProductList {
			out <- ent
		}
		close(out)
	}()
	return out
}

func (mr *Reader) RiderCategories() chan tl.RiderCategory {
	out := make(chan tl.RiderCategory, bufferSize)
	go func() {
		for _, ent := range mr.RiderCategoryList {
			out <- ent
		}
		close(out)
	}()
	return out
}
