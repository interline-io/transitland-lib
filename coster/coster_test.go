package coster

import (
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/adapters/direct"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// baseFeed returns the base feed for tests: one agency, routes on separate networks, and
// stops that include a station with platforms and stops in another timezone.
func baseFeed() *direct.Reader {
	return &direct.Reader{
		AgencyList: []gtfs.Agency{{
			AgencyID:       str("agency"),
			AgencyName:     str("Agency"),
			AgencyURL:      tt.NewUrl("https://example.com"),
			AgencyTimezone: tt.NewTimezone("America/Los_Angeles"),
		}},
		StopList: []gtfs.Stop{
			stop("a", 37.1, -122.1, 0, "", ""),
			stop("b", 37.2, -122.2, 0, "", ""),
			stop("c", 37.3, -122.3, 0, "", ""),
			stop("d", 37.4, -122.4, 0, "", ""),
			stop("east", 40.7, -74.0, 0, "", "America/New_York"),
			stop("east_station", 40.7, -74.0, 1, "", "America/New_York"),
			stop("east_platform", 40.7, -74.0, 0, "east_station", ""),
			stop("east_boarding", 40.7, -74.0, 4, "east_platform", ""),
			stop("station", 37.5, -122.5, 1, "", ""),
			stop("platform1", 37.5, -122.5, 0, "station", ""),
			stop("platform2", 37.5, -122.5, 0, "station", ""),
			stop("platform3", 37.5, -122.5, 0, "station", "America/New_York"),
		},
		RouteList: []gtfs.Route{
			route("bus1", 3, "bus"),
			route("bus2", 3, "bus"),
			route("rail1", 2, "rail"),
			route("rail2", 2, "rail"),
			route("ferry", 4, "ferry"),
		},
	}
}

// str returns a tt.String that is unset when v is empty, as for an empty CSV field.
func str(v string) tt.String {
	if v == "" {
		return tt.String{}
	}
	return tt.NewString(v)
}

// key returns a tt.Key that is unset when v is empty, as for an empty CSV field.
func key(v string) tt.Key {
	if v == "" {
		return tt.Key{}
	}
	return tt.NewKey(v)
}

// stop returns a stop. Its parent station and timezone may be empty.
func stop(id string, lat float64, lon float64, locationType int, parent string, timezone string) gtfs.Stop {
	ret := gtfs.Stop{
		StopID:        str(id),
		StopName:      str(id),
		Geometry:      tt.NewPoint(lon, lat),
		LocationType:  tt.NewInt(locationType),
		ParentStation: key(parent),
	}
	if timezone != "" {
		ret.StopTimezone = tt.NewTimezone(timezone)
	}
	return ret
}

// route returns a route of the base feed's agency.
func route(id string, routeType int, network string) gtfs.Route {
	return gtfs.Route{RouteID: str(id), AgencyID: key("agency"), RouteShortName: str(id), RouteType: tt.NewInt(routeType), NetworkID: str(network)}
}

// fareProduct returns a fare product row in USD. Its rider category and fare medium may be
// empty.
func fareProduct(id string, amount float64, category string, media string) gtfs.FareProduct {
	return gtfs.FareProduct{
		FareProductID:   str(id),
		Amount:          tt.NewCurrencyAmount(amount),
		Currency:        tt.NewCurrency("USD"),
		RiderCategoryID: key(category),
		FareMediaID:     key(media),
	}
}

// pass returns a fare product row with the draft duration fields: a pass for one day.
func pass(id string, amount float64, category string) gtfs.FareProduct {
	ret := fareProduct(id, amount, category, "")
	ret.DurationAmount, ret.DurationType, ret.DurationUnit = tt.NewFloat(1), tt.NewInt(1), tt.NewInt(3)
	return ret
}

// legRule returns a fare leg rule for a network. Its leg group and network may be empty.
func legRule(group string, network string, product string) gtfs.FareLegRule {
	return gtfs.FareLegRule{LegGroupID: str(group), NetworkID: str(network), FareProductID: str(product)}
}

// areaRule returns a fare leg rule for a network between two areas, either of which may
// be empty.
func areaRule(network string, from string, to string, product string) gtfs.FareLegRule {
	return gtfs.FareLegRule{NetworkID: str(network), FromAreaID: str(from), ToAreaID: str(to), FareProductID: str(product)}
}

// transferRule returns a fare transfer rule. Its leg groups and product may be empty.
func transferRule(from string, to string, transferType int, product string) gtfs.FareTransferRule {
	return gtfs.FareTransferRule{FromLegGroupID: str(from), ToLegGroupID: str(to), FareTransferType: tt.NewInt(transferType), FareProductID: str(product)}
}

// areas returns an area for each ID.
func areas(ids ...string) []gtfs.Area {
	var ret []gtfs.Area
	for _, id := range ids {
		ret = append(ret, gtfs.Area{AreaID: str(id)})
	}
	return ret
}

// stopArea returns a stop's membership in an area.
func stopArea(area string, stop string) gtfs.StopArea {
	return gtfs.StopArea{AreaID: key(area), StopID: key(stop)}
}

// medium returns a fare medium.
func medium(id string, mediaType int) gtfs.FareMedia {
	return gtfs.FareMedia{FareMediaID: str(id), FareMediaName: str(id), FareMediaType: tt.NewInt(mediaType)}
}

// category returns a rider category.
func category(id string, isDefault bool) gtfs.RiderCategory {
	ret := gtfs.RiderCategory{RiderCategoryID: str(id), RiderCategoryName: str(id), IsDefaultFareCategory: tt.NewInt(0)}
	if isDefault {
		ret.IsDefaultFareCategory = tt.NewInt(1)
	}
	return ret
}

// calendar returns a service on the given days of the week during 2026.
func calendar(id string, days ...time.Weekday) gtfs.Calendar {
	ret := gtfs.Calendar{
		ServiceID: str(id),
		StartDate: tt.NewDate(time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)),
		EndDate:   tt.NewDate(time.Date(2026, time.December, 31, 0, 0, 0, 0, time.UTC)),
	}
	byDay := [7]*tt.Int{&ret.Sunday, &ret.Monday, &ret.Tuesday, &ret.Wednesday, &ret.Thursday, &ret.Friday, &ret.Saturday}
	for _, f := range byDay {
		*f = tt.NewInt(0)
	}
	for _, d := range days {
		*byDay[d] = tt.NewInt(1)
	}
	return ret
}

