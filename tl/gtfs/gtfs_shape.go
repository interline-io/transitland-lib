package gtfs

import (
	"sort"
	"strconv"

	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/interline-io/transitland-lib/tl/tt"
)

// Shape shapes.txt
type Shape struct {
	ShapeID           string        `csv:",required"`
	ShapePtLat        float64       `db:"-" csv:",required"`
	ShapePtLon        float64       `db:"-" csv:",required"`
	ShapePtSequence   int           `db:"-" csv:",required"`
	ShapeDistTraveled float64       `db:"-"`
	Geometry          tt.LineString `db:"geometry" csv:"-"`
	Generated         bool          `db:"generated" csv:"-"`
	tt.BaseEntity
}

// ValidateShapes returns errors for an array of shapes.
func ValidateShapes(shapes []Shape) []error {
	errs := []error{}
	last := -1
	dist := -1.0
	for _, shape := range shapes {
		// Check for duplicate ID errors
		if shape.ShapePtSequence == last {
			errs = append(errs, causes.NewSequenceError("shape_pt_sequence", tt.TryCsv(last)))
		}
		last = shape.ShapePtSequence
		if shape.ShapeDistTraveled < dist {
			errs = append(errs, causes.NewSequenceError("shape_dist_traveled", tt.TryCsv(shape.ShapeDistTraveled)))
		} else if shape.ShapeDistTraveled > 0 {
			dist = shape.ShapeDistTraveled
		}
	}
	return errs
}

// NewShapeFromShapes takes Shapes with single points and returns a Shape with linestring geometry.
// Any errors from the input errors, or errors such as duplicate sequences, are added as entity errors.
func NewShapeFromShapes(shapes []Shape) Shape {
	ent := Shape{}
	coords := []float64{}
	sort.Slice(shapes, func(i, j int) bool {
		return shapes[i].ShapePtSequence < shapes[j].ShapePtSequence
	})
	// Get sequence errors
	if errs := ValidateShapes(shapes); len(errs) > 0 {
		for _, err := range errs {
			ent.AddError(err)
		}
	}
	// expectError is just for validation tests.
	// Add to coords, add base errors
	for _, shape := range shapes {
		coords = append(coords, shape.ShapePtLon, shape.ShapePtLat, shape.ShapeDistTraveled)
		for _, err := range shape.Errors() {
			ent.AddError(err)
		}
		// For tests...
		if v, ok := shape.GetExtra("expect_error"); len(v) > 0 && ok {
			ent.SetExtra("expect_error", v)
		}
	}
	ent.Geometry = tt.NewLineStringFromFlatCoords(coords)
	return ent
}

// EntityID returns the ID or ShapeID.
func (ent *Shape) EntityID() string {
	return entID(ent.ID, ent.ShapeID)
}

// EntityKey returns the GTFS identifier.
func (ent *Shape) EntityKey() string {
	return ent.ShapeID
}

// Errors for this Entity.
func (ent *Shape) Errors() (errs []error) {
	errs = append(errs, ent.BaseEntity.Errors()...)
	errs = append(errs, tt.CheckPresent("shape_id", ent.ShapeID)...)
	errs = append(errs, tt.CheckInsideRange("shape_pt_lat", ent.ShapePtLat, -90.0, 90.0)...)
	errs = append(errs, tt.CheckInsideRange("shape_pt_lon", ent.ShapePtLon, -180.0, 180.0)...)
	errs = append(errs, tt.CheckPositiveInt("shape_pt_sequence", ent.ShapePtSequence)...)
	errs = append(errs, tt.CheckPositive("shape_dist_traveled", ent.ShapeDistTraveled)...)
	return errs
}

// Filename shapes.txt
func (ent *Shape) Filename() string {
	return "shapes.txt"
}

// TableName gtfs_shapes
func (ent *Shape) TableName() string {
	return "gtfs_shapes"
}

// SetString provides a fast, non-reflect loading path.
func (ent *Shape) SetString(key string, value string) error {
	var perr error
	switch key {
	case "shape_id":
		ent.ShapeID = value
	case "shape_dist_traveled":
		if len(value) == 0 {
			ent.ShapeDistTraveled = 0.0
		} else if a, err := strconv.ParseFloat(value, 64); err != nil {
			perr = causes.NewFieldParseError("shape_dist_traveled", value)
		} else {
			ent.ShapeDistTraveled = a
		}
	case "shape_pt_lon":
		if len(value) == 0 {
			perr = causes.NewRequiredFieldError("shape_pt_lon")
		} else if a, err := strconv.ParseFloat(value, 64); err != nil {
			perr = causes.NewFieldParseError("shape_pt_lon", value)
		} else {
			ent.ShapePtLon = a
		}
	case "shape_pt_lat":
		if len(value) == 0 {
			perr = causes.NewRequiredFieldError("shape_pt_lat")
		} else if a, err := strconv.ParseFloat(value, 64); err != nil {
			perr = causes.NewFieldParseError("shape_pt_lat", value)
		} else {
			ent.ShapePtLat = a
		}
	case "shape_pt_sequence":
		if len(value) == 0 {
			perr = causes.NewRequiredFieldError("shape_pt_sequence")
		} else if a, err := strconv.Atoi(value); err != nil {
			perr = causes.NewFieldParseError("shape_pt_sequence", value)
		} else {
			ent.ShapePtSequence = a
		}
	default:
		ent.SetExtra(key, value)
	}
	return perr
}
