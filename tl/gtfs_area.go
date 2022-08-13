package tl

import (
	"github.com/interline-io/transitland-lib/tl/tt"
)

// Area fare_areas.txt
type Area struct {
	AreaID    String
	AreaName  String
	AgencyIDs Strings    `csv:"-"` // interline ext
	Geometry  tt.Polygon `csv:"-"` // interline ext
	BaseEntity
}

func (ent *Area) EntityID() string {
	return ent.AreaID.Val
}

func (ent *Area) Filename() string {
	return "areas.txt"
}

func (ent *Area) TableName() string {
	return "gtfs_areas"
}

func (ent *Area) Errors() (errs []error) {
	errs = append(errs, ent.BaseEntity.Errors()...)
	errs = append(errs, tt.CheckPresent("area_id", ent.AreaID.Val)...)
	return errs
}
