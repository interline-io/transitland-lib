package gotransit

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/interline-io/gotransit/causes"
)

// Shape shapes.txt
type Shape struct {
	ShapeID           string      `csv:"shape_id" required:"true" gorm:"index;not null"`
	ShapePtLat        float64     `csv:"shape_pt_lat" db:"-" required:"true" min:"-90" max:"90" gorm:"-"`
	ShapePtLon        float64     `csv:"shape_pt_lon" db:"-" required:"true" min:"-180" max:"180" gorm:"-"`
	ShapePtSequence   int         `csv:"shape_pt_sequence" db:"-" required:"true" min:"0" gorm:"-"`
	ShapeDistTraveled float64     `csv:"shape_dist_traveled" db:"-" min:"0" gorm:"-"`
	Geometry          *LineString `db:"geometry,insert=ST_GeomFromWKB(?@4326)"`
	Generated         bool
	BaseEntity
}

// ValidateShapes returns errors for an array of shapes.
func ValidateShapes(shapes []Shape) []error {
	errs := []error{}
	last := -1
	dist := -1.0
	for _, shape := range shapes {
		// Check for duplicate ID errors
		if shape.ShapePtSequence == last {
			errs = append(errs, causes.NewSequenceError("shape_pt_sequence", strconv.Itoa(last)))
		}
		last = shape.ShapePtSequence
		if shape.ShapeDistTraveled < dist {
			errs = append(errs, causes.NewSequenceError("shape_dist_traveled", fmt.Sprintf("%f", shape.ShapeDistTraveled)))
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
		if v, ok := shape.Extra()["expect_error"]; len(v) > 0 && ok {
			ent.SetExtra("expect_error", v)
		}
	}
	ent.Geometry = NewLineStringFromFlatCoords(coords)
	return ent
}

// EntityID returns the ID or ShapeID.
func (ent *Shape) EntityID() string {
	return entID(ent.ID, ent.ShapeID)
}

// Warnings for this Entity.
func (ent *Shape) Warnings() (errs []error) {
	coords := []float64{ent.ShapePtLon, ent.ShapePtLat}
	if ent.Geometry != nil {
		coords = ent.Geometry.FlatCoords()
	}
	if coords[0] == 0 {
		errs = append(errs, causes.NewValidationWarning("shape_pt_lon", "required field shape_pt_lon is 0.0"))
	}
	if coords[1] == 0 {
		errs = append(errs, causes.NewValidationWarning("shape_pt_lat", "required field shape_pt_lat is 0.0"))
	}
	return errs
}

// Errors for this Entity.
func (ent *Shape) Errors() (errs []error) {
	errs = ValidateTags(ent)
	errs = append(errs, ent.BaseEntity.loadErrors...)
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
