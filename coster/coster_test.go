package coster

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testFeed is the base feed for tests: one agency, routes on separate networks, and stops
// that include a station with two platforms and a stop in another timezone.
var testFeed = map[string]string{
	"agency.txt": `
agency_id,agency_name,agency_url,agency_timezone
agency,Agency,https://example.com,America/Los_Angeles`,
	"stops.txt": `
stop_id,stop_name,stop_lat,stop_lon,location_type,parent_station,stop_timezone
a,A,37.1,-122.1,,,
b,B,37.2,-122.2,,,
c,C,37.3,-122.3,,,
d,D,37.4,-122.4,,,
east,East,40.7,-74.0,,,America/New_York
station,Station,37.5,-122.5,1,,
platform1,Platform 1,37.5,-122.5,0,station,
platform2,Platform 2,37.5,-122.5,0,station,`,
	"routes.txt": `
route_id,agency_id,route_short_name,route_type,network_id
bus1,agency,1,3,bus
bus2,agency,2,3,bus
rail1,agency,R1,2,rail
rail2,agency,R2,2,rail
ferry,agency,F,4,ferry`,
}

// openTestFeed writes the base feed plus each set of files in turn to a temporary directory,
// and returns an open reader. A file replaces any earlier file with the same name.
func openTestFeed(t *testing.T, feeds ...map[string]string) *tlcsv.Reader {
	t.Helper()
	dir := t.TempDir()
	for _, feed := range append([]map[string]string{testFeed}, feeds...) {
		for name, body := range feed {
			require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(strings.TrimSpace(body)+"\n"), 0o644))
		}
	}
	reader, err := tlcsv.NewReader(dir)
	require.NoError(t, err)
	require.NoError(t, reader.Open())
	t.Cleanup(func() { reader.Close() })
	return reader
}

// newTestCoster loads a Coster from the base feed plus each set of files.
func newTestCoster(t *testing.T, feeds ...map[string]string) *Coster {
	t.Helper()
	c, err := New(openTestFeed(t, feeds...))
	require.NoError(t, err)
	return c
}

// pdt is the base feed's timezone in October 2026.
var pdt = time.FixedZone("PDT", -7*60*60)

// oct returns a time in October 2026 in the base feed's timezone.
func oct(day int, hour int, minute int) time.Time {
	return time.Date(2026, time.October, day, hour, minute, 0, 0, pdt)
}

// at returns a time on Monday, October 5, 2026.
func at(hour int, minute int) time.Time {
	return oct(5, hour, minute)
}

func leg(route string, from string, to string, departure time.Time, arrival time.Time) Leg {
	return Leg{RouteID: route, FromStopID: from, ToStopID: to, DepartureTime: departure, ArrivalTime: arrival}
}

func journey(legs ...Leg) Journey {
	return Journey{Legs: legs}
}

// fareAmounts returns the amount of the journey's fare for each fare medium.
func fareAmounts(t *testing.T, c *Coster, j Journey) map[string]float64 {
	t.Helper()
	fares, err := c.Fares(j)
	require.NoError(t, err)
	ret := map[string]float64{}
	for _, fare := range fares {
		ret[fare.FareMediaID] = fare.Amount
	}
	return ret
}

func TestFares_RiderCategories(t *testing.T) {
	c := newTestCoster(t, map[string]string{
		"rider_categories.txt": `
rider_category_id,rider_category_name,is_default_fare_category
adult,Adult,1
senior,Senior,0
youth,Youth,0`,
		"fare_products.txt": `
fare_product_id,amount,currency,rider_category_id
bus_fare,2.00,USD,adult
bus_fare,1.00,USD,senior
rail_fare,4.00,USD,`,
		"fare_leg_rules.txt": `
network_id,fare_product_id
bus,bus_fare
rail,rail_fare`,
	})
	bus := leg("bus1", "a", "b", at(8, 0), at(8, 20))
	rail := leg("rail1", "a", "b", at(8, 0), at(8, 20))
	tcs := []struct {
		name       string
		leg        Leg
		categories []string
		want       map[string]float64
	}{
		{"default category", bus, nil, map[string]float64{"": 2.00}},
		{"listed category", bus, []string{"senior"}, map[string]float64{"": 1.00}},
		{"cheapest of several categories", bus, []string{"adult", "senior"}, map[string]float64{"": 1.00}},
		{"no product for the category", bus, []string{"youth"}, map[string]float64{}},
		{"product without a category", rail, []string{"youth"}, map[string]float64{"": 4.00}},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			j := journey(tc.leg)
			j.RiderCategoryIDs = tc.categories
			assert.Equal(t, tc.want, fareAmounts(t, c, j))
		})
	}
	t.Run("unknown category", func(t *testing.T) {
		j := journey(bus)
		j.RiderCategoryIDs = []string{"student"}
		_, err := c.Fares(j)
		assert.ErrorContains(t, err, `unknown rider category "student"`)
	})
}

