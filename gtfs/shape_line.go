package gtfs

import (
	"sort"

	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/tt"
)

// Shape shapes.txt
type ShapeLine struct {
	ShapeID   tt.String     `csv:",required"`
	Geometry  tt.LineString `db:"geometry" csv:"-"`
	Generated bool          `db:"generated" csv:"-"`
	tt.BaseEntity
}

// EntityID returns the ID or ShapeID.
func (ent *ShapeLine) EntityID() string {
	return entID(ent.ID, ent.ShapeID.Val)
}

// EntityKey returns the GTFS identifier.
func (ent *ShapeLine) EntityKey() string {
	return ent.ShapeID.Val
}

// NewShapeFromShapes takes Shapes with single points and returns a Shape with linestring geometry.
// Any errors from the input errors, or errors such as duplicate sequences, are added as entity errors.
func NewShapeLineFromShapes(shapes []Shape) ShapeLine {
	ent := ShapeLine{}
	coords := []float64{}
	sort.Slice(shapes, func(i, j int) bool {
		return shapes[i].ShapePtSequence.Val < shapes[j].ShapePtSequence.Val
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
		coords = append(coords, shape.ShapePtLon.Val, shape.ShapePtLat.Val, shape.ShapeDistTraveled.Val)
		for _, err := range shape.LoadErrors() {
			ent.AddError(err)
		}
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

// ValidateShapes returns errors for an array of shapes.
func ValidateShapes(shapes []Shape) []error {
	errs := []error{}
	last := -1
	dist := -1.0
	for _, shape := range shapes {
		// Check for duplicate ID errors
		if shape.ShapePtSequence.Int() == last {
			errs = append(errs, causes.NewSequenceError("shape_pt_sequence", tt.TryCsv(last)))
		}
		last = shape.ShapePtSequence.Int()
		if shape.ShapeDistTraveled.Val < dist {
			errs = append(errs, causes.NewSequenceError("shape_dist_traveled", tt.TryCsv(shape.ShapeDistTraveled)))
		} else if shape.ShapeDistTraveled.Val > 0 {
			dist = shape.ShapeDistTraveled.Val
		}
	}
	return errs
}

func FlattenShape(ent ShapeLine) []Shape {
	coords := ent.Geometry.FlatCoords()
	shapes := []Shape{}
	totaldist := 0.0
	for i := 0; i < len(coords); i += 3 {
		s := Shape{
			ShapeID:           ent.ShapeID,
			ShapePtSequence:   tt.NewInt(i),
			ShapePtLon:        tt.NewFloat(coords[i]),
			ShapePtLat:        tt.NewFloat(coords[i+1]),
			ShapeDistTraveled: tt.NewFloat(coords[i+2]),
		}
		totaldist += coords[i+2]
		shapes = append(shapes, s)
	}
	cur := 0.0
	for i := 0; i < len(shapes); i++ {
		if shapes[i].ShapeDistTraveled.Val < cur {
			shapes[i].ShapeDistTraveled.Unset()
		}
		cur = shapes[i].ShapeDistTraveled.Val
	}
	if cur == 0.0 {
		for i := 0; i < len(shapes); i++ {
			shapes[i].ShapeDistTraveled.Unset()
		}
	}
	return shapes
}
