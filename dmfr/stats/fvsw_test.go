package stats

import (
	"testing"

	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/tl/tt"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/stretchr/testify/assert"
)

func TestNewFeedVersionServiceWindowsFromReader(t *testing.T) {
	tcs := []struct {
		name                  string
		url                   string
		feedStartDate         tt.Date
		feedEndDate           tt.Date
		expectFallbackWeek    tt.Date
		expectDefaultTimezone string
	}{
		{
			"example",
			testutil.ExampleZip.URL,
			pd(""),
			pd(""),
			pd("2007-01-01"),
			"America/Los_Angeles",
		},
		{
			"bart",
			testutil.ExampleFeedBART.URL,
			pd("2018-05-26"),
			pd("2019-07-01"),
			pd("2018-06-04"),
			"America/Los_Angeles",
		},
		{
			"caltrain",
			testutil.ExampleFeedCaltrain.URL,
			pd(""),
			pd(""),
			pd("2018-06-18"),
			"America/Los_Angeles",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			reader, err := tlcsv.NewReader(tc.url)
			if err != nil {
				t.Fatal(err)
			}
			fvsls, err := NewFeedVersionServiceLevelsFromReader(reader)
			if err != nil {
				t.Error(err)
			}
			fvsw, err := NewFeedVersionServiceWindowFromReader(reader)
			if err != nil {
				t.Error(err)
			}

			if d, err := ServiceLevelDefaultWeek(fvsw.FeedStartDate, fvsw.FeedEndDate, fvsls); err != nil {
				t.Error(err)
			} else {
				fvsw.FallbackWeek = d
			}
			// t.Log("fvsw:", fvsw.FeedStartDate)
			assert.EqualValues(t, tc.feedStartDate.String(), fvsw.FeedStartDate.String())
			assert.EqualValues(t, tc.feedEndDate.String(), fvsw.FeedEndDate.String())
			assert.EqualValues(t, tc.expectFallbackWeek.String(), fvsw.FallbackWeek.String())
			assert.EqualValues(t, tc.expectDefaultTimezone, fvsw.DefaultTimezone.Val)
		})
	}
}
