package builders

import (
	"strconv"
	"testing"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/internal/testpath"
	"github.com/interline-io/transitland-lib/internal/testreader"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFeedVersionGeometryBuilder(t *testing.T) {
	// The hull covers every stop, and a flex feed's locations too.
	tcs := []struct {
		name   string
		url    string
		expect []float64
	}{
		{
			name:   "ExampleFeed",
			url:    testreader.ExampleZip.URL,
			expect: []float64{-117.133162, 36.425288, -116.81797, 36.88108, -116.76821, 36.914893, -116.751677, 36.915682, -116.40094, 36.641496, -117.133162, 36.425288},
		},
		{
			name:   "Caltrain",
			url:    testreader.ExampleFeedCaltrain.URL,
			expect: []float64{-121.566225, 37.003485, -122.232, 37.486101, -122.386832, 37.599797, -122.412076, 37.631108, -122.394992, 37.77639, -122.394935, 37.776348, -121.650244, 37.129363, -121.610936, 37.086653, -121.610049, 37.085225, -121.566088, 37.003538, -121.566225, 37.003485},
		},
		{
			name:   "BART",
			url:    testreader.ExampleFeedBART.URL,
			expect: []float64{-121.939313, 37.502171, -122.386702, 37.600271, -122.466233, 37.684638, -122.469081, 37.706121, -122.353099, 37.936853, -122.024653, 38.003193, -121.945154, 38.018914, -121.889457, 38.016941, -121.78042, 37.995388, -121.939313, 37.502171},
		},
		{
			name:   "TriMet-2Routes",
			url:    testpath.RelPath("testdata/gtfs-external/trimet-2routes.zip"),
			expect: []float64{-122.567769, 45.435721, -122.671376, 45.493891, -122.698688, 45.530612, -122.696445, 45.531308, -122.621367, 45.532957, -122.578437, 45.533478, -122.563627, 45.530839, -122.563602, 45.530554, -122.563578, 45.530269, -122.567769, 45.435721},
		},
		{
			name:   "ctran-flex",
			url:    testpath.RelPath("testdata/gtfs-external/ctran-flex.zip"),
			expect: []float64{-122.33863611161699, 45.55614558774685, -122.4364333023227, 45.56993373848007, -122.68653064857236, 45.628926593221045, -122.72714226902016, 45.64584473015525, -122.73663261022726, 45.65320073963615, -122.75598128639, 45.82431341448629, -122.74887976218521, 45.83177050749847, -122.69085020031012, 45.8734447163598, -122.68583707198361, 45.87699958672877, -122.68354008121293, 45.87700412846506, -122.66510457544769, 45.87682251316417, -122.6599361175616, 45.876014972557094, -122.52671446536463, 45.80281651353668, -122.51649678818269, 45.79558370666473, -122.30620247861314, 45.592571327141755, -122.30616554734422, 45.584496565769385, -122.30883596862645, 45.57373784021691, -122.32137169640104, 45.55979528563866, -122.33863611161699, 45.55614558774685},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			e := NewFeedVersionGeometryBuilder()
			if _, _, err := newMockCopier(tc.url, e); err != nil {
				t.Fatal(err)
			}
			got := e.Geometry()
			if assert.True(t, got.Valid, "a hull") {
				assert.InEpsilonSlice(t, tc.expect, got.FlatCoords(), 0.001)
			}
		})
	}
}

func TestFeedVersionGeometryBuilder_StopWithoutCoordinates(t *testing.T) {
	// A stop without coordinates stays out of the hull instead of pulling it to
	// (0,0).
	e := NewFeedVersionGeometryBuilder()
	for i, lonlat := range [][2]float64{{-122.0, 37.0}, {-122.1, 37.1}, {-122.0, 37.2}} {
		stop := gtfs.Stop{Geometry: tt.NewPoint(lonlat[0], lonlat[1])}
		require.NoError(t, e.AfterWrite(strconv.Itoa(i), &stop, nil))
	}
	require.NoError(t, e.AfterWrite("node", &gtfs.Stop{}, nil))
	require.NoError(t, e.Copy(nil))
	got := e.Geometry()
	require.True(t, got.Valid, "a hull")
	coords := got.FlatCoords()
	for i := 0; i < len(coords); i += 2 {
		assert.Less(t, coords[i], -121.0, "every vertex is one of the three stops")
	}
}
