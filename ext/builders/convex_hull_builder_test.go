package builders

import (
	"testing"

	"github.com/interline-io/transitland-lib/internal/testutil"
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
			testutil.ExampleZip.URL,
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
			testutil.ExampleFeedCaltrain.URL,
			[]testcase{{
				FeedVersionGeometry: []float64{-121.566225, 37.003485, -122.232, 37.486101, -122.386832, 37.599797, -122.412076, 37.631108, -122.394992, 37.77639, -122.394935, 37.776348, -121.650244, 37.129363, -121.610936, 37.086653, -121.610049, 37.085225, -121.566088, 37.003538, -121.566225, 37.003485},
				AgencyGeoms: map[string][]float64{
					"caltrain-ca-us": {-121.566225, 37.003485, -122.232, 37.486101, -122.386832, 37.599797, -122.412076, 37.631108, -122.394992, 37.77639, -122.394935, 37.776348, -121.650244, 37.129363, -121.610936, 37.086653, -121.610049, 37.085225, -121.566088, 37.003538, -121.566225, 37.003485},
				},
			}},
		},
		"BART": {
			testutil.ExampleFeedBART.URL,
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
			testutil.RelPath("testdata/external/trimet-2routes.zip"),
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
	}
	for groupName, testGroup := range groups {
		t.Run(groupName, func(t *testing.T) {
			cp, writer, err := newMockCopier(testGroup.URL)
			if err != nil {
				t.Fatal(err)
			}
			e := NewConvexHullBuilder()
			cp.AddExtension(e)
			cpr := cp.Copy()
			if cpr.WriteError != nil {
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
