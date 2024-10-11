package tl

import (
	"errors"
	"sort"
	"time"

	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/interline-io/transitland-lib/tl/tt"
)

type canSHA1 interface {
	SHA1() (string, error)
}

type canDirSHA1 interface {
	DirSHA1() (string, error)
}

type canPath interface {
	Path() string
}

// NewFeedVersionFromReader returns a FeedVersion from a Reader.
func NewFeedVersionFromReader(reader Reader) (FeedVersion, error) {
	fv := FeedVersion{}
	// Perform basic GTFS validity checks
	if errs := reader.ValidateStructure(); len(errs) > 0 {
		return fv, errs[0]
	}
	// Get service dates
	if start, end, err := FeedVersionServiceBounds(reader); err == nil {
		fv.EarliestCalendarDate = tt.NewDate(start)
		fv.LatestCalendarDate = tt.NewDate(end)
	} else {
		return fv, err
	}
	// Get path and sha1
	if s, ok := reader.(canSHA1); ok {
		if h, err := s.SHA1(); err == nil {
			fv.SHA1 = h
		}
	}
	if s, ok := reader.(canDirSHA1); ok {
		if h, err := s.DirSHA1(); err == nil {
			fv.SHA1Dir = h
		}
	}
	if s, ok := reader.(canPath); ok {
		fv.File = s.Path()
	}
	return fv, nil
}

func FeedVersionServiceBounds(reader Reader) (time.Time, time.Time, error) {
	var start time.Time
	var end time.Time
	for c := range reader.Calendars() {
		if start.IsZero() || c.StartDate.Before(start) {
			start = c.StartDate
		}
		if end.IsZero() || c.EndDate.After(end) {
			end = c.EndDate
		}
	}
	for cd := range reader.CalendarDates() {
		if cd.ExceptionType != 1 {
			continue
		}
		if start.IsZero() || cd.Date.Before(start) {
			start = cd.Date
		}
		if end.IsZero() || cd.Date.After(end) {
			end = cd.Date
		}
	}
	if start.IsZero() || end.IsZero() {
		return start, end, errors.New("start or end dates were empty")
	}
	if end.Before(start) {
		return start, end, errors.New("end before start")
	}
	return start, end, nil
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

// ValidateStopTimes checks if the trip follows GTFS rules.
func ValidateStopTimes(stoptimes []StopTime) []error {
	errs := []error{}
	if len(stoptimes) == 0 {
		errs = append(errs, causes.NewEmptyTripError(len(stoptimes)))
		return errs // assumes >= 1 below
	}
	if len(stoptimes) < 2 {
		errs = append(errs, causes.NewEmptyTripError(len(stoptimes)))
	}
	if lastSt := stoptimes[len(stoptimes)-1]; lastSt.ArrivalTime.Int() <= 0 {
		errs = append(errs, causes.NewSequenceError("arrival_time", lastSt.ArrivalTime.String()))
	}
	lastDist := stoptimes[0].ShapeDistTraveled
	lastTime := stoptimes[0].DepartureTime
	lastSequence := stoptimes[0].StopSequence
	for _, st := range stoptimes[1:] {
		// Ensure we do not have duplicate StopSequennce
		if st.StopSequence == lastSequence {
			errs = append(errs, causes.NewSequenceError("stop_sequence", tt.TryCsv(st.StopSequence)))
		} else {
			lastSequence = st.StopSequence
		}
		// Ensure the arrows of time are pointing towards the future.
		if st.ArrivalTime.Int() > 0 && st.ArrivalTime.Int() < lastTime.Int() {
			errs = append(errs, causes.NewSequenceError("arrival_time", st.ArrivalTime.String()))
		} else if st.DepartureTime.Int() > 0 && st.DepartureTime.Int() < st.ArrivalTime.Int() {
			errs = append(errs, causes.NewSequenceError("departure_time", st.DepartureTime.String()))
		} else if st.DepartureTime.Int() > 0 {
			lastTime = st.DepartureTime
		}
		if st.ShapeDistTraveled.Valid && st.ShapeDistTraveled.Val < lastDist.Val {
			errs = append(errs, causes.NewSequenceError("shape_dist_traveled", st.ShapeDistTraveled.String()))
		} else if st.ShapeDistTraveled.Valid {
			lastDist = st.ShapeDistTraveled
		}
	}
	return errs
}
