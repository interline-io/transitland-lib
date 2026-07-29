package gql

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

// One `stops` selection can span feed versions, and every stop in it shares a
// single filter. Each feed version resolves the requested date against its own
// service window, relocating a date its window excludes into its fallback week;
// that resolution must not follow the shared filter into the next feed version.
//
// 2018-05-29 falls inside BA's service window, so BA answers for the date as
// asked. The other feed versions here all relocate it. The finder visits them in
// map order, so leaking the resolution makes BA's answer depend on whether it
// was visited first, and the same request returns different stop times from one
// call to the next.
func TestStopResolver_ServiceWindowIsPerFeedVersion(t *testing.T) {
	c, _ := newTestClient(t)

	query := func(q string) string {
		var resp map[string]interface{}
		if err := c.Post(q, &resp); err != nil {
			t.Fatal(err)
		}
		return toJson(resp)
	}
	// Resolved rather than hardcoded: these are unstable database ids.
	stopID := func(feed string, stopID string) string {
		j := query(`query{stops(where:{feed_onestop_id:"` + feed + `", stop_id:"` + stopID + `"}){id}}`)
		id := gjson.Get(j, "stops.0.id")
		if !id.Exists() {
			t.Fatalf("fixture stop %s:%s not found", feed, stopID)
		}
		return id.String()
	}
	inWindow := stopID("BA", "MONT")
	// Three more feed versions, none of whose windows cover the date. Only the
	// visit order decides whether a leak shows, so one alone would leave a coin
	// flip; with these the odds of missing it across the runs below are about
	// one in a million.
	var relocating []string
	for _, s := range [][2]string{{"CT", "70011"}, {"HA", "1014"}, {"WMATA", "ENT_A01_C01_E"}} {
		relocating = append(relocating, stopID(s[0], s[1]))
	}

	departures := func(ids ...string) string {
		j := query(`query{stops(ids:[` + strings.Join(ids, ",") + `]){
			id stop_times(where:{date:"2018-05-29", use_service_window:true}){ service_date departure_time }
		}}`)
		// Only the in-window stop's answer is under test.
		for _, s := range gjson.Get(j, "stops").Array() {
			if s.Get("id").String() == inWindow {
				return s.Get("stop_times").Raw
			}
		}
		t.Fatalf("stop %s missing from response %s", inWindow, j)
		return ""
	}

	// The answer when nothing else shares the request.
	alone := departures(inWindow)
	assert.Contains(t, alone, `"service_date":"2018-05-29"`,
		"fixture should answer for the requested date, not a relocated one")

	// Adding stops whose feed versions relocate the date must not change it, and
	// must not change it differently from one call to the next.
	for i := 0; i < 10; i++ {
		got := departures(append([]string{inWindow}, relocating...)...)
		if !assert.JSONEq(t, alone, got, "run %d differs from the single-stop query", i) {
			t.FailNow()
		}
	}
}