func TestFares_FareMedia(t *testing.T) {
	c := newTestCoster(t, map[string]string{
		"fare_media.txt": `
fare_media_id,fare_media_name,fare_media_type
cash,Cash,0
card,Card,2`,
		"fare_products.txt": `
fare_product_id,amount,currency,fare_media_id
bus_fare,2.00,USD,card
bus_fare,2.50,USD,cash
rail_fare,4.00,USD,`,
		"fare_leg_rules.txt": `
network_id,fare_product_id
bus,bus_fare
rail,rail_fare`,
	})
	bus := leg("bus1", "a", "b", at(8, 0), at(8, 20))
	t.Run("one fare per fare medium, cheapest first", func(t *testing.T) {
		fares, err := c.Fares(journey(bus))
		require.NoError(t, err)
		require.Len(t, fares, 2)
		assert.Equal(t, "card", fares[0].FareMediaID)
		assert.Equal(t, 2.00, fares[0].Amount)
		assert.Equal(t, "cash", fares[1].FareMediaID)
		assert.Equal(t, 2.50, fares[1].Amount)
	})
	t.Run("listed fare media", func(t *testing.T) {
		j := journey(bus)
		j.FareMediaIDs = []string{"cash"}
		assert.Equal(t, map[string]float64{"cash": 2.50}, fareAmounts(t, c, j))
	})
	t.Run("product without a fare medium", func(t *testing.T) {
		j := journey(leg("rail1", "a", "b", at(8, 0), at(8, 20)))
		assert.Equal(t, map[string]float64{"card": 4.00, "cash": 4.00}, fareAmounts(t, c, j))
	})
	t.Run("unknown fare medium", func(t *testing.T) {
		j := journey(bus)
		j.FareMediaIDs = []string{"token"}
		_, err := c.Fares(j)
		assert.ErrorContains(t, err, `unknown fare media "token"`)
	})
}

func TestFares_FareLegs(t *testing.T) {
	c := newTestCoster(t, transferFeed, map[string]string{
		"fare_transfer_rules.txt": `
from_leg_group_id,to_leg_group_id,fare_transfer_type,fare_product_id
bus,rail,0,ab`,
	})
	fares, err := c.Fares(journey(
		leg("bus1", "a", "b", at(8, 0), at(8, 20)),
		leg("rail1", "b", "c", at(8, 30), at(8, 50)),
	))
	require.NoError(t, err)
	require.Len(t, fares, 1)
	fare := fares[0]
	assert.Equal(t, 2.50, fare.Amount)
	assert.Equal(t, "USD", fare.Currency)
	require.Len(t, fare.FareLegs, 2)

	first := fare.FareLegs[0]
	assert.Equal(t, []int{0}, first.LegIndexes)
	assert.Equal(t, "bus", first.LegRule.LegGroupID.Val)
	assert.Equal(t, "bus_fare", first.Product.FareProductID.Val)
	assert.Nil(t, first.TransferRule)
	assert.Equal(t, 2.00, first.Amount)

	second := fare.FareLegs[1]
	assert.Equal(t, []int{1}, second.LegIndexes)
	assert.Equal(t, "rail_fare", second.Product.FareProductID.Val)
	require.NotNil(t, second.TransferRule)
	assert.Equal(t, "bus", second.TransferRule.FromLegGroupID.Val)
	assert.Equal(t, "ab", second.TransferProduct.FareProductID.Val)
	assert.Equal(t, 0.50, second.Amount)
}

func TestFares_Errors(t *testing.T) {
	c := newTestCoster(t)
	tcs := []struct {
		name    string
		journey Journey
		err     string
	}{
		{"no legs", journey(), "journey has no legs"},
		{"unknown route", journey(leg("tram", "a", "b", at(8, 0), at(8, 20))), `leg 0: unknown route "tram"`},
		{"unknown stop", journey(leg("bus1", "a", "z", at(8, 0), at(8, 20))), `leg 0: unknown stop "z"`},
		{"missing time", journey(leg("bus1", "a", "b", at(8, 0), time.Time{})), "leg 0: departure and arrival times are required"},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			_, err := c.Fares(tc.journey)
			assert.EqualError(t, err, tc.err)
		})
	}
}
