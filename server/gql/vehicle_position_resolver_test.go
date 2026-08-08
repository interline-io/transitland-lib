package gql

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

// baVehiclePositions is the BART realtime fixture; see
// testdata/server/rt/BA-vehicle-positions.json for the vehicles it contains.
func baVehiclePositions() []testconfig.RTJsonFile {
	return []testconfig.RTJsonFile{
		{Feed: "BA", Ftype: "realtime_vehicle_positions", Fname: "BA-vehicle-positions.json"},
	}
}

// Bounding boxes over the fixture. bayArea holds every vehicle; the rest each
// isolate one part of it.
var (
	bayAreaBbox   = hw{"min_lon": -122.50, "min_lat": 37.50, "max_lon": -122.10, "max_lat": 37.90}
	oaklandBbox   = hw{"min_lon": -122.30, "min_lat": 37.79, "max_lon": -122.25, "max_lat": 37.84}
	twelfthStBbox = hw{"min_lon": -122.28, "min_lat": 37.80, "max_lon": -122.26, "max_lat": 37.81}
	powellStBbox  = hw{"min_lon": -122.42, "min_lat": 37.78, "max_lon": -122.40, "max_lat": 37.79}
	sfoBbox       = hw{"min_lon": -122.40, "min_lat": 37.60, "max_lon": -122.38, "max_lat": 37.63}
	nowhereBbox   = hw{"min_lon": -60.0, "min_lat": 10.0, "max_lon": -59.9, "max_lat": 10.1}
)

const vehiclePositionQuery = `
query($where: VehiclePositionFilter!) {
	vehicle_positions(where: $where) {
		id
		rt_feed_onestop_id
		vehicle { id label license_plate }
		trip_descriptor { trip_id route_id direction_id start_time start_date schedule_relationship }
		position
		bearing
		speed
		current_stop_sequence
		stop_id
		current_status
		congestion_level
		occupancy_status
		occupancy_percentage
		timestamp
	}
}`

func TestVehiclePositionResolver(t *testing.T) {
	tcs := []struct {
		name   string
		bbox   map[string]interface{}
		expect []string
	}{
		{
			name:   "whole bay area",
			bbox:   bayAreaBbox,
			expect: []string{"1001", "1002", "1003", "1004", "1005"},
		},
		{
			// 12TH and MCAR are inside; POWL, FTVL and SFO are not.
			name:   "oakland",
			bbox:   oaklandBbox,
			expect: []string{"1001", "1002"},
		},
		{
			// The message names a route and a stop that are not in the
			// schedule; the vehicle is still returned.
			name:   "vehicle with unmatched entities",
			bbox:   sfoBbox,
			expect: []string{"1005"},
		},
		{
			name:   "no match",
			bbox:   nowhereBbox,
			expect: []string{},
		},
	}
	for _, tc := range tcs {
		testRt(t, rtTestCase{
			name:    tc.name,
			query:   vehiclePositionQuery,
			vars:    hw{"where": hw{"bbox": tc.bbox}},
			rtfiles: baVehiclePositions(),
			cb: func(t *testing.T, jj string) {
				var got []string
				for _, v := range gjson.Get(jj, "vehicle_positions.#.vehicle.id").Array() {
					got = append(got, v.String())
				}
				assert.ElementsMatch(t, tc.expect, got)
			},
		})
	}
}

