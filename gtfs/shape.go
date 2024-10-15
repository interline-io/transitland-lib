package gtfs

import (
	"strconv"

	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/tt"
)

// Shape shapes.txt
type Shape struct {
	ShapeID           tt.String     `csv:",required"`
	ShapePtLat        tt.Float      `db:"-" csv:",required" range:"-90,90"`
	ShapePtLon        tt.Float      `db:"-" csv:",required" range:"-180,180"`
	ShapePtSequence   tt.Int        `db:"-" csv:",required" range:"0,"`
	ShapeDistTraveled tt.Float      `db:"-" range:"0,"`
	Geometry          tt.LineString `db:"geometry" csv:"-"`
	Generated         bool          `db:"generated" csv:"-"`
	tt.BaseEntity
}

// EntityID returns the ID or ShapeID.
func (ent *Shape) EntityID() string {
	return entID(ent.ID, ent.ShapeID.Val)
}

// EntityKey returns the GTFS identifier.
func (ent *Shape) EntityKey() string {
	return ent.ShapeID.Val
}

// Filename shapes.txt
func (ent *Shape) Filename() string {
	return "shapes.txt"
}

// TableName gtfs_shapes
func (ent *Shape) TableName() string {
	return "gtfs_shapes"
}

// Errors for this Entity.
func (ent *Shape) Errors() (errs []error) {
	// Defer on moving this to reflect path for now
	errs = append(errs, tt.CheckPresent("shape_id", ent.ShapeID.Val)...)
	errs = append(errs, tt.CheckInsideRange("shape_pt_lat", ent.ShapePtLat.Val, -90.0, 90.0)...)
	errs = append(errs, tt.CheckInsideRange("shape_pt_lon", ent.ShapePtLon.Val, -180.0, 180.0)...)
	errs = append(errs, tt.CheckPositiveInt("shape_pt_sequence", ent.ShapePtSequence.Val)...)
	errs = append(errs, tt.CheckPositive("shape_dist_traveled", ent.ShapeDistTraveled.Val)...)
	return errs
}

// SetString provides a fast, non-reflect loading path.
func (ent *Shape) SetString(key string, value string) error {
	var perr error
	switch key {
	case "shape_id":
		ent.ShapeID.Set(value)
	case "shape_dist_traveled":
		if len(value) == 0 {
			// Leave unset
		} else if a, err := strconv.ParseFloat(value, 64); err != nil {
			perr = causes.NewFieldParseError("shape_dist_traveled", value)
		} else {
			ent.ShapeDistTraveled.Set(a)
		}
	case "shape_pt_lon":
		if len(value) == 0 {
			perr = causes.NewRequiredFieldError("shape_pt_lon")
		} else if a, err := strconv.ParseFloat(value, 64); err != nil {
			perr = causes.NewFieldParseError("shape_pt_lon", value)
		} else {
			ent.ShapePtLon.Set(a)
		}
	case "shape_pt_lat":
		if len(value) == 0 {
			perr = causes.NewRequiredFieldError("shape_pt_lat")
		} else if a, err := strconv.ParseFloat(value, 64); err != nil {
			perr = causes.NewFieldParseError("shape_pt_lat", value)
		} else {
			ent.ShapePtLat.Set(a)
		}
	case "shape_pt_sequence":
		if len(value) == 0 {
			perr = causes.NewRequiredFieldError("shape_pt_sequence")
		} else if a, err := strconv.Atoi(value); err != nil {
			perr = causes.NewFieldParseError("shape_pt_sequence", value)
		} else {
			ent.ShapePtSequence.SetInt(a)
		}
	default:
		ent.SetExtra(key, value)
	}
	return perr
}
