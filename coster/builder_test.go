package coster

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/adapters/empty"
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// builderFeed covers data that a copy delivers differently from a reader: calendars and
// their dates, a service only in calendar_dates.txt, stop areas, and fare media.
var builderFeed = map[string]string{
	"calendar.txt": `
service_id,monday,tuesday,wednesday,thursday,friday,saturday,sunday,start_date,end_date
weekdays,1,1,1,1,1,0,0,20260101,20261231`,
	"calendar_dates.txt": `
service_id,date,exception_type
weekdays,20261012,2
holidays,20261012,1`,
	"timeframes.txt": `
timeframe_group_id,start_time,end_time,service_id
regular,,,weekdays
holiday,,,holidays`,
	"fare_media.txt": `
fare_media_id,fare_media_name,fare_media_type
card,Card,2
cash,Cash,0`,
	"rider_categories.txt": `
rider_category_id,rider_category_name,is_default_fare_category
adult,Adult,1`,
	"fare_products.txt": `
fare_product_id,amount,currency,rider_category_id,fare_media_id
rail_fare,4.00,USD,adult,card
rail_fare,4.50,USD,adult,cash
holiday_fare,1.00,USD,,
bus_fare,2.00,USD,,
card_transfer,0.00,USD,,card`,
	"areas.txt": `
area_id
zone1
zone2`,
	"stop_areas.txt": `
area_id,stop_id
zone1,a
zone1,station
zone2,d`,
	"fare_leg_rules.txt": `
leg_group_id,network_id,from_area_id,to_area_id,from_timeframe_group_id,fare_product_id
rail,rail,zone1,zone2,regular,rail_fare
rail,rail,zone1,zone2,holiday,holiday_fare
bus,bus,,,,bus_fare`,
	"fare_leg_join_rules.txt": `
from_network_id,to_network_id
rail,rail`,
	"fare_transfer_rules.txt": `
from_leg_group_id,to_leg_group_id,fare_transfer_type,fare_product_id
bus,rail,0,card_transfer`,
}

// copyTestCoster copies the base feed plus each set of files to writer, and returns the
// Coster that a Builder collected from the copy.
func copyTestCoster(t *testing.T, writer adapters.Writer, feeds ...map[string]string) *Coster {
	t.Helper()
	b := NewBuilder()
	opts := copier.Options{}
	opts.AddExtension(b)
	_, err := copier.CopyWithOptions(context.Background(), openTestFeed(t, feeds...), writer, opts)
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
	fromReader := newTestCoster(t, builderFeed)
	fromCopy := copyTestCoster(t, &empty.Writer{}, builderFeed)
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
			// October 12 drops the weekday service and adds the holiday service.
			"holiday",
			journey(leg("rail1", "platform1", "d", oct(12, 8, 0), oct(12, 8, 30))),
			map[string]float64{"card": 1.00, "cash": 1.00},
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
	c := copyTestCoster(t, &idWriter{}, builderFeed)
	joined := journey(
		leg("w:rail1", "w:a", "w:platform1", at(8, 0), at(8, 20)),
		leg("w:rail2", "w:platform2", "w:d", at(8, 30), at(8, 50)),
	)
	assert.Equal(t, map[string]float64{"w:card": 4.00, "w:cash": 4.50}, fareAmounts(t, c, joined))
	holiday := journey(leg("w:rail1", "w:platform1", "w:d", oct(12, 8, 0), oct(12, 8, 30)))
	assert.Equal(t, map[string]float64{"w:card": 1.00, "w:cash": 1.00}, fareAmounts(t, c, holiday))
}