// Results come back in feed and entity id order so a polling client's markers
// stay put, while a limit keeps the freshest vehicles: 1001 and 1002 report a
// timestamp and the rest do not.
func TestVehiclePositionResolver_OrderAndLimit(t *testing.T) {
	q := `query($limit: Int, $where: VehiclePositionFilter!) {
		vehicle_positions(limit: $limit, where: $where) { id }
	}`
	tcs := []struct {
		name   string
		limit  interface{}
		expect []string
	}{
		{name: "no limit", limit: nil, expect: []string{"1001", "1002", "1003", "1004", "1005"}},
		{name: "limit 2", limit: 2, expect: []string{"1001", "1002"}},
		{name: "limit 0", limit: 0, expect: []string{}},
	}
	for _, tc := range tcs {
		testRt(t, rtTestCase{
			name:    tc.name,
			query:   q,
			vars:    hw{"limit": tc.limit, "where": hw{"bbox": bayAreaBbox}},
			rtfiles: baVehiclePositions(),
			cb: func(t *testing.T, jj string) {
				got := []string{}
				for _, v := range gjson.Get(jj, "vehicle_positions.#.id").Array() {
					got = append(got, v.String())
				}
				assert.Equal(t, tc.expect, got)
			},
		})
	}
}

func TestVehiclePositionResolver_Fields(t *testing.T) {
	testRt(t, rtTestCase{
		name:    "all fields",
		query:   vehiclePositionQuery,
		vars:    hw{"where": hw{"bbox": twelfthStBbox}},
		rtfiles: baVehiclePositions(),
		cb: func(t *testing.T, jj string) {
			vps := gjson.Get(jj, "vehicle_positions").Array()
			if len(vps) != 1 {
				t.Fatalf("got %d vehicle positions, expected 1", len(vps))
			}
			vp := vps[0]
			assert.Equal(t, "1001", vp.Get("id").String())
			assert.Equal(t, "BA", vp.Get("rt_feed_onestop_id").String())
			assert.Equal(t, "1001", vp.Get("vehicle.id").String())
			assert.Equal(t, "Antioch Train", vp.Get("vehicle.label").String())
			assert.Equal(t, "BART1001", vp.Get("vehicle.license_plate").String())
			assert.Equal(t, "3210613WKDY", vp.Get("trip_descriptor.trip_id").String())
			assert.Equal(t, "01", vp.Get("trip_descriptor.route_id").String())
			assert.Equal(t, int64(0), vp.Get("trip_descriptor.direction_id").Int())
			assert.Equal(t, "06:13:00", vp.Get("trip_descriptor.start_time").String())
			assert.Equal(t, "2022-09-01", vp.Get("trip_descriptor.start_date").String())
			assert.Equal(t, "SCHEDULED", vp.Get("trip_descriptor.schedule_relationship").String())
			assert.Equal(t, "Point", vp.Get("position.type").String())
			assert.InDelta(t, -122.27145, vp.Get("position.coordinates.0").Float(), 0.00001)
			assert.InDelta(t, 37.80377, vp.Get("position.coordinates.1").Float(), 0.00001)
			assert.InDelta(t, 45.0, vp.Get("bearing").Float(), 0.001)
			assert.InDelta(t, 12.5, vp.Get("speed").Float(), 0.001)
			assert.Equal(t, int64(4), vp.Get("current_stop_sequence").Int())
			assert.Equal(t, "12TH", vp.Get("stop_id").String())
			assert.Equal(t, "STOPPED_AT", vp.Get("current_status").String())
			assert.Equal(t, "RUNNING_SMOOTHLY", vp.Get("congestion_level").String())
			assert.Equal(t, "FEW_SEATS_AVAILABLE", vp.Get("occupancy_status").String())
			assert.Equal(t, int64(40), vp.Get("occupancy_percentage").Int())
			assert.Equal(t, "2022-08-31T23:59:00Z", vp.Get("timestamp").String())
		},
	})
}

