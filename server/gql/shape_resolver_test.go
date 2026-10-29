package gql

import (
	"context"
	"math"
	"testing"

	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tlxy"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/stretchr/testify/assert"
)

// testShape is a shape through lon/lat points.
func testShape(pts ...tlxy.Point) *model.Shape {
	var coords []float64
	for _, p := range pts {
		coords = append(coords, p.Lon, p.Lat, 0)
	}
	s := &model.Shape{}
	s.Geometry = tt.NewLineStringFromFlatCoords(coords)
	return s
}

func TestStopShapeDistances(t *testing.T) {
	// A shape 1.1 km due north, with a stop at each end and one 2 km east of
	// its middle, which has no place on the shape.
	shape := testShape(tlxy.Point{Lon: -122.4, Lat: 37.8}, tlxy.Point{Lon: -122.4, Lat: 37.81})
	stops := []tlxy.Point{{Lon: -122.4, Lat: 37.8}, {Lon: -122.377, Lat: 37.805}, {Lon: -122.4, Lat: 37.81}}
	assert.Equal(t, []tt.Int{tt.NewInt(0), {}, tt.NewInt(1112)}, stopShapeDistances(shape, stops, nil))
	// A shape with a point without finite coordinates places no stop.
	bad := testShape(tlxy.Point{Lon: -122.4, Lat: 37.8}, tlxy.Point{Lon: -122.4, Lat: math.NaN()})
	assert.Equal(t, make([]tt.Int, len(stops)), stopShapeDistances(bad, stops, nil))
}

func TestShapeResolver_Distances(t *testing.T) {
	r := &shapeResolver{}
	got, err := r.Distances(context.Background(), testShape(tlxy.Point{Lon: -122.4, Lat: 37.8}, tlxy.Point{Lon: -122.4, Lat: 37.81}))
	assert.NoError(t, err)
	assert.Len(t, got, 2)
	assert.InDelta(t, 1111.95, got[1], 0.01)
	// A NaN coordinate gives no distances, rather than a NaN the response
	// can't encode.
	got, err = r.Distances(context.Background(), testShape(tlxy.Point{Lon: -122.4, Lat: 37.8}, tlxy.Point{Lon: math.NaN(), Lat: 37.81}))
	assert.NoError(t, err)
	assert.Nil(t, got)
}
