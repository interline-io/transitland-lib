package gtfs

import (
	"testing"

	"github.com/interline-io/transitland-lib/tt"
	geom "github.com/twpayne/go-geom"
)

func TestLevel_Errors(t *testing.T) {
	newLevel := func(fn func(*Level)) *Level {
		level := &Level{
			LevelID:    tt.NewString("L1"),
			LevelIndex: tt.NewFloat(1.0),
			LevelName:  tt.NewString("Level 1"),
		}
		if fn != nil {
			fn(level)
		}
		return level
	}

	tests := []struct {
		name           string
		level          *Level
		expectedErrors []ExpectError
	}{
		{
			name:           "Valid level",
			level:          newLevel(nil),
			expectedErrors: nil,
		},
		{
			name: "Missing level_id",
			level: newLevel(func(l *Level) {
				l.LevelID = tt.String{}
			}),
			expectedErrors: PE("RequiredFieldError:level_id"),
		},
		{
			name: "Missing level_index",
			level: newLevel(func(l *Level) {
				l.LevelIndex = tt.Float{}
			}),
			expectedErrors: PE("RequiredFieldError:level_index"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			errs := tt.CheckErrors(tc.level)
			CheckErrors(tc.expectedErrors, errs, t)
		})
	}
}

func TestLevel_Methods(t *testing.T) {
	level := &Level{
		LevelID:    tt.NewString("L1"),
		LevelIndex: tt.NewFloat(1.0),
		LevelName:  tt.NewString("Level 1"),
	}
	level.ID = 123

	if got := level.EntityID(); got != "123" {
		t.Errorf("EntityID() = %v, want %v", got, "123")
	}
	level.ID = 0
	if got := level.EntityID(); got != "L1" {
		t.Errorf("EntityID() = %v, want %v", got, "L1")
	}

	if got := level.EntityKey(); got != "L1" {
		t.Errorf("EntityKey() = %v, want %v", got, "L1")
	}
	if got := level.Filename(); got != "levels.txt" {
		t.Errorf("Filename() = %v, want %v", got, "levels.txt")
	}
	if got := level.TableName(); got != "gtfs_levels" {
		t.Errorf("TableName() = %v, want %v", got, "gtfs_levels")
	}
}

func TestLevel_WithGeometry(t *testing.T) {
	// Create a polygon geometry
	polygon, err := geom.NewPolygon(geom.XY).SetCoords([][]geom.Coord{
		{{-122.4194, 37.7749}, {-122.4094, 37.7749}, {-122.4094, 37.7649}, {-122.4194, 37.7649}, {-122.4194, 37.7749}},
	})
	if err != nil {
		t.Fatal(err)
	}
	polygon.SetSRID(4326)

	level := &Level{
		LevelID:    tt.NewString("L1"),
		LevelIndex: tt.NewFloat(0),
		LevelName:  tt.NewString("Ground Floor"),
		Geometry:   tt.NewGeometry(polygon),
	}

	// Check that geometry is valid
	if !level.Geometry.Valid {
		t.Error("Geometry should be valid")
	}

	// Check that geometry can be retrieved
	if level.Geometry.Val == nil {
		t.Error("Geometry.Val should not be nil")
	}

	// Check that it's a Polygon
	if _, ok := level.Geometry.Val.(*geom.Polygon); !ok {
		t.Errorf("Geometry should be a Polygon, got %T", level.Geometry.Val)
	}
}

func TestLevel_WithoutGeometry(t *testing.T) {
	level := &Level{
		LevelID:    tt.NewString("L1"),
		LevelIndex: tt.NewFloat(0),
		LevelName:  tt.NewString("Ground Floor"),
	}

	// Check that geometry is not valid when not set
	if level.Geometry.Valid {
		t.Error("Geometry should not be valid when not set")
	}
}