// timeframe returns a timeframe from one hour of the day to another.
func timeframe(group string, startHour int, endHour int, service string) gtfs.Timeframe {
	return gtfs.Timeframe{
		TimeframeGroupID: str(group),
		StartTime:        tt.NewSeconds(startHour * 3600),
		EndTime:          tt.NewSeconds(endHour * 3600),
		ServiceID:        key(service),
	}
}

// newTestCoster loads a Coster from a test feed.
func newTestCoster(t *testing.T, feed *direct.Reader) *Coster {
	t.Helper()
	c, err := New(feed)
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

// fareAmounts returns the lowest fare for the journey paid entirely with each fare medium:
// the journey's fare media, or else every fare medium in the feed.
func fareAmounts(t *testing.T, c *Coster, j Journey) map[string]float64 {
	t.Helper()
	media := j.FareMediaIDs
	if len(media) == 0 {
		media = c.fareMediaIDs
	}
	ret := map[string]float64{}
	if len(media) == 0 {
		// A feed without fare media has one empty option.
		media = []string{""}
	}
	for _, m := range media {
		single := j
		single.FareMediaIDs = nil
		if m != "" {
			single.FareMediaIDs = []string{m}
		}
		fare, err := c.LowestFare(single)
		require.NoError(t, err)
		if fare != nil {
			ret[m] = fare.Amount.InexactFloat64()
		}
	}
	return ret
}

func TestLowestFare_RiderCategories(t *testing.T) {
	feed := baseFeed()
	feed.RiderCategoryList = []gtfs.RiderCategory{category("adult", true), category("senior", false), category("youth", false)}
	feed.FareProductList = []gtfs.FareProduct{
		fareProduct("bus_fare", 2.00, "adult", ""),
		fareProduct("bus_fare", 1.00, "senior", ""),
		fareProduct("rail_fare", 4.00, "", ""),
	}
	feed.FareLegRuleList = []gtfs.FareLegRule{legRule("", "bus", "bus_fare"), legRule("", "rail", "rail_fare")}
	c := newTestCoster(t, feed)
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
		_, err := c.LowestFare(j)
		assert.ErrorContains(t, err, `unknown rider category "student"`)
	})
}

func TestLowestFare_FareMedia(t *testing.T) {
	feed := baseFeed()
	feed.FareMediaList = []gtfs.FareMedia{medium("cash", 0), medium("card", 2)}
	feed.FareProductList = []gtfs.FareProduct{
		fareProduct("bus_fare", 2.00, "", "card"),
		fareProduct("bus_fare", 2.50, "", "cash"),
		fareProduct("rail_fare", 4.00, "", ""),
	}
	feed.FareLegRuleList = []gtfs.FareLegRule{legRule("", "bus", "bus_fare"), legRule("", "rail", "rail_fare")}
	c := newTestCoster(t, feed)
	bus := leg("bus1", "a", "b", at(8, 0), at(8, 20))
	t.Run("cheapest fare medium", func(t *testing.T) {
		fare, err := c.LowestFare(journey(bus))
		require.NoError(t, err)
		require.NotNil(t, fare)
		assert.Equal(t, 2.00, fare.Amount.InexactFloat64())
		assert.Equal(t, "card", fare.FareLegs[0].Product.FareMediaID.Val)
		assert.Equal(t, "card", fare.FareLegs[0].FareMediaID)
	})
	t.Run("one fare medium at a time", func(t *testing.T) {
		assert.Equal(t, map[string]float64{"card": 2.00, "cash": 2.50}, fareAmounts(t, c, journey(bus)))
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
		_, err := c.LowestFare(j)
		assert.ErrorContains(t, err, `unknown fare media "token"`)
	})
}

