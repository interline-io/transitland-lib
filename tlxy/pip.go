// Package tlxy provides spatial indexing for polygons.
package tlxy

import (
	_ "embed"

	"github.com/tidwall/rtree"
	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/geojson"
	"github.com/twpayne/go-geom/xy"
)

// PolygonIndex enables efficient point-in-polygon lookups using an R-tree.
type PolygonIndex struct {
	idx rtree.Generic[pipShape]
}

// NewPolygonIndex builds an index from a GeoJSON FeatureCollection.
// Returns (index, error).
func NewPolygonIndex(fc geojson.FeatureCollection) (*PolygonIndex, error) {
	PolygonIndex := PolygonIndex{
		idx: rtree.Generic[pipShape]{},
	}
	for _, feature := range fc.Features {
		bb := geom.NewBounds(geom.XY)
		bb.Extend(feature.Geometry)
		bb1, bb2 := [2]float64{bb.Min(0), bb.Min(1)}, [2]float64{bb.Max(0), bb.Max(1)}
		if v, ok := feature.Geometry.(*geom.Polygon); ok {
			PolygonIndex.idx.Insert(bb1, bb2, pipShape{
				Name:       feature.ID,
				Properties: feature.Properties,
				Polygon:    v,
			},
			)
		} else if v, ok := feature.Geometry.(*geom.MultiPolygon); ok {
			for i := 0; i < v.NumPolygons(); i++ {
				PolygonIndex.idx.Insert(bb1, bb2, pipShape{
					Name:       feature.ID,
					Properties: feature.Properties,
					Polygon:    v.Polygon(i),
				},
				)
			}
		}
	}
	return &PolygonIndex, nil
}

// WithinFeature returns a polygon containing the point, and the number of matching polygons.
func (pip *PolygonIndex) WithinFeature(pt Point) (*geojson.Feature, int) {
	// No, we are not being fancy with projections.
	// That could be improved.
	var ret *geojson.Feature
	gp := geom.NewPointFlat(geom.XY, []float64{pt.Lon, pt.Lat})
	count := 0
	pip.idx.Search(
		[2]float64{pt.Lon, pt.Lat},
		[2]float64{pt.Lon, pt.Lat},
		func(min, max [2]float64, s pipShape) bool {
			if pointInPolygon(s.Polygon, gp) {
				ret = &geojson.Feature{
					ID:         s.Name,
					Properties: s.Properties,
					Geometry:   s.Polygon,
				}
				count += 1
			}
			return true
		},
	)
	return ret, count
}

// Check does a quick search, then does a nearest feature search if no match is found.
func (pip *PolygonIndex) NearestFeature(pt Point) (*geojson.Feature, int) {
	ret, count := pip.WithinFeature(pt)
	if count >= 1 {
		return ret, count
	}
	tolerance := 0.25
	nearestAdmin, _, count := pip.nearestFeatureTolerance(pt, tolerance)
	return nearestAdmin, count
}

// NearestFeature returns the nearest polygon to the point within a tolerance, the distance, and the number of matching polygons.
func (pip *PolygonIndex) nearestFeatureTolerance(pt Point, tolerance float64) (*geojson.Feature, float64, int) {
	minDist := -1.0
	gp := geom.NewPointFlat(geom.XY, []float64{pt.Lon, pt.Lat})
	count := 0
	var ret *geojson.Feature
	pip.idx.Search(
		[2]float64{pt.Lon - tolerance, pt.Lat - tolerance},
		[2]float64{pt.Lon + tolerance, pt.Lat + tolerance},
		func(min, max [2]float64, s pipShape) bool {
			d := pointPolygonDistance(s.Polygon, gp)
			if d < tolerance && (d < minDist || minDist < 0) {
				ret = &geojson.Feature{
					ID:         s.Name,
					Properties: s.Properties,
					Geometry:   s.Polygon,
				}
				count += 1
				minDist = d
			}
			return true
		},
	)
	return ret, minDist, count
}

// pointInPolygon tests if a point is inside a polygon's outer ring but not in its holes.
// Returns true if point is contained.
func pointInPolygon(pg *geom.Polygon, p *geom.Point) bool {
	if !xy.IsPointInRing(geom.XY, p.Coords(), pg.LinearRing(0).FlatCoords()) {
		return false
	}
	for i := 1; i < pg.NumLinearRings(); i++ {
		if xy.IsPointInRing(geom.XY, p.Coords(), pg.LinearRing(i).FlatCoords()) {
			return false
		}
	}
	return true
}

func pointPolygonDistance(pg *geom.Polygon, p *geom.Point) float64 {
	minDist := -1.0
	c := geom.Coord{p.X(), p.Y()}
	for i := 0; i < pg.NumLinearRings(); i++ {
		d := xy.DistanceFromPointToLineString(p.Layout(), c, pg.LinearRing(i).FlatCoords())
		if d < minDist || minDist < 0 {
			minDist = d
		}
	}
	return minDist
}

// pipShape stores a polygon with its identifier and metadata.
type pipShape struct {
	Name       string
	Properties map[string]any
	Polygon    *geom.Polygon
}
