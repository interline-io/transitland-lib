package builders

import (
	"fmt"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestRouteHeadwayBuilder(t *testing.T) {
	type testcase struct {
		RouteID       string
		DowCat        int
		DirectionID   int
		StopID        string
		ServiceDate   string
		HeadwaySecs   int
		StopTripCount int
		Departures    []int
	}
	type testgroup struct {
		URL   string
		Cases []testcase
	}
	groups := map[string]testgroup{
		"Caltrain": {
			testutil.ExampleFeedCaltrain.URL,
			[]testcase{
				{RouteID: "Bu-130"},
				{RouteID: "Lo-130"},
			},
		},
		"BART": {
			testutil.ExampleFeedBART.URL,
			[]testcase{
				{RouteID: "07", DowCat: 1, DirectionID: 0, StopID: "12TH", ServiceDate: "2018-05-29"},
				{RouteID: "07", DowCat: 1, DirectionID: 1, StopID: "12TH", ServiceDate: "2018-05-29"},
				{RouteID: "07", DowCat: 6, DirectionID: 0, StopID: "12TH", ServiceDate: "2018-05-26"},
				{RouteID: "03", DowCat: 1, DirectionID: 0, StopID: "12TH", ServiceDate: "2018-05-29"},
				{RouteID: "03", DowCat: 1, DirectionID: 1, StopID: "12TH", ServiceDate: "2018-05-29"},
				{RouteID: "03", DowCat: 6, DirectionID: 0, StopID: "12TH", ServiceDate: "2018-05-26"},
				{RouteID: "05", DowCat: 1, DirectionID: 0, StopID: "FRMT", ServiceDate: "2018-05-29"},
				{RouteID: "05", DowCat: 1, DirectionID: 1, StopID: "16TH", ServiceDate: "2018-05-29"},
				{RouteID: "05", DowCat: 6, DirectionID: 0, StopID: "16TH", ServiceDate: "2018-05-26"},
				{RouteID: "11", DowCat: 1, DirectionID: 0, StopID: "BAYF", ServiceDate: "2018-05-29"},
				{RouteID: "11", DowCat: 1, DirectionID: 1, StopID: "BAYF", ServiceDate: "2018-05-29"},
				{RouteID: "11", DowCat: 6, DirectionID: 0, StopID: "16TH", ServiceDate: "2018-05-26"},
				{RouteID: "11", DowCat: 7, DirectionID: 1, StopID: "BAYF", ServiceDate: "2018-05-27"},
				{RouteID: "19", DowCat: 1, DirectionID: 0, StopID: "COLS", ServiceDate: "2018-05-29"},
				{RouteID: "19", DowCat: 1, DirectionID: 1, StopID: "COLS", ServiceDate: "2018-05-29"},
				{RouteID: "19", DowCat: 6, DirectionID: 0, StopID: "COLS", ServiceDate: "2018-05-26"},
				{RouteID: "19", DowCat: 7, DirectionID: 0, StopID: "COLS", ServiceDate: "2018-05-27"},
			},
		},
		"TriMet-2Routes": {
			testutil.RelPath("test/data/external/trimet-2routes.zip"),
			[]testcase{
				{RouteID: "193"},
				{RouteID: "200"},
			},
		},
	}
	for groupName, testGroup := range groups {
		t.Run(groupName, func(t *testing.T) {
			cp, writer, err := newMockCopier(testGroup.URL)
			if err != nil {
				t.Fatal(err)
			}
			e := NewRouteHeadwayBuilder()
			cp.AddExtension(e)
			cpr := cp.Copy()
			if cpr.WriteError != nil {
				t.Fatal(err)
			}
			routeHeadways := map[string][]*RouteHeadway{}
			for _, ent := range writer.Reader.OtherList {
				switch v := ent.(type) {
				case *RouteHeadway:
					routeHeadways[v.RouteID] = append(routeHeadways[v.RouteID], v)
				}
			}
			for _, tc := range testGroup.Cases {
				t.Run(fmt.Sprintf("%s-%d-%d", tc.RouteID, tc.DowCat, tc.DirectionID), func(t *testing.T) {
					found := false
					for _, ent := range routeHeadways[tc.RouteID] {
						// fmt.Printf("\t %#v\n", ent)
						if ent.DowCategory.Int == tc.DowCat && ent.DirectionID.Int == tc.DirectionID {
							if found {
								t.Error("found more than one match")
							}
							found = true
							assert.Equal(t, tc.StopID, ent.SelectedStopID)
							assert.Equal(t, tc.ServiceDate, ent.ServiceDate.Time.Format("2006-01-02"))
						}
					}
					if !found {
						t.Error("no match for test case")
					}
				})
			}
		})
	}
}