func TestLowestFare_Passes(t *testing.T) {
	// A product with the draft duration fields is a pass, which never prices a trip.
	feed := baseFeed()
	feed.FareProductList = []gtfs.FareProduct{
		fareProduct("bus_fare", 2.00, "", ""),
		pass("bus_pass", 1.00, ""),
		pass("rail_pass", 5.00, ""),
	}
	feed.FareLegRuleList = []gtfs.FareLegRule{
		legRule("", "bus", "bus_fare"),
		legRule("", "bus", "bus_pass"),
		legRule("", "rail", "rail_pass"),
	}
	c := newTestCoster(t, feed)
	t.Run("single ride even when a pass costs less", func(t *testing.T) {
		assert.Equal(t, map[string]float64{"": 2.00}, fareAmounts(t, c, journey(leg("bus1", "a", "b", at(8, 0), at(8, 20)))))
	})
	t.Run("no fare when only a pass covers the leg", func(t *testing.T) {
		fare, err := c.LowestFare(journey(leg("rail1", "a", "b", at(8, 0), at(8, 20))))
		require.NoError(t, err)
		assert.Nil(t, fare)
	})
	t.Run("not even when the rider holds the pass", func(t *testing.T) {
		j := journey(leg("bus1", "a", "b", at(8, 0), at(8, 20)))
		j.FareProductIDs = []string{"bus_pass"}
		assert.Equal(t, map[string]float64{"": 2.00}, fareAmounts(t, c, j))
	})
}

func TestLowestFare_HeldProducts(t *testing.T) {
	// A held product doesn't change the fare, but each fare leg lists the held products that
	// may cover the leg.
	feed := baseFeed()
	feed.RiderCategoryList = []gtfs.RiderCategory{category("adult", true), category("smd", false)}
	feed.FareProductList = []gtfs.FareProduct{
		fareProduct("bus_fare", 2.00, "adult", ""),
		fareProduct("bus_fare", 1.00, "smd", ""),
		pass("bus_pass", 80.00, "adult"),
		pass("senior_pass", 40.00, "smd"),
		fareProduct("rail_fare", 4.00, "", ""),
	}
	feed.FareLegRuleList = []gtfs.FareLegRule{
		legRule("", "bus", "bus_fare"),
		legRule("", "bus", "bus_pass"),
		legRule("", "bus", "senior_pass"),
		legRule("", "rail", "rail_fare"),
	}
	c := newTestCoster(t, feed)
	rider := func(categories []string, held ...string) Journey {
		j := journey(
			leg("bus1", "a", "b", at(8, 0), at(8, 20)),
			leg("rail1", "b", "c", at(8, 30), at(8, 50)),
		)
		j.RiderCategoryIDs, j.FareProductIDs = categories, held
		return j
	}
	tcs := []struct {
		name    string
		journey Journey
		amount  float64
		held    [][]string // HeldProductIDs for each fare leg
	}{
		{"no held products", rider(nil), 6.00, [][]string{nil, nil}},
		{"held pass", rider(nil, "bus_pass"), 6.00, [][]string{{"bus_pass"}, nil}},
		{"held pass for another category", rider(nil, "senior_pass"), 6.00, [][]string{nil, nil}},
		{"held pass for the rider's category", rider([]string{"smd"}, "senior_pass"), 5.00, [][]string{{"senior_pass"}, nil}},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			fare, err := c.LowestFare(tc.journey)
			require.NoError(t, err)
			require.NotNil(t, fare)
			assert.Equal(t, tc.amount, fare.Amount.InexactFloat64())
			var held [][]string
			for _, fl := range fare.FareLegs {
				held = append(held, fl.HeldProductIDs)
			}
			assert.Equal(t, tc.held, held)
		})
	}
	t.Run("unknown product", func(t *testing.T) {
		_, err := c.LowestFare(rider(nil, "token_book"))
		assert.ErrorContains(t, err, `unknown fare product "token_book"`)
	})
}