func TestVehiclePositionResolver_MatchedEntities(t *testing.T) {
	q := `
	query($where: VehiclePositionFilter!) {
		vehicle_positions(where: $where) {
			vehicle { id }
			trip { trip_id trip_headsign }
			route { route_id route_long_name }
			stop { stop_id stop_name }
		}
	}`
	testRt(t, rtTestCase{
		name:    "matched entities",
		query:   q,
		vars:    hw{"where": hw{"bbox": bayAreaBbox}},
		rtfiles: baVehiclePositions(),
		cb: func(t *testing.T, jj string) {
			byVehicle := map[string]gjson.Result{}
			for _, vp := range gjson.Get(jj, "vehicle_positions").Array() {
				byVehicle[vp.Get("vehicle.id").String()] = vp
			}

			// Trip, route and stop all named in the message.
			vp := byVehicle["1001"]
			assert.Equal(t, "3210613WKDY", vp.Get("trip.trip_id").String())
			assert.Equal(t, "01", vp.Get("route.route_id").String())
			assert.Equal(t, "12TH", vp.Get("stop.stop_id").String())
			assert.NotEmpty(t, vp.Get("stop.stop_name").String())

			// No stop_id in the message.
			vp = byVehicle["1002"]
			assert.Equal(t, "1010400WKDY", vp.Get("trip.trip_id").String())
			assert.Equal(t, "05", vp.Get("route.route_id").String())
			assert.False(t, vp.Get("stop").Exists() && vp.Get("stop").Type != gjson.Null)

			// No trip descriptor at all.
			vp = byVehicle["1004"]
			assert.Equal(t, gjson.Null, vp.Get("trip").Type)
			assert.Equal(t, gjson.Null, vp.Get("route").Type)

			// Ids present in the message but absent from the schedule. The
			// route falls back to the trip's route only when the message names
			// no route at all, so an unmatched route_id stays null.
			vp = byVehicle["1005"]
			assert.Equal(t, gjson.Null, vp.Get("trip").Type)
			assert.Equal(t, gjson.Null, vp.Get("route").Type)
			assert.Equal(t, gjson.Null, vp.Get("stop").Type)
		},
	})
}

// A vehicle whose trip descriptor names only a trip_id still resolves a route,
// through the matched trip.
func TestVehiclePositionResolver_RouteFromTrip(t *testing.T) {
	testRt(t, rtTestCase{
		name: "route resolved from trip",
		query: `query($where: VehiclePositionFilter!) {
			vehicle_positions(where: $where) {
				trip_descriptor { route_id }
				route { route_id }
			}
		}`,
		vars:    hw{"where": hw{"bbox": powellStBbox}},
		rtfiles: baVehiclePositions(),
		cb: func(t *testing.T, jj string) {
			assert.Equal(t, gjson.Null, gjson.Get(jj, "vehicle_positions.0.trip_descriptor.route_id").Type)
			assert.Equal(t, "07", gjson.Get(jj, "vehicle_positions.0.route.route_id").String())
		},
	})
}

