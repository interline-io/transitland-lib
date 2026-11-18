package ne

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"sync"

	"github.com/interline-io/transitland-lib/tlxy"
	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/geojson"
)

// AdminArea is a first-level admin area, such as a state or province.
type AdminArea struct {
	Name     string // the area's name
	Admin    string // its country's name
	IsoA2    string // the country's ISO 3166-1 alpha-2 code
	Iso31662 string // the area's ISO 3166-2 code
	Geometry geom.T // an XY polygon or multipolygon
}

// PopulatedPlace is a populated place, such as a city, with its admin area and
// country.
type PopulatedPlace struct {
	Name     string
	Adm1Name string
	Adm0Name string
	Point    tlxy.Point
}

// PlaceIndex holds admin areas and populated places in memory.
type PlaceIndex struct {
	areas          *tlxy.PolygonIndex
	areasByName    map[[2]string][]AdminArea // by name and country
	areasByCountry map[string][]AdminArea
	places         []PopulatedPlace               // by latitude
	placesByName   map[[3]string][]PopulatedPlace // by a place's own names, and by its containing area's
}

var sharedPlaces = sync.OnceValues(readPlaceIndex)

// Places returns the index of the embedded admin areas and populated places,
// which the first call reads and later calls share.
func Places() (*PlaceIndex, error) {
	return sharedPlaces()
}

// NewPlaceIndex indexes admin areas and populated places. A place lying in an
// admin area takes that area's names.
func NewPlaceIndex(areas []AdminArea, places []PopulatedPlace) (*PlaceIndex, error) {
	var fc geojson.FeatureCollection
	for _, a := range areas {
		// The polygon index tests points against XY polygons only.
		switch g := a.Geometry.(type) {
		case *geom.Polygon, *geom.MultiPolygon:
			if g.Layout() != geom.XY {
				return nil, fmt.Errorf("admin area %s: layout is %v, not XY", a.Iso31662, g.Layout())
			}
		default:
			return nil, fmt.Errorf("admin area %s: not a polygon or multipolygon", a.Iso31662)
		}
		fc.Features = append(fc.Features, &geojson.Feature{Geometry: a.Geometry, Properties: map[string]any{"area": a}})
	}
	idx, err := tlxy.NewPolygonIndex(fc)
	if err != nil {
		return nil, err
	}
	ix := &PlaceIndex{
		areas:          idx,
		areasByName:    map[[2]string][]AdminArea{},
		areasByCountry: map[string][]AdminArea{},
		placesByName:   map[[3]string][]PopulatedPlace{},
	}
	for _, a := range areas {
		k := [2]string{a.Name, a.Admin}
		ix.areasByName[k] = append(ix.areasByName[k], a)
		ix.areasByCountry[a.Admin] = append(ix.areasByCountry[a.Admin], a)
	}
	for _, p := range places {
		// A place goes by its own names and by those of the area holding it,
		// which can differ.
		own := [3]string{p.Name, p.Adm1Name, p.Adm0Name}
		if a, ok := ix.Admin(p.Point); ok {
			p.Adm1Name, p.Adm0Name = a.Name, a.Admin
		}
		ix.places = append(ix.places, p)
		ix.placesByName[own] = append(ix.placesByName[own], p)
		if k := [3]string{p.Name, p.Adm1Name, p.Adm0Name}; k != own {
			ix.placesByName[k] = append(ix.placesByName[k], p)
		}
	}
	slices.SortFunc(ix.places, func(a, b PopulatedPlace) int { return cmp.Compare(a.Point.Lat, b.Point.Lat) })
	return ix, nil
}

// Admin returns the admin area holding pt.
func (ix *PlaceIndex) Admin(pt tlxy.Point) (AdminArea, bool) {
	return foundArea(ix.areas.WithinFeature(pt))
}

// NearestAdmin returns the admin area holding pt or, for a point just outside
// every area, such as a ferry landing, the nearest area within a quarter degree.
func (ix *PlaceIndex) NearestAdmin(pt tlxy.Point) (AdminArea, bool) {
	return foundArea(ix.areas.NearestFeature(pt))
}

// AdminsNamed returns the admin areas with a name in a country. Several areas
// of a country can share one.
func (ix *PlaceIndex) AdminsNamed(name, country string) []AdminArea {
	return ix.areasByName[[2]string{name, country}]
}

// CountryAdmins returns a country's admin areas.
func (ix *PlaceIndex) CountryAdmins(country string) []AdminArea {
	return ix.areasByCountry[country]
}

// PlacesNamed returns the populated places with a name, in an admin area and
// country with these names: the place's own, or its containing area's.
func (ix *PlaceIndex) PlacesNamed(name, adm1, adm0 string) []PopulatedPlace {
	return ix.placesByName[[3]string{name, adm1, adm0}]
}

// foundArea returns the admin area that a search of the polygon index found.
func foundArea(feat *geojson.Feature, n int) (AdminArea, bool) {
	if n == 0 {
		return AdminArea{}, false
	}
	a, ok := feat.Properties["area"].(AdminArea)
	return a, ok
}

// NearestPlace returns the populated place nearest pt, within maxMeters.
func (ix *PlaceIndex) NearestPlace(pt tlxy.Point, maxMeters float64) (PopulatedPlace, bool) {
	// A place within maxMeters of pt lies within dLat degrees of pt's latitude.
	dLat := maxMeters / tlxy.EarthRadiusMeters * 180 / math.Pi
	i, _ := slices.BinarySearchFunc(ix.places, pt.Lat-dLat, func(p PopulatedPlace, lat float64) int {
		return cmp.Compare(p.Point.Lat, lat)
	})
	var nearest PopulatedPlace
	found := false
	best := maxMeters
	for ; i < len(ix.places) && ix.places[i].Point.Lat <= pt.Lat+dLat; i++ {
		if d := tlxy.DistanceHaversine(pt, ix.places[i].Point); d <= best {
			nearest, best, found = ix.places[i], d, true
		}
	}
	return nearest, found
}

// readPlaceIndex reads the embedded shapefiles into an index.
func readPlaceIndex() (*PlaceIndex, error) {
	areaShapes, err := readEmbeddedShapes(adminsZip)
	if err != nil {
		return nil, err
	}
	placeShapes, err := readEmbeddedShapes(placesZip)
	if err != nil {
		return nil, err
	}
	var areas []AdminArea
	for _, s := range areaShapes {
		areas = append(areas, AdminArea{
			Name:     s.attrs["name"],
			Admin:    s.attrs["admin"],
			IsoA2:    s.attrs["iso_a2"],
			Iso31662: s.attrs["iso_3166_2"],
			Geometry: s.geometry,
		})
	}
	var places []PopulatedPlace
	for _, s := range placeShapes {
		pt, ok := s.geometry.(*geom.Point)
		if !ok {
			continue
		}
		places = append(places, PopulatedPlace{
			Name:     s.attrs["name"],
			Adm1Name: s.attrs["adm1name"],
			Adm0Name: s.attrs["adm0name"],
			Point:    tlxy.Point{Lon: pt.X(), Lat: pt.Y()},
		})
	}
	return NewPlaceIndex(areas, places)
}
