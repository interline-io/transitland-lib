package gql

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

// The trip counterpart of TestStopResolver_ServiceWindowIsPerFeedVersion, and
// the same shape of defect: Route.trips groups its routes by feed version and
// loops, and each pass both resolves the requested date against that feed
// version's own service window and accumulates its rows. Sharing either the
// filter or the destination slice across passes loses results, and which
// results depends on map iteration order — so the same request answers
// differently from one call to the next.
//
// 2018-05-29 falls inside BA's service window; CT has none covering it and
// relocates the date into its fallback week.
func TestRouteResolver_TripsAcrossFeedVersions(t *testing.T) {
	c, _ := newTestClient(t)

	query := func(q string) string {
		var resp map[string]interface{}
		if err := c.Post(q, &resp); err != nil {
			t.Fatal(err)
		}
		return toJson(resp)
	}
	// Resolved rather than hardcoded: these are unstable database ids.
	routeID := func(feed string, gtfsRouteID string) string {
		j := query(`query{routes(where:{feed_onestop_id:"` + feed + `", route_id:"` + gtfsRouteID + `"}){id}}`)
		id := gjson.Get(j, "routes.0.id")
		if !id.Exists() {
			t.Fatalf("fixture route %s:%s not found", feed, gtfsRouteID)
		}
		return id.String()
	}
	inWindow := routeID("BA", "01")
	relocates := routeID("CT", "Bu-130")

	// The trips of one route out of a request covering all the given routes.
	trips := func(want string, ids ...string) string {
		j := query(`query{routes(ids:[` + strings.Join(ids, ",") + `]){
			id trips(limit:5, where:{service_date:"` + serviceWindowDate + `", use_service_window:true}){ trip_id }
		}}`)
		for _, r := range gjson.Get(j, "routes").Array() {
			if r.Get("id").String() == want {
				return r.Get("trips.#.trip_id").Raw
			}
		}
		t.Fatalf("route %s missing from response %s", want, j)
		return ""
	}

	// The answer when nothing else shares the request.
	alone := trips(inWindow, inWindow)
	assert.NotEqual(t, "[]", alone, "fixture route should run on the requested date")

	// The premise: the other route's feed version really does resolve the date
	// differently. Without this the test can keep passing while asserting
	// nothing, if the fixtures' service windows ever drift.
	assert.NotEqual(t, "[]", trips(relocates, relocates),
		"fixture should include a second feed version that returns trips for this date")

	// Asking for both must not change either one's answer, and must not change
	// it differently from one call to the next.
	for i := 0; i < 10; i++ {
		if !assert.JSONEq(t, alone, trips(inWindow, inWindow, relocates), "run %d differs from the single-route query", i) {
			t.FailNow()
		}
	}
}
