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
				`{"ID":0,"StartDate":"2018-07-09","EndDate":"2018-09-02","Monday":3394620,"Tuesday":3394620,"Wednesday":3394620,"Thursday":3394620,"Friday":3394620,"Saturday":2147760,"Sunday":1567680}`,
				// a regular week
				`{"ID":0,"StartDate":"2018-11-26","EndDate":"2018-12-23","Monday":3394620,"Tuesday":3394620,"Wednesday":3394620,"Thursday":3394620,"Friday":3394620,"Saturday":2147760,"Sunday":1567680}`,
				// thanksgiving
				`{"ID":0,"StartDate":"2018-11-19","EndDate":"2018-11-25","Monday":3394620,"Tuesday":3394620,"Wednesday":3394620,"Thursday":1567680,"Friday":3394620,"Saturday":2147760,"Sunday":1567680}`,
				// end of feed
				`{"ID":0,"StartDate":"2019-07-01","EndDate":"2019-07-07","Monday":3394620,"Tuesday":0,"Wednesday":0,"Thursday":0,"Friday":0,"Saturday":0,"Sunday":0}`,
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			reader, err := tlcsv.NewReader(tc.url)
			if err != nil {
				t.Fatal(err)
			}
			results, err := NewFeedVersionServiceInfosFromReader(reader)
			if err != nil {
				t.Error(err)
			}
			// Check for matches; uses json marshal/unmarshal for comparison and loading.
			for _, check := range tc.expectResult {
				checksl := FeedVersionServiceLevel{}
				if err := json.Unmarshal([]byte(check), &checksl); err != nil {
					t.Error(err)
				}
				match := false
				for _, a := range results {
					if a.StartDate.String() == checksl.StartDate.String() &&
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