func TestLowestFare_FareLegs(t *testing.T) {
	feed := transferFeed()
	feed.FareTransferRuleList = []gtfs.FareTransferRule{transferRule("bus", "rail", 0, "ab")}
	c := newTestCoster(t, feed)
	j := journey(
		leg("bus1", "a", "b", at(8, 0), at(8, 20)),
		leg("rail1", "b", "c", at(8, 30), at(8, 50)),
	)
	fare, err := c.LowestFare(j)
	require.NoError(t, err)
	require.NotNil(t, fare)
	assert.Equal(t, 2.50, fare.Amount.InexactFloat64())
	assert.Equal(t, "USD", fare.Currency)
	require.Len(t, fare.FareLegs, 2)

	first := fare.FareLegs[0]
	assert.Equal(t, []int{0}, first.LegIndexes)
	assert.Equal(t, "bus", first.LegRule.LegGroupID.Val)
	assert.Equal(t, "bus_fare", first.Product.FareProductID.Val)
	assert.Nil(t, first.TransferRule)
	assert.Equal(t, 2.00, first.Amount.InexactFloat64())

	second := fare.FareLegs[1]
	assert.Equal(t, []int{1}, second.LegIndexes)
	assert.Equal(t, "rail_fare", second.Product.FareProductID.Val)
	require.NotNil(t, second.TransferRule)
	assert.Equal(t, "bus", second.TransferRule.FromLegGroupID.Val)
	assert.Equal(t, "ab", second.TransferProduct.FareProductID.Val)
	assert.Equal(t, 0.50, second.Amount.InexactFloat64())

	t.Run("fares hold copies of the Coster's entities", func(t *testing.T) {
		first.Product.Amount.Val = 0
		assert.Equal(t, map[string]float64{"": 2.50}, fareAmounts(t, c, j))
	})
}

func TestLowestFare_DecimalAmounts(t *testing.T) {
	// Amounts add up exactly, though 2.15 + 4.80 is 6.949999999999999 in float64.
	decimalFeed := func(rules ...gtfs.FareTransferRule) *direct.Reader {
		feed := baseFeed()
		feed.FareProductList = []gtfs.FareProduct{
			fareProduct("bus_fare", 2.15, "", ""),
			fareProduct("rail_fare", 4.80, "", ""),
			fareProduct("ab", 2.50, "", ""),
		}
		feed.FareLegRuleList = []gtfs.FareLegRule{legRule("bus", "bus", "bus_fare"), legRule("rail", "rail", "rail_fare")}
		feed.FareTransferRuleList = rules
		return feed
	}
	j := journey(
		leg("bus1", "a", "b", at(8, 0), at(8, 20)),
		leg("rail1", "b", "c", at(8, 30), at(8, 50)),
	)
	t.Run("sum", func(t *testing.T) {
		fare, err := newTestCoster(t, decimalFeed()).LowestFare(j)
		require.NoError(t, err)
		require.NotNil(t, fare)
		assert.Equal(t, "6.95", fare.Amount.String())
	})
	t.Run("difference", func(t *testing.T) {
		fare, err := newTestCoster(t, decimalFeed(transferRule("bus", "rail", 2, "ab"))).LowestFare(j)
		require.NoError(t, err)
		require.NotNil(t, fare)
		// AB replaces A, so the second fare leg adds 2.50 - 2.15.
		assert.Equal(t, "0.35", fare.FareLegs[1].Amount.String())
		assert.Equal(t, "2.5", fare.Amount.String())
	})
}

func TestLowestFare_Currencies(t *testing.T) {
	currencyFeed := func(railCurrency string) *direct.Reader {
		feed := baseFeed()
		rail := fareProduct("rail_fare", 4.00, "", "")
		rail.Currency = tt.NewCurrency(railCurrency)
		feed.FareProductList = []gtfs.FareProduct{fareProduct("bus_fare", 2.00, "", ""), rail}
		feed.FareLegRuleList = []gtfs.FareLegRule{legRule("", "bus", "bus_fare"), legRule("", "rail", "rail_fare")}
		return feed
	}
	j := journey(
		leg("bus1", "a", "b", at(8, 0), at(8, 20)),
		leg("rail1", "b", "c", at(8, 30), at(8, 50)),
	)
	t.Run("codes in any case", func(t *testing.T) {
		assert.Equal(t, map[string]float64{"": 6.00}, fareAmounts(t, newTestCoster(t, currencyFeed("usd")), j))
	})
	t.Run("more than one currency", func(t *testing.T) {
		_, err := newTestCoster(t, currencyFeed("CAD")).LowestFare(j)
		assert.EqualError(t, err, "fare mixes currencies USD and CAD")
	})
}

func TestLowestFare_Errors(t *testing.T) {
	c := newTestCoster(t, baseFeed())
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
			_, err := c.LowestFare(tc.journey)
			assert.EqualError(t, err, tc.err)
		})
	}
}
