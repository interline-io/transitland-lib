package ne

import (
	"github.com/interline-io/transitland-lib/ext/builders"
	"github.com/interline-io/transitland-lib/tlxy"
	"github.com/twpayne/go-geom"
)

// PlaceIndex returns an index of the admin areas and populated places in the
// embedded Natural Earth data.
func PlaceIndex() (*builders.PlaceIndex, error) {
	areaShapes, err := readEmbeddedShapes(adminsZip)
	if err != nil {
		return nil, err
	}
	placeShapes, err := readEmbeddedShapes(placesZip)
	if err != nil {
		return nil, err
	}
	var areas []builders.AdminArea
	for _, s := range areaShapes {
		areas = append(areas, builders.AdminArea{
			Name:     s.attrs["name"],
			Admin:    s.attrs["admin"],
			IsoA2:    s.attrs["iso_a2"],
			Iso31662: s.attrs["iso_3166_2"],
			Geometry: s.geometry,
		})
	}
	var places []builders.PopulatedPlace
	for _, s := range placeShapes {
		pt, ok := s.geometry.(*geom.Point)
		if !ok {
			continue
		}
		places = append(places, builders.PopulatedPlace{
			Name:     s.attrs["name"],
			Adm1Name: s.attrs["adm1name"],
			Adm0Name: s.attrs["adm0name"],
			Point:    tlxy.Point{Lon: pt.X(), Lat: pt.Y()},
		})
	}
	return builders.NewPlaceIndex(areas, places)
}
