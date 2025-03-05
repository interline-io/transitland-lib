package builders

import (
	"fmt"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testpath"
	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestRouteHeadwayBuilder(t *testing.T) {
	type testcase struct {
		RouteID       string
		DowCat        int64
		DirectionID   int64
		StopID        string
		ServiceDate   string
		HeadwaySecs   int64
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
				{RouteID: "Bu-130", DowCat: 1, DirectionID: 0, StopID: "70011", ServiceDate: "2017-10-02"},
				{RouteID: "Lo-130", DowCat: 1, DirectionID: 0, StopID: "70011", ServiceDate: "2017-10-02"},
			},
		},
		"BART": {
			testutil.ExampleFeedBART.URL,
			[]testcase{
				{RouteID: "07", DowCat: 1, DirectionID: 0, StopID: "12TH", ServiceDate: "2018-05-29", HeadwaySecs: 900},
				{RouteID: "07", DowCat: 1, DirectionID: 1, StopID: "12TH", ServiceDate: "2018-05-29", HeadwaySecs: 900},
				{RouteID: "07", DowCat: 6, DirectionID: 0, StopID: "12TH", ServiceDate: "2018-05-26", HeadwaySecs: 1200},
				{RouteID: "03", DowCat: 1, DirectionID: 0, StopID: "12TH", ServiceDate: "2018-05-29", HeadwaySecs: 900},
				{RouteID: "03", DowCat: 1, DirectionID: 1, StopID: "12TH", ServiceDate: "2018-05-29", HeadwaySecs: 900},
				{RouteID: "03", DowCat: 6, DirectionID: 0, StopID: "12TH", ServiceDate: "2018-05-26", HeadwaySecs: 1200},
				{RouteID: "05", DowCat: 1, DirectionID: 0, StopID: "FRMT", ServiceDate: "2018-05-29", HeadwaySecs: 900},
				{RouteID: "05", DowCat: 1, DirectionID: 1, StopID: "16TH", ServiceDate: "2018-05-29", HeadwaySecs: 900},
				{RouteID: "05", DowCat: 6, DirectionID: 0, StopID: "16TH", ServiceDate: "2018-05-26"},
				{RouteID: "11", DowCat: 1, DirectionID: 0, StopID: "BAYF", ServiceDate: "2018-05-29", HeadwaySecs: 900},
				{RouteID: "11", DowCat: 1, DirectionID: 1, StopID: "BAYF", ServiceDate: "2018-05-29", HeadwaySecs: 900},
				{RouteID: "11", DowCat: 6, DirectionID: 0, StopID: "16TH", ServiceDate: "2018-05-26", HeadwaySecs: 1200},
				{RouteID: "11", DowCat: 7, DirectionID: 1, StopID: "BAYF", ServiceDate: "2018-05-27", HeadwaySecs: 1200},
				{RouteID: "19", DowCat: 1, DirectionID: 0, StopID: "COLS", ServiceDate: "2018-05-29", HeadwaySecs: 360},
				{RouteID: "19", DowCat: 1, DirectionID: 1, StopID: "COLS", ServiceDate: "2018-05-29", HeadwaySecs: 360},
				{RouteID: "19", DowCat: 6, DirectionID: 0, StopID: "COLS", ServiceDate: "2018-05-26", HeadwaySecs: 360},
				{RouteID: "19", DowCat: 7, DirectionID: 0, StopID: "COLS", ServiceDate: "2018-05-27", HeadwaySecs: 360},
			},
		},
		"TriMet-2Routes": {
			testpath.RelPath("testdata/external/trimet-2routes.zip"),
			[]testcase{
				{RouteID: "193", DowCat: 1, DirectionID: 0, StopID: "10776", ServiceDate: "2021-10-18", HeadwaySecs: 960},
				{RouteID: "200", DowCat: 1, DirectionID: 0, StopID: "10293", ServiceDate: "2021-10-25", HeadwaySecs: 900},
			},
		},
	}
	for groupName, testGroup := range groups {
		t.Run(groupName, func(t *testing.T) {
			e := NewRouteHeadwayBuilder()
			cp, writer, err := newMockCopier(testGroup.URL, e)
			if err != nil {
				t.Fatal(err)
			}
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
						if ent.DowCategory.Val == tc.DowCat && ent.DirectionID.Val == tc.DirectionID {
							if found {
								t.Error("found more than one match")
							}
							found = true
							assert.Equal(t, tc.StopID, ent.SelectedStopID)
							assert.Equal(t, tc.ServiceDate, ent.ServiceDate.Format("2006-01-02"))
							if tc.HeadwaySecs > 0 {
								assert.Equal(t, tc.HeadwaySecs, ent.HeadwaySecs.Val)
							}
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
