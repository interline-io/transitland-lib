package builders

import (
	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/xy"
)

// FeedVersionGeometry is a feed version's convex hull.
type FeedVersionGeometry struct {
	Geometry tt.Polygon
	tt.MinEntity
	tt.FeedVersionEntity
}

func (ent *FeedVersionGeometry) Filename() string {
	return "tl_feed_version_geometries.txt"
}

func (ent *FeedVersionGeometry) TableName() string {
	return "tl_feed_version_geometries"
}

// FeedVersionGeometryBuilder computes a feed version's convex hull, from its
// stops and flex locations.
type FeedVersionGeometryBuilder struct {
	coords   []float64
	geometry tt.Polygon
}

// NewFeedVersionGeometryBuilder returns a FeedVersionGeometryBuilder.
func NewFeedVersionGeometryBuilder() *FeedVersionGeometryBuilder {
	return &FeedVersionGeometryBuilder{}
}

// AfterWrite collects the coordinates of each stop and flex location.
func (pp *FeedVersionGeometryBuilder) AfterWrite(eid string, ent tt.Entity, emap *tt.EntityMap) error {
	switch v := ent.(type) {
	case *gtfs.Stop:
		// A stop without coordinates, such as a generic node, has none to add;
		// its zero values would pull the hull to (0,0).
		if v.Geometry.Valid {
			pp.coords = append(pp.coords, v.Geometry.X(), v.Geometry.Y())
		}
	case *gtfs.Location:
		pp.coords = append(pp.coords, v.Geometry.FlatCoords()...)
	}
	return nil
}

// Copy computes the hull of the collected coordinates.
func (pp *FeedVersionGeometryBuilder) Copy(_ adapters.EntityCopier) error {
	if v, ok := xy.ConvexHullFlat(geom.XY, pp.coords).(*geom.Polygon); ok {
		pp.geometry = tt.NewPolygon(v)
	}
	return nil
}

// Geometry returns the hull after Copy has run. It is unset when the stops and
// locations make no polygon.
func (pp *FeedVersionGeometryBuilder) Geometry() tt.Polygon {
	return pp.geometry
}
