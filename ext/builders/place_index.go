package builders

import (
	"cmp"
	"context"
	"math"
	"slices"

	"github.com/interline-io/transitland-lib/tlxy"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/geojson"
)

// AdminArea is a first-level admin area, such as a state or province.
type AdminArea struct {
	Name     string // the area's name
	Admin    string // its country's name
	IsoA2    string // the country's ISO 3166-1 alpha-2 code
	Iso31662 string // the area's ISO 3166-2 code
	Geometry geom.T // a polygon or multipolygon
}

// PopulatedPlace is a populated place, with the admin area and country its
// source names.
type PopulatedPlace struct {
	Name     string
	Adm1Name string
	Adm0Name string
	Point    tlxy.Point
}

// PlaceIndex answers PlaceLookup from admin areas and populated places held in
// memory.
type PlaceIndex struct {
	areas  *tlxy.PolygonIndex
	places []indexedPlace // by latitude
}

type indexedPlace struct {
	place Place
	pt    tlxy.Point
}

// NewPlaceIndex indexes admin areas and populated places. A place lying in an
// admin area takes that area's names.
func NewPlaceIndex(areas []AdminArea, places []PopulatedPlace) (*PlaceIndex, error) {
	// Each polygon of an area goes in on its own bounds, so a search tests
	// only the polygons near the point.
	var fc geojson.FeatureCollection
	for _, a := range areas {
		var pgs []*geom.Polygon
		switch g := a.Geometry.(type) {
		case *geom.Polygon:
			pgs = append(pgs, g)
		case *geom.MultiPolygon:
			for i := 0; i < g.NumPolygons(); i++ {
				pgs = append(pgs, g.Polygon(i))
			}
		}
		area := a
		area.Geometry = nil
		for _, pg := range pgs {
			fc.Features = append(fc.Features, &geojson.Feature{
				Geometry:   pg,
				Properties: map[string]any{"area": area},
			})
		}
	}
	idx, err := tlxy.NewPolygonIndex(fc)
	if err != nil {
		return nil, err
	}
	ix := &PlaceIndex{areas: idx}
	for _, p := range places {
		pl := Place{Name: tt.NewString(p.Name), Adm1name: tt.NewString(p.Adm1Name), Adm0name: tt.NewString(p.Adm0Name)}
		if a, ok := ix.within(p.Point); ok {
			pl.Adm1name, pl.Adm0name = tt.NewString(a.Name), tt.NewString(a.Admin)
		}
		ix.places = append(ix.places, indexedPlace{place: pl, pt: p.Point})
	}
	slices.SortFunc(ix.places, func(a, b indexedPlace) int { return cmp.Compare(a.pt.Lat, b.pt.Lat) })
	return ix, nil
}

// Admin returns the admin area holding pt.
func (ix *PlaceIndex) Admin(ctx context.Context, pt tlxy.Point) (Place, bool, error) {
	a, ok := ix.within(pt)
	if !ok {
		return Place{}, false, nil
	}
	return Place{Adm1name: tt.NewString(a.Name), Adm0name: tt.NewString(a.Admin)}, true, nil
}

// NearestAdmin returns the admin area holding pt or, for a point just outside
// every area, such as a ferry landing, the nearest area within a quarter degree.
func (ix *PlaceIndex) NearestAdmin(pt tlxy.Point) (AdminArea, bool) {
	feat, n := ix.areas.NearestFeature(pt)
	if n == 0 {
		return AdminArea{}, false
	}
	a, ok := feat.Properties["area"].(AdminArea)
	return a, ok
}

func (ix *PlaceIndex) within(pt tlxy.Point) (AdminArea, bool) {
	feat, n := ix.areas.WithinFeature(pt)
	if n == 0 {
		return AdminArea{}, false
	}
	a, ok := feat.Properties["area"].(AdminArea)
	return a, ok
}

// NearestPlace returns the populated place nearest pt, within maxMeters.
func (ix *PlaceIndex) NearestPlace(ctx context.Context, pt tlxy.Point, maxMeters float64) (Place, bool, error) {
	// A place within maxMeters of pt lies within maxMeters of its latitude.
	dLat := maxMeters / tlxy.EarthRadiusMeters * 180 / math.Pi
	i, _ := slices.BinarySearchFunc(ix.places, pt.Lat-dLat, func(p indexedPlace, lat float64) int {
		return cmp.Compare(p.pt.Lat, lat)
	})
	var nearest Place
	found := false
	best := maxMeters
	for ; i < len(ix.places) && ix.places[i].pt.Lat <= pt.Lat+dLat; i++ {
		if d := tlxy.DistanceHaversine(pt, ix.places[i].pt); d <= best {
			nearest, best, found = ix.places[i].place, d, true
		}
	}
	return nearest, found, nil
}
