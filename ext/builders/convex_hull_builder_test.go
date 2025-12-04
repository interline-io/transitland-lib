package builders

import (
	"testing"

	"github.com/interline-io/transitland-lib/internal/testpath"
	"github.com/interline-io/transitland-lib/internal/testreader"
	"github.com/stretchr/testify/assert"
)

func TestConvexHullBuilder(t *testing.T) {
	type testcase struct {
		FeedVersionGeometry []float64
		AgencyGeoms         map[string][]float64
	}
	type testgroup struct {
		URL   string
		Cases []testcase
	}
	groups := map[string]testgroup{
		"ExampleFeed": {
			testreader.ExampleZip.URL,
			[]testcase{
				{
					FeedVersionGeometry: []float64{-117.133162, 36.425288, -116.81797, 36.88108, -116.76821, 36.914893, -116.751677, 36.915682, -116.40094, 36.641496, -117.133162, 36.425288},
					AgencyGeoms: map[string][]float64{
						"DTA": {-117.133162, 36.425288, -116.81797, 36.88108, -116.76821, 36.914893, -116.751677, 36.915682, -116.40094, 36.641496, -117.133162, 36.425288},
					},
				},
			},
		},
		"Caltrain": {
			testreader.ExampleFeedCaltrain.URL,
			[]testcase{{
				FeedVersionGeometry: []float64{-121.566225, 37.003485, -122.232, 37.486101, -122.386832, 37.599797, -122.412076, 37.631108, -122.394992, 37.77639, -122.394935, 37.776348, -121.650244, 37.129363, -121.610936, 37.086653, -121.610049, 37.085225, -121.566088, 37.003538, -121.566225, 37.003485},
				AgencyGeoms: map[string][]float64{
					"caltrain-ca-us": {-121.566225, 37.003485, -122.232, 37.486101, -122.386832, 37.599797, -122.412076, 37.631108, -122.394992, 37.77639, -122.394935, 37.776348, -121.650244, 37.129363, -121.610936, 37.086653, -121.610049, 37.085225, -121.566088, 37.003538, -121.566225, 37.003485},
				},
			}},
		},
		"BART": {
			testreader.ExampleFeedBART.URL,
			[]testcase{
				{
					FeedVersionGeometry: []float64{-121.939313, 37.502171, -122.386702, 37.600271, -122.466233, 37.684638, -122.469081, 37.706121, -122.353099, 37.936853, -122.024653, 38.003193, -121.945154, 38.018914, -121.889457, 38.016941, -121.78042, 37.995388, -121.939313, 37.502171},
					AgencyGeoms: map[string][]float64{
						"BART": {-121.939313, 37.502171, -122.386702, 37.600271, -122.466233, 37.684638, -122.469081, 37.706121, -122.353099, 37.936853, -122.024653, 38.003193, -121.945154, 38.018914, -121.889457, 38.016941, -121.78042, 37.995388, -121.939313, 37.502171},
					},
				},
			},
		},
		"TriMet-2Routes": {
			testpath.RelPath("testdata/gtfs-external/trimet-2routes.zip"),
			[]testcase{
				{
					FeedVersionGeometry: []float64{-122.567769, 45.435721, -122.671376, 45.493891, -122.698688, 45.530612, -122.696445, 45.531308, -122.621367, 45.532957, -122.578437, 45.533478, -122.563627, 45.530839, -122.563602, 45.530554, -122.563578, 45.530269, -122.567769, 45.435721},
					AgencyGeoms: map[string][]float64{
						"TRIMET": {-122.567769, 45.435721, -122.682999, 45.508979, -122.683593, 45.509616, -122.676517, 45.527222, -122.665557, 45.530235, -122.621367, 45.532957, -122.578437, 45.533478, -122.563627, 45.530839, -122.563578, 45.530269, -122.567769, 45.435721},
						"PSC":    {-122.671376, 45.493891, -122.698688, 45.530612, -122.696445, 45.531308, -122.694455, 45.531346, -122.689417, 45.531434, -122.685357, 45.531503, -122.68332, 45.531535, -122.681364, 45.53128, -122.670739, 45.498939, -122.670933, 45.495594, -122.671376, 45.493891},
					},
				},
			},
		},
		"C-Tran-Flex": {
			testpath.RelPath("testdata/gtfs-external/ctran-flex.zip"),
			[]testcase{
				{
					FeedVersionGeometry: []float64{-122.33863611161699, 45.55614558774685, -122.4364333023227, 45.56993373848007, -122.68653064857236, 45.628926593221045, -122.72714226902016, 45.64584473015525, -122.73663261022726, 45.65320073963615, -122.75598128639, 45.82431341448629, -122.74887976218521, 45.83177050749847, -122.69085020031012, 45.8734447163598, -122.68583707198361, 45.87699958672877, -122.68354008121293, 45.87700412846506, -122.66510457544769, 45.87682251316417, -122.6599361175616, 45.876014972557094, -122.52671446536463, 45.80281651353668, -122.51649678818269, 45.79558370666473, -122.30620247861314, 45.592571327141755, -122.30616554734422, 45.584496565769385, -122.30883596862645, 45.57373784021691, -122.32137169640104, 45.55979528563866, -122.33863611161699, 45.55614558774685},
					AgencyGeoms:         map[string][]float64{},
				},
			},
		},
	}
	for groupName, testGroup := range groups {
		t.Run(groupName, func(t *testing.T) {
			e := NewConvexHullBuilder()
			_, writer, err := newMockCopier(testGroup.URL, e)
			if err != nil {
				t.Fatal(err)
			}
			fvGeoms := []*FeedVersionGeometry{}
			aGeoms := map[string]*AgencyGeometry{}
			for _, ent := range writer.Reader.OtherList {
				switch v := ent.(type) {
				case *AgencyGeometry:
					aGeoms[v.AgencyID.Val] = v
					// t.Logf("%s %#v\n", v.AgencyID.Key, v.Geometry.FlatCoords())
					// z, _ := geojson.Marshal(&v.Geometry.Polygon)
					// t.Log(string(z))
				case *FeedVersionGeometry:
					fvGeoms = append(fvGeoms, v)
				}
			}
			for _, tc := range testGroup.Cases {
				t.Run("FeedVersion", func(t *testing.T) {
					if len(fvGeoms) != 1 {
						t.Error("did not get feed version geometry")
					} else if fvg := fvGeoms[0]; fvg != nil {
						gotfvg := fvg.Geometry.FlatCoords()
						assert.InEpsilonSlice(t, gotfvg, tc.FeedVersionGeometry, 0.001)
					}
				})
				for aid, v := range tc.AgencyGeoms {
					t.Run("Agency:"+aid, func(t *testing.T) {
						if gotaidg, ok := aGeoms[aid]; !ok {
							t.Errorf("no agency geometry for %s", aid)
						} else {
							assert.InEpsilonSlice(t, gotaidg.Geometry.FlatCoords(), v, 0.001)
						}
					})
				}
			}
		})
	}
}
