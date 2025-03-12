package stats

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/adapters/empty"
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/stretchr/testify/assert"
)

func NewFeedVersionServiceWindowFromReader(reader adapters.Reader) (dmfr.FeedVersionServiceWindow, error) {
	ret := dmfr.FeedVersionServiceWindow{}
	fvswBuilder := NewFeedVersionServiceWindowBuilder()
	if _, err := copier.QuietCopy(
		context.TODO(),
		reader,
		&empty.Writer{},
		func(o *copier.Options) {
			o.AddExtension(fvswBuilder)
		},
	); err != nil {
		return ret, err
	}
	ret, err := fvswBuilder.ServiceWindow()
	if err != nil {
		return ret, err
	}
	return ret, nil
}

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
