package tlxy

import (
	_ "embed"

	"github.com/tidwall/rtree"
	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/geojson"
	"github.com/twpayne/go-geom/xy"
)

type pipShape struct {
	Name       string
	Properties map[string]any
	Polygon    *geom.Polygon
}

type PolygonIndex struct {
	idx *rtree.RTreeG[pipShape]
}

func (pip *PolygonIndex) FeatureAt(pt Point) (*geojson.Feature, bool) {
	ggPoint := geom.NewPointFlat(geom.XY, []float64{pt.Lon, pt.Lat})
	rtPoint := [2]float64{pt.Lon, pt.Lat}
	found := false
	var ret *geojson.Feature
	pip.idx.Search(rtPoint, rtPoint, func(a, b [2]float64, s pipShape) bool {
		if pointInPolygon(s.Polygon, ggPoint) {
			ret = &geojson.Feature{
				ID:         s.Name,
				Properties: s.Properties,
				Geometry:   s.Polygon,
			}
			found = true
			return false
		}
		return true
	})
	return ret, found
}

func (pip *PolygonIndex) FeatureNameAt(pt Point) (string, bool) {
	a, ok := pip.FeatureAt(pt)
	if ok {
		return a.ID, true
	}
	return "", false
}

func NewPolygonIndex(fc geojson.FeatureCollection) (*PolygonIndex, error) {
	PolygonIndex := PolygonIndex{
		idx: &rtree.RTreeG[pipShape]{},
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
