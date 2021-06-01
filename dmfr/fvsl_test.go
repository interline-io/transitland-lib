package dmfr

import (
	"encoding/json"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/tlcsv"
)

type msi = map[string]int
type fvsl = FeedVersionServiceLevel

func TestNewFeedVersionServiceLevelsFromReader(t *testing.T) {
	tcs := []struct {
		name         string
		url          string
		expectCounts msi
		expectResult []string
	}{
		{
			"example",
			testutil.ExampleZip.URL,
			msi{"CITY": 4, "AB": 4, "STBA": 4, "": 4},
			[]string{},
		},
		{
			"bart",
			testutil.ExampleFeedBART.URL,
			msi{"01": 12, "11": 12, "03": 12},
			[]string{
				// feed
				`{"ID":0,"RouteID":null,"StartDate":"2018-07-09","EndDate":"2018-09-02","Monday":3394620,"Tuesday":3394620,"Wednesday":3394620,"Thursday":3394620,"Friday":3394620,"Saturday":2147760,"Sunday":1567680,"AgencyName":"","RouteShortName":"","RouteLongName":"","RouteType":0}`,
				`{"ID":0,"RouteID":null,"StartDate":"2018-11-19","EndDate":"2018-11-25","Monday":3394620,"Tuesday":3394620,"Wednesday":3394620,"Thursday":1567680,"Friday":3394620,"Saturday":2147760,"Sunday":1567680,"AgencyName":"","RouteShortName":"","RouteLongName":"","RouteType":0}`,
				// a regular week
				`{"ID":0,"RouteID":"01","StartDate":"2018-11-26","EndDate":"2018-12-23","Monday":1068060,"Tuesday":1068060,"Wednesday":1068060,"Thursday":1068060,"Friday":1068060,"Saturday":720720,"Sunday":643140,"AgencyName":"Bay Area Rapid Transit","RouteShortName":"","RouteLongName":"Antioch - SFIA/Millbrae","RouteType":1}`,
				// thanksgiving
				`{"ID":0,"RouteID":"03","StartDate":"2018-11-19","EndDate":"2018-11-25","Monday":581220,"Tuesday":581220,"Wednesday":581220,"Thursday":403860,"Friday":581220,"Saturday":452460,"Sunday":403860,"AgencyName":"Bay Area Rapid Transit","RouteShortName":"","RouteLongName":"Warm Springs/South Fremont - Richmond","RouteType":1}`,
				// end of feed
				`{"ID":0,"RouteID":"11","StartDate":"2019-07-01","EndDate":"2019-07-07","Monday":577380,"Tuesday":0,"Wednesday":0,"Thursday":0,"Friday":0,"Saturday":0,"Sunday":0,"AgencyName":"Bay Area Rapid Transit","RouteShortName":"","RouteLongName":"Dublin/Pleasanton - Daly City","RouteType":1}`,
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			reader, err := tlcsv.NewReader(tc.url)
			if err != nil {
				panic(err)
			}
			results, err := NewFeedVersionServiceInfosFromReader(reader)
			if err != nil {
				t.Error(err)
			}
			counts := msi{}
			for _, result := range results {
				counts[result.RouteID.String]++
			}
			for k, v := range tc.expectCounts {
				if a := counts[k]; a != v {
					t.Errorf("got %d results for route '%s', expected %d", a, k, v)
				}
			}
			// Check for matches; uses json marshal/unmarshal for comparison and loading.
			for _, check := range tc.expectResult {
				checksl := FeedVersionServiceLevel{}
				if err := json.Unmarshal([]byte(check), &checksl); err != nil {
					t.Error(err)
				}
				match := false
				for _, a := range results {
					if a.RouteID.String == checksl.RouteID.String &&
						a.AgencyName == checksl.AgencyName &&
						a.StartDate.String() == checksl.StartDate.String() &&
						a.EndDate.String() == checksl.EndDate.String() &&
						a.Monday == checksl.Monday &&
						a.Tuesday == checksl.Tuesday &&
						a.Wednesday == checksl.Wednesday &&
						a.Thursday == checksl.Thursday &&
						a.Friday == checksl.Friday &&
						a.Saturday == checksl.Saturday &&
						a.Sunday == checksl.Sunday {
						match = true
					}
				}
				if !match {
					t.Errorf("no match for %#v\n", check)
				}
			}
		})
	}
}