// The nested fields are how a client asks for one agency's, route's or trip's
// vehicles; each takes the same filter as the top-level search.
func TestVehiclePositionResolver_Nested(t *testing.T) {
	tcs := []rtTestCase{
		{
			name: "agency vehicle_positions",
			query: `query {
				agencies(where: {onestop_id: "o-9q9-bayarearapidtransit"}) {
					vehicle_positions { vehicle { id } }
				}
			}`,
			cb: func(t *testing.T, jj string) {
				var got []string
				for _, v := range gjson.Get(jj, "agencies.0.vehicle_positions.#.vehicle.id").Array() {
					got = append(got, v.String())
				}
				assert.ElementsMatch(t, []string{"1001", "1002", "1003", "1004", "1005"}, got)
			},
		},
		{
			name: "agency vehicle_positions, bbox",
			query: `query($where: VehiclePositionFilter) {
				agencies(where: {onestop_id: "o-9q9-bayarearapidtransit"}) {
					vehicle_positions(where: $where) { vehicle { id } }
				}
			}`,
			vars: hw{"where": hw{"bbox": oaklandBbox}},
			cb: func(t *testing.T, jj string) {
				var got []string
				for _, v := range gjson.Get(jj, "agencies.0.vehicle_positions.#.vehicle.id").Array() {
					got = append(got, v.String())
				}
				assert.ElementsMatch(t, []string{"1001", "1002"}, got)
			},
		},
		{
			// The two vehicles reporting a timestamp, not the two lowest ids.
			name: "agency vehicle_positions, limit",
			query: `query {
				agencies(where: {onestop_id: "o-9q9-bayarearapidtransit"}) {
					vehicle_positions(limit: 2) { id }
				}
			}`,
			cb: func(t *testing.T, jj string) {
				var got []string
				for _, v := range gjson.Get(jj, "agencies.0.vehicle_positions.#.id").Array() {
					got = append(got, v.String())
				}
				assert.Equal(t, []string{"1001", "1002"}, got)
			},
		},
		{
			name: "route vehicle_positions",
			query: `query {
				routes(where: {feed_onestop_id: "BA", route_id: "01"}) {
					vehicle_positions { vehicle { id } }
				}
			}`,
			cb: func(t *testing.T, jj string) {
				assert.Equal(t, "1001", gjson.Get(jj, "routes.0.vehicle_positions.0.vehicle.id").String())
			},
		},
		{
			// Vehicle 1003's descriptor names only a trip; the route is matched
			// through the trip, as the `route` field does.
			name: "route vehicle_positions, matched by trip",
			query: `query {
				routes(where: {feed_onestop_id: "BA", route_id: "07"}) {
					vehicle_positions { vehicle { id } }
				}
			}`,
			cb: func(t *testing.T, jj string) {
				assert.Equal(t, "1003", gjson.Get(jj, "routes.0.vehicle_positions.0.vehicle.id").String())
			},
		},
		{
			name: "route vehicle_positions, bbox excludes",
			query: `query($where: VehiclePositionFilter) {
				routes(where: {feed_onestop_id: "BA", route_id: "01"}) {
					vehicle_positions(where: $where) { vehicle { id } }
				}
			}`,
			vars: hw{"where": hw{"bbox": nowhereBbox}},
			cb: func(t *testing.T, jj string) {
				assert.Empty(t, gjson.Get(jj, "routes.0.vehicle_positions").Array())
			},
		},
		{
			name: "trip vehicle_position",
			query: `query {
				trips(where: {feed_onestop_id: "BA", trip_id: "3210613WKDY"}) {
					vehicle_position { vehicle { id } }
				}
			}`,
			cb: func(t *testing.T, jj string) {
				assert.Equal(t, "1001", gjson.Get(jj, "trips.0.vehicle_position.vehicle.id").String())
			},
		},
		{
			name: "trip vehicle_position, bbox excludes",
			query: `query($where: VehiclePositionFilter) {
				trips(where: {feed_onestop_id: "BA", trip_id: "3210613WKDY"}) {
					vehicle_position(where: $where) { vehicle { id } }
				}
			}`,
			vars: hw{"where": hw{"bbox": nowhereBbox}},
			cb: func(t *testing.T, jj string) {
				assert.Equal(t, gjson.Null, gjson.Get(jj, "trips.0.vehicle_position").Type)
			},
		},
		{
			name: "trip vehicle_position, no match",
			query: `query {
				trips(where: {feed_onestop_id: "BA", trip_id: "3210652WKDY"}) {
					vehicle_position { vehicle { id } }
				}
			}`,
			cb: func(t *testing.T, jj string) {
				assert.Equal(t, gjson.Null, gjson.Get(jj, "trips.0.vehicle_position").Type)
			},
		},
	}
	for _, tc := range tcs {
		tc.rtfiles = baVehiclePositions()
		testRt(t, tc)
	}
}

// A viewport larger than the server's maximum search radius is refused rather
// than fanned out over every agency on the continent.
func TestVehiclePositionResolver_BboxTooLarge(t *testing.T) {
	c, _ := newTestClientWithOpts(t, testconfig.Options{RTJsons: baVehiclePositions()})
	var resp map[string]interface{}
	err := c.Post(`query { vehicle_positions(where: {bbox: {min_lon: -130.0, min_lat: 20.0, max_lon: -70.0, max_lat: 50.0}}) { id } }`, &resp)
	if err == nil {
		t.Fatal("expected an error for an oversized bbox")
	}
	assert.Contains(t, err.Error(), "bbox too large")
}

