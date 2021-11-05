package builders

import (
	"testing"

	"github.com/interline-io/transitland-lib/internal/testutil"
)

func TestAgencyPlaceBuilder(t *testing.T) {
	type testcase struct {
	}
	type testgroup struct {
		URL   string
		Cases []testcase
	}
	groups := map[string]testgroup{
		"ExampleFeed": {
			testutil.ExampleZip.URL,
			[]testcase{
				{},
			},
		},
		"Caltrain": {
			testutil.ExampleFeedCaltrain.URL,
			[]testcase{{}},
		},
		"BART": {
			testutil.ExampleFeedBART.URL,
			[]testcase{
				{},
			},
		},
	}
	for groupName, testGroup := range groups {
		t.Run(groupName, func(t *testing.T) {
			cp, writer, err := newMockCopier(testGroup.URL)
			if err != nil {
				t.Fatal(err)
			}
			e := NewAgencyPlaceBuilder()
			cp.AddExtension(e)
			cpr := cp.Copy()
			if cpr.WriteError != nil {
				t.Fatal(err)
			}
			agencyPlaces := []*AgencyPlace{}
			for _, ent := range writer.Reader.OtherList {
				switch v := ent.(type) {
				case *AgencyPlace:
					agencyPlaces = append(agencyPlaces, v)
				}
			}
			for _, tc := range testGroup.Cases {
				_ = tc
				// for aid, v := range tc.AgencyGeoms {
				// 	t.Run("Agency:"+aid, func(t *testing.T) {
				// 		if gotaidg, ok := aGeoms[aid]; !ok {
				// 			t.Errorf("no agency geometry for %s", aid)
				// 		} else {
				// 			assert.InEpsilonSlice(t, gotaidg.Geometry.FlatCoords(), v, 0.001)
				// 		}
				// 	})
				// }
			}
		})
	}
}
