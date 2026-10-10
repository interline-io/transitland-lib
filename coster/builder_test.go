package coster

import (
	"context"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/adapters/direct"
	"github.com/interline-io/transitland-lib/adapters/empty"
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// builderFeed covers data that a copy delivers differently from a reader: calendars and
// their dates, a service only in calendar_dates.txt, stop areas, and fare media.
func builderFeed() *direct.Reader {
	feed := baseFeed()
	feed.CalendarList = []gtfs.Calendar{
		calendar("weekdays", time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday),
	}
	// October 12 drops the weekday service and adds the holiday service.
	holiday := tt.NewDate(time.Date(2026, time.October, 12, 0, 0, 0, 0, time.UTC))
	feed.CalendarDateList = []gtfs.CalendarDate{
		{ServiceID: key("weekdays"), Date: holiday, ExceptionType: tt.NewInt(2)},
		{ServiceID: key("holidays"), Date: holiday, ExceptionType: tt.NewInt(1)},
	}
	feed.TimeframeList = []gtfs.Timeframe{
		{TimeframeGroupID: str("regular"), ServiceID: key("weekdays")},
		{TimeframeGroupID: str("holiday"), ServiceID: key("holidays")},
	}
	feed.FareMediaList = []gtfs.FareMedia{medium("card", 2), medium("cash", 0)}
	feed.RiderCategoryList = []gtfs.RiderCategory{category("adult", true)}
	feed.FareProductList = []gtfs.FareProduct{
		fareProduct("rail_fare", 4.00, "adult", "card"),
		fareProduct("rail_fare", 4.50, "adult", "cash"),
		fareProduct("holiday_fare", 6.00, "", ""),
		fareProduct("bus_fare", 2.00, "", ""),
		fareProduct("card_transfer", 0.00, "", "card"),
	}
	feed.AreaList = areas("zone1", "zone2")
	feed.StopAreaList = []gtfs.StopArea{stopArea("zone1", "a"), stopArea("zone1", "station"), stopArea("zone2", "d")}
	feed.FareLegRuleList = []gtfs.FareLegRule{
		{LegGroupID: str("rail"), NetworkID: str("rail"), FromAreaID: str("zone1"), ToAreaID: str("zone2"), FromTimeframeGroupID: str("regular"), FareProductID: str("rail_fare")},
		{LegGroupID: str("rail"), NetworkID: str("rail"), FromAreaID: str("zone1"), ToAreaID: str("zone2"), FromTimeframeGroupID: str("holiday"), FareProductID: str("holiday_fare")},
		legRule("bus", "bus", "bus_fare"),
	}
	feed.FareLegJoinRuleList = []gtfs.FareLegJoinRule{{FromNetworkID: str("rail"), ToNetworkID: str("rail")}}
	feed.FareTransferRuleList = []gtfs.FareTransferRule{transferRule("bus", "rail", 0, "card_transfer")}
	return feed
}

// copyTestCoster copies a test feed to writer, and returns the Coster that a Builder
// collected from the copy.
func copyTestCoster(t *testing.T, writer adapters.Writer, feed *direct.Reader) *Coster {
	t.Helper()
	b := NewBuilder()
	opts := copier.Options{}
	opts.AddExtension(b)
	_, err := copier.CopyWithOptions(context.Background(), feed, writer, opts)
	require.NoError(t, err)
	c, err := b.Coster()
	require.NoError(t, err)
	return c
}

// idWriter writes nothing and returns new IDs, as a database writer does.
type idWriter struct {
	empty.Writer
}

func (w *idWriter) AddEntity(ent tt.Entity) (string, error) {
	return "w:" + ent.EntityID(), nil
}

func (w *idWriter) AddEntities(ents []tt.Entity) ([]string, error) {
	var ret []string
	for _, ent := range ents {
		ret = append(ret, "w:"+ent.EntityID())
	}
	return ret, nil
}

func TestBuilder_Copy(t *testing.T) {
	// A Builder that collects from a copy prices journeys the same way as New.
	fromReader := newTestCoster(t, builderFeed())
	fromCopy := copyTestCoster(t, &empty.Writer{}, builderFeed())
	tcs := []struct {
		name    string
		journey Journey
		want    map[string]float64
	}{
		{
			"weekday",
			journey(leg("rail1", "platform1", "d", at(8, 0), at(8, 30))),
			map[string]float64{"card": 4.00, "cash": 4.50},
		},
		{
			// The holiday fare costs more, so it wins only if the weekday service is gone.
			"holiday",
			journey(leg("rail1", "platform1", "d", oct(12, 8, 0), oct(12, 8, 30))),
			map[string]float64{"card": 6.00, "cash": 6.00},
		},
		{
			"joined legs",
			journey(
				leg("rail1", "a", "platform1", at(8, 0), at(8, 20)),
				leg("rail2", "platform2", "d", at(8, 30), at(8, 50)),
			),
			map[string]float64{"card": 4.00, "cash": 4.50},
		},
		{
			"transfer for card riders",
			journey(
				leg("bus1", "c", "platform1", at(8, 0), at(8, 20)),
				leg("rail1", "platform1", "d", at(8, 30), at(8, 50)),
			),
			map[string]float64{"card": 2.00, "cash": 6.50},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, fareAmounts(t, fromReader, tc.journey))
			assert.Equal(t, tc.want, fareAmounts(t, fromCopy, tc.journey))
		})
	}
}

func TestBuilder_WriterIDs(t *testing.T) {
	// The Builder keys routes, stops, calendars, and fare media by the writer's IDs.
	c := copyTestCoster(t, &idWriter{}, builderFeed())
	joined := journey(
		leg("w:rail1", "w:a", "w:platform1", at(8, 0), at(8, 20)),
		leg("w:rail2", "w:platform2", "w:d", at(8, 30), at(8, 50)),
	)
	assert.Equal(t, map[string]float64{"w:card": 4.00, "w:cash": 4.50}, fareAmounts(t, c, joined))
	holiday := journey(leg("w:rail1", "w:platform1", "w:d", oct(12, 8, 0), oct(12, 8, 30)))
	assert.Equal(t, map[string]float64{"w:card": 6.00, "w:cash": 6.00}, fareAmounts(t, c, holiday))
}

func TestBuilder_TimezoneCase(t *testing.T) {
	// tlib accepts a timezone name in any case.
	feed := timeframeFeed()
	feed.AgencyList[0].AgencyTimezone = tt.NewTimezone("america/los_angeles")
	feed.FareLegRuleList = []gtfs.FareLegRule{{NetworkID: str("rail"), FromTimeframeGroupID: str("peak"), FareProductID: str("peak_fare")}}
	c := newTestCoster(t, feed)
	assert.Equal(t, map[string]float64{"": 5.00}, fareAmounts(t, c, journey(leg("rail1", "c", "d", at(8, 0), at(8, 30)))))
	// A case-insensitive file system can load the name as written, so check for the canonical name.
	loc, err := c.location("c")
	require.NoError(t, err)
	assert.Equal(t, "America/Los_Angeles", loc.String())
}
