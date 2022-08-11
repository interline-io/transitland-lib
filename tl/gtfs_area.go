package tl

import (
	"github.com/interline-io/transitland-lib/tl/tt"
)

// Area fare_areas.txt
type Area struct {
	AreaID    string
	AreaName  string
	AgencyIDs []string   `csv:"-"` // interline ext
	Geometry  tt.Polygon `csv:"-"` // interline ext
	BaseEntity
}

func (ent *Area) Filename() string {
	return "areas.txt"
}

func (ent *Area) TableName() string {
	return "ext_faresv2_areas"
}

func (ent *Area) Errors() (errs []error) {
	errs = append(errs, ent.BaseEntity.Errors()...)
	errs = append(errs, tt.CheckPresent("area_id", ent.AreaID)...)
	return errs
}