func vehiclePositionAgency(t *testing.T, ctx context.Context, cfg model.Config, onestopId string) *model.Agency {
	agencies, err := cfg.Finder.FindAgencies(ctx, nil, nil, nil, &model.AgencyFilter{OnestopID: &onestopId})
	if err != nil {
		t.Fatal(err)
	}
	if len(agencies) != 1 {
		t.Fatalf("got %d agencies for %s, expected 1", len(agencies), onestopId)
	}
	return agencies[0]
}

// vehiclePositionIDs returns the entity ids attributed to an agency, which is
// what the attribution rules below are about.
func vehiclePositionIDs(ctx context.Context, cfg model.Config, agency *model.Agency) []string {
	var ret []string
	for _, ent := range cfg.RTFinder.FindVehiclePositionsForAgency(ctx, agency, nil, nil) {
		ret = append(ret, ent.ID)
	}
	return ret
}

// A realtime feed serving one operator is that operator's, so every vehicle in
// it is attributed even when the message names no route.
func TestVehiclePositions_ExclusiveFeed(t *testing.T) {
	testconfig.ConfigTxRollback(t, testconfig.Options{RTJsons: baVehiclePositions()}, func(cfg model.Config) {
		ctx := model.WithConfig(context.Background(), cfg)
		agency := vehiclePositionAgency(t, ctx, cfg, "o-9q9-bayarearapidtransit")
		assert.ElementsMatch(t,
			[]string{"1001", "1002", "1003", "1004", "1005"},
			vehiclePositionIDs(ctx, cfg, agency),
		)
	})
}

// Regional feeds are associated with dozens of operators. Once a feed is shared
// no operator owns it outright, so only vehicles naming one of the agency's
// routes are attributed to it.
func TestVehiclePositions_SharedFeed(t *testing.T) {
	testconfig.ConfigTxRollback(t, testconfig.Options{RTJsons: baVehiclePositions()}, func(cfg model.Config) {
		ctx := model.WithConfig(context.Background(), cfg)
		agency := vehiclePositionAgency(t, ctx, cfg, "o-9q9-bayarearapidtransit")
		if _, err := cfg.Adapter.DBX().ExecContext(ctx, `
			insert into current_operators_in_feed(feed_id, resolved_onestop_id, resolved_name)
			select id, 'o-9q9-someoneelse', 'Someone Else' from current_feeds where onestop_id = 'BA'`); err != nil {
			t.Fatal(err)
		}
		// 1001 names route 01 and 1002 names route 05; 1005 names route 99,
		// which is not BART's, and the rest name no route at all.
		assert.ElementsMatch(t,
			[]string{"1001", "1002"},
			vehiclePositionIDs(ctx, cfg, agency),
		)
	})
}

// A feed version carrying more than one agency cannot hand every vehicle to
// each of them either.
func TestVehiclePositions_MultiAgencyFeedVersion(t *testing.T) {
	testconfig.ConfigTxRollback(t, testconfig.Options{RTJsons: baVehiclePositions()}, func(cfg model.Config) {
		ctx := model.WithConfig(context.Background(), cfg)
		agency := vehiclePositionAgency(t, ctx, cfg, "o-9q9-bayarearapidtransit")
		if _, err := cfg.Adapter.DBX().ExecContext(ctx, `
			insert into gtfs_agencies(feed_version_id, agency_id, agency_name, agency_url, agency_timezone)
			select feed_version_id, 'OTHER', 'Other Agency', 'http://example.com', agency_timezone
			from gtfs_agencies where id = $1`, agency.ID); err != nil {
			t.Fatal(err)
		}
		assert.ElementsMatch(t,
			[]string{"1001", "1002"},
			vehiclePositionIDs(ctx, cfg, agency),
		)
	})
}
