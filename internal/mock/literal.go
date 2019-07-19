package mock

import (
	"fmt"
	"strings"
	"time"

	"github.com/interline-io/gotransit"
)

// NewOptionalTime supports ReaderLiteral.
func NewOptionalTime(isostring string) gotransit.OptionalTime {
	t := gotransit.OptionalTime{}
	err := t.Time.UnmarshalText([]byte(isostring))
	if err != nil {
		t.Valid = true
	}
	return t
}

// NewTime supports ReaderLiteral.
func NewTime(isostring string) time.Time {
	t := time.Time{}
	err := t.UnmarshalText([]byte(isostring))
	_ = err
	return t
}

// NewPoint supports ReaderLiteral.
func NewPoint(lon, lat float64) *gotransit.Point {
	return gotransit.NewPoint(lon, lat)
}

// NewLineStringFromFlatCoords supports ReaderLiteral.
func NewLineStringFromFlatCoords(coords []float64) *gotransit.LineString {
	return gotransit.NewLineStringFromFlatCoords(coords)

}

// ReaderLiteral takes a Reader and creates a Go literal for a Reader.
func ReaderLiteral(reader gotransit.Reader) (string, error) {
	baseent := gotransit.BaseEntity{}
	rep := []string{}
	rep = append(rep, fmt.Sprintf(", BaseEntity:%#v", baseent), "")
	mr := &Reader{}
	for ent := range reader.Agencies() {
		ent.BaseEntity = baseent
		mr.AgencyList = append(mr.AgencyList, ent)
	}
	for ent := range reader.Routes() {
		ent.BaseEntity = baseent
		mr.RouteList = append(mr.RouteList, ent)
	}
	for ent := range reader.Trips() {
		ent.BaseEntity = baseent
		mr.TripList = append(mr.TripList, ent)
	}
	for ent := range reader.Stops() {
		ent.BaseEntity = baseent
		s := fmt.Sprintf("(%T)(%p)", ent.Geometry, ent.Geometry)
		rep = append(rep, s, fmt.Sprintf("NewPoint(%0.5f, %0.5f)", ent.Geometry.X(), ent.Geometry.Y()))
		mr.StopList = append(mr.StopList, ent)
	}
	for ent := range reader.StopTimes() {
		ent.BaseEntity = baseent
		mr.StopTimeList = append(mr.StopTimeList, ent)
	}
	for ent := range reader.ShapeLinesByShapeID() {
		ent.BaseEntity = baseent
		coords := []string{}
		for _, c := range ent.Geometry.FlatCoords() {
			coords = append(coords, fmt.Sprintf("%0.5f", c))
		}
		rep = append(rep, fmt.Sprintf("(%T)(%p)", ent.Geometry, ent.Geometry), fmt.Sprintf("NewLineStringFromFlatCoords([]float64{%s})", strings.Join(coords, ",")))
		mr.ShapeList = append(mr.ShapeList, ent)
	}
	for ent := range reader.Calendars() {
		ent.BaseEntity = baseent
		a, _ := ent.StartDate.MarshalText()
		b, _ := ent.EndDate.MarshalText()
		rep = append(rep, fmt.Sprintf("%#v", ent.StartDate), fmt.Sprintf("NewTime(\"%s\")", a))
		rep = append(rep, fmt.Sprintf("%#v", ent.EndDate), fmt.Sprintf("NewTime(\"%s\")", b))
		mr.CalendarList = append(mr.CalendarList, ent)
	}
	for ent := range reader.CalendarDates() {
		ent.BaseEntity = baseent
		a, _ := ent.Date.MarshalText()
		rep = append(rep, fmt.Sprintf("%#v", ent.Date), fmt.Sprintf("NewTime(\"%s\")", a))
		mr.CalendarDateList = append(mr.CalendarDateList, ent)
	}
	for ent := range reader.FeedInfos() {
		ent.BaseEntity = baseent
		a, _ := ent.FeedStartDate.Time.MarshalText()
		b, _ := ent.FeedEndDate.Time.MarshalText()
		rep = append(rep, fmt.Sprintf("%#v", ent.FeedStartDate), fmt.Sprintf("NewOptionalTime(\"%s\")", a))
		rep = append(rep, fmt.Sprintf("%#v", ent.FeedEndDate), fmt.Sprintf("NewOptionalTime(\"%s\")", b))
		mr.FeedInfoList = append(mr.FeedInfoList, ent)
	}
	for ent := range reader.FareRules() {
		ent.BaseEntity = baseent
		mr.FareRuleList = append(mr.FareRuleList, ent)
	}
	for ent := range reader.FareAttributes() {
		ent.BaseEntity = baseent
		mr.FareAttributeList = append(mr.FareAttributeList, ent)
	}
	for ent := range reader.Frequencies() {
		ent.BaseEntity = baseent
		mr.FrequencyList = append(mr.FrequencyList, ent)
	}
	for ent := range reader.Transfers() {
		ent.BaseEntity = baseent
		mr.TransferList = append(mr.TransferList, ent)
	}
	s := fmt.Sprintf("%#v\n", mr)
	for i := 0; i < len(rep); i += 2 {
		s = strings.Replace(s, rep[i], rep[i+1], -1)
	}
	return s, nil
}
